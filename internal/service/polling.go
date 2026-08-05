package service

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
	"uptime/internal/database"

	"github.com/robfig/cron/v3"
)

type cronEntry struct {
	id       cron.EntryID
	schedule string
}

// certExpiryWarningDays is the threshold (inclusive) at which the poller starts
// logging a WARNING when a target's TLS certificate has this many days or fewer
// remaining (or has already expired).
const certExpiryWarningDays = 10

// PollingService is the background worker that runs the health checks.
//
// It schedules one cron job per target using robfig/cron (in 6-field, seconds
// enabled format) and keeps the scheduler in sync with the database. A periodic
// reconciliation (resyncInterval, default 30s) picks up new, deleted, or
// rescheduled targets without a restart: each registered job's cron.EntryID and
// schedule are tracked in the entries map, so a changed schedule is removed and
// re-added rather than updated in place (robfig/cron has no edit method).
//
// Lifecycle:
//   - Start(ctx) performs an initial sync, starts the scheduler, then loops on a
//     ticker calling sync() every resyncInterval, stopping cleanly on ctx cancel.
//   - sync() reconciles the database with the scheduler in three passes:
//     remove entries for deleted targets, reschedule entries whose schedule
//     changed, and register new targets.
//   - addTarget(t) registers a single cron job whose closure pings that target.
//   - pingTarget(t) performs the actual check: it issues an HTTP GET with a 60s
//     timeout, measures latency, and records a Check row in the database (down
//     with error message on failure, otherwise with status code and response
//     time; isUp = statusCode < 500).
//
// Concurrency: the entries map is only mutated from the Start goroutine, and
// cron.Add/Remove are goroutine-safe, so no locking is required. pingTarget runs
// on the scheduler's goroutines and only touches the repository and HTTP client.
type PollingService struct {
	repo           *database.TargetRepository
	client         *http.Client
	cron           *cron.Cron
	entries        map[int]*cronEntry
	resyncInterval time.Duration
	rootPool       *x509.CertPool
}

func NewPollingService(repo *database.TargetRepository) *PollingService {
	s := &PollingService{
		repo:           repo,
		cron:           cron.New(cron.WithSeconds()),
		entries:        make(map[int]*cronEntry),
		resyncInterval: 30 * time.Second,
	}
	if roots, err := x509.SystemCertPool(); err == nil {
		s.rootPool = roots
	}
	s.client = &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			// InsecureSkipVerify is required so that expired certificates still
			// complete the handshake; checkCert then performs the real chain and
			// hostname verification, tolerating only expiry-related failures so
			// they can be observed and reported.
			//nolint:gosec
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				VerifyConnection:   s.checkCert,
			},
		},
	}
	return s
}

// checkCert logs the remaining validity of the peer's leaf certificate. A
// WARNING is emitted when the certificate expires within certExpiryWarningDays
// (inclusive) or has already expired. It then verifies the certificate chain
// and hostname against the system root store, returning an error (which aborts
// the TLS connection) for anything other than an expired or not-yet-valid
// certificate, so those can still be observed without exposing the poller to
// man-in-the-middle attacks.
func (s *PollingService) checkCert(cs tls.ConnectionState) error {
	if len(cs.PeerCertificates) == 0 {
		return nil
	}
	cert := cs.PeerCertificates[0]
	remaining := time.Until(cert.NotAfter)
	expiry := cert.NotAfter.Format(time.RFC3339)

	if remaining <= 0 {
		daysExpired := int(-remaining.Hours() / 24)
		log.Printf("[Polling] WARNING TLS cert for %s expired %d days ago (%s)", cs.ServerName, daysExpired, expiry)
	} else {
		daysLeft := int(remaining.Hours() / 24)
		if daysLeft <= certExpiryWarningDays {
			log.Printf("[Polling] WARNING TLS cert for %s expires in %d days (%s)", cs.ServerName, daysLeft, expiry)
		} else {
			log.Printf("[Polling] TLS cert for %s expires in %d days (%s)", cs.ServerName, daysLeft, expiry)
		}
	}

	return s.verifyCertChain(cs)
}

// verifyCertChain validates the peer's certificate chain and hostname against
// the system root store. Expired or not-yet-valid certificates are tolerated
// (the expiry is already logged by checkCert); any other failure rejects the
// connection.
func (s *PollingService) verifyCertChain(cs tls.ConnectionState) error {
	if len(cs.PeerCertificates) == 0 {
		return nil
	}
	roots := s.rootPool
	if roots == nil {
		pool, err := x509.SystemCertPool()
		if err != nil {
			return fmt.Errorf("load system root cert pool: %w", err)
		}
		roots = pool
	}
	intermediates := x509.NewCertPool()
	for _, cert := range cs.PeerCertificates[1:] {
		intermediates.AddCert(cert)
	}
	opts := x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		DNSName:       cs.ServerName,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if _, err := cs.PeerCertificates[0].Verify(opts); err != nil {
		var invalid x509.CertificateInvalidError
		if errors.As(err, &invalid) && invalid.Reason == x509.Expired {
			return nil
		}
		return err
	}
	return nil
}

// checkCertExpiry persists the leaf certificate's NotAfter for a target after a
// successful check. It writes to the database only when the observed expiry
// differs from the stored value, and it is a no-op for non-TLS targets.
func (s *PollingService) checkCertExpiry(t database.Target, resp *http.Response) {
	if resp.TLS == nil || len(resp.TLS.PeerCertificates) == 0 {
		return
	}
	expiresAt := resp.TLS.PeerCertificates[0].NotAfter.UTC().Format(time.RFC3339)
	if t.CertExpiresAt != nil && *t.CertExpiresAt == expiresAt {
		return
	}
	if err := s.repo.UpdateCertExpiresAt(t.ID, expiresAt); err != nil {
		log.Printf("[Polling] Error saving cert expiry for target %s (%s): %v", t.Name, t.URL, err)
	}
}

func (s *PollingService) Start(ctx context.Context) {
	if err := s.sync(); err != nil {
		log.Printf("[Polling] Error resyncing targets: %v", err)
		return
	}

	s.cron.Start()

	ticker := time.NewTicker(s.resyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.cron.Stop()
			return
		case <-ticker.C:
			if err := s.sync(); err != nil {
				log.Printf("[Polling] Error resyncing targets: %v", err)
			}
		}
	}
}

func (s *PollingService) addTarget(t database.Target) error {
	id, err := s.cron.AddFunc(t.Schedule, func() {
		s.pingTarget(t)
	})
	if err != nil {
		return err
	}
	s.entries[t.ID] = &cronEntry{id: id, schedule: t.Schedule}
	return nil
}

func (s *PollingService) sync() error {
	targets, err := s.repo.GetTargets()
	if err != nil {
		return err
	}

	valid := make(map[int]database.Target, len(targets))
	for _, t := range targets {
		valid[t.ID] = t
	}

	for id, entry := range s.entries {
		t, ok := valid[id]
		if !ok {
			s.cron.Remove(entry.id)
			delete(s.entries, id)
			log.Printf("[Polling] Removed cron for target %d", id)
			continue
		}
		if entry.schedule != t.Schedule {
			s.cron.Remove(entry.id)
			delete(s.entries, id)
			if err := s.addTarget(t); err != nil {
				log.Printf("[Polling] Error re-adding cron for target %d (schedule %s): %v", t.ID, t.Schedule, err)
			} else {
				log.Printf("[Polling] Rescheduled target %d: %q -> %q", t.ID, entry.schedule, t.Schedule)
			}
		}
	}

	for _, t := range targets {
		if _, ok := s.entries[t.ID]; ok {
			continue
		}
		if err := s.addTarget(t); err != nil {
			log.Printf("[Polling] Error adding cron for target %d (schedule %s): %v", t.ID, t.Schedule, err)
		} else {
			log.Printf("[Polling] Registered cron for target %d (schedule %s)", t.ID, t.Schedule)
		}
	}

	return nil
}

func (s *PollingService) pingTarget(t database.Target) {
	start := time.Now()
	resp, err := s.client.Get(t.URL)
	duration := time.Since(start).Milliseconds()
	durInt := int(duration)

	if err != nil {
		errMsg := err.Error()
		log.Printf("[Polling] Target %s (%s) - Error: %v (took %dms)", t.Name, t.URL, err, duration)
		_ = s.repo.CreateCheck(&database.Check{
			TargetID:       t.ID,
			ResponseTimeMS: &durInt,
			IsUp:           false,
			ErrorMessage:   &errMsg,
		})
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("[Polling] Error closing response body for target %s (%s): %v", t.Name, t.URL, err)
		}
	}(resp.Body)

	statusCode := resp.StatusCode
	isUp := statusCode < 500

	s.checkCertExpiry(t, resp)

	if isUp {
		log.Printf("[Polling] Target %s (%s) - Reachable: status %d, took %dms", t.Name, t.URL, statusCode, duration)
	} else {
		log.Printf("[Polling] Target %s (%s) - Unreachable (Server Error): status %d, took %dms", t.Name, t.URL, statusCode, duration)
	}

	_ = s.repo.CreateCheck(&database.Check{
		TargetID:       t.ID,
		StatusCode:     &statusCode,
		ResponseTimeMS: &durInt,
		IsUp:           isUp,
	})
}
