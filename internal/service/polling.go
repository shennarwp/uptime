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
	"net/url"
	"strings"
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

// certNotify30dDays is the threshold (inclusive) at which the poller sends a
// one-time ntfy notification warning that a target's TLS certificate expires
// within this many days.
const certNotify30dDays = 30

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
//   - A separate @daily job runs checkCertificates(), which checks the TLS
//     certificate of every target once a day regardless of its poll interval,
//     sending expiry notifications and persisting the observed expiry.
//
// Concurrency: the entries map is only mutated from the Start goroutine, and
// cron.Add/Remove are goroutine-safe, so no locking is required. pingTarget runs
// on the scheduler's goroutines and only touches the repository and HTTP client.
type PollingService struct {
	repo           *database.TargetRepository
	client         *http.Client
	notifyClient   *http.Client
	ntfyURL        string
	cron           *cron.Cron
	entries        map[int]*cronEntry
	resyncInterval time.Duration
	rootPool       *x509.CertPool
	// now returns the current time; overridden in tests to control the
	// calendar day used for daily certificate reminders.
	now func() time.Time
}

func NewPollingService(repo *database.TargetRepository, ntfyURL string) *PollingService {
	s := &PollingService{
		repo:           repo,
		ntfyURL:        strings.TrimSpace(ntfyURL),
		cron:           cron.New(cron.WithSeconds()),
		entries:        make(map[int]*cronEntry),
		resyncInterval: 30 * time.Second,
		notifyClient:   &http.Client{Timeout: 10 * time.Second},
		now:            time.Now,
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
	logCertExpiry(cs.ServerName, cs.PeerCertificates[0].NotAfter)
	return s.verifyCertChain(cs)
}

// logCertExpiry logs the remaining validity of a certificate. A WARNING is
// emitted when the certificate expires within certExpiryWarningDays (inclusive)
// or has already expired.
func logCertExpiry(serverName string, notAfter time.Time) {
	remaining := time.Until(notAfter)
	expiry := notAfter.Format(time.RFC3339)
	if remaining <= 0 {
		daysExpired := int(-remaining.Hours() / 24)
		log.Printf("[Polling] WARNING TLS cert for %s expired %d days ago (%s)", serverName, daysExpired, expiry)
		return
	}
	daysLeft := int(remaining.Hours() / 24)
	if daysLeft <= certExpiryWarningDays {
		log.Printf("[Polling] WARNING TLS cert for %s expires in %d days (%s)", serverName, daysLeft, expiry)
	} else {
		log.Printf("[Polling] TLS cert for %s expires in %d days (%s)", serverName, daysLeft, expiry)
	}
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

// checkTargetCert reads the TLS certificate expiry for an HTTPS target. It
// reuses the polling HTTP client, whose TLS configuration still performs real
// chain and hostname verification (see verifyCertChain) — only expiry-related
// failures are tolerated so they can be observed and reported. It returns the
// leaf certificate's NotAfter, or ok=false for non-HTTPS targets and failed
// connections.
func (s *PollingService) checkTargetCert(t database.Target) (time.Time, bool) {
	u, err := url.Parse(t.URL)
	if err != nil {
		log.Printf("[Polling] Error parsing URL for target %s (%s): %v", t.Name, t.URL, err)
		return time.Time{}, false
	}
	if u.Scheme != "https" {
		return time.Time{}, false
	}
	resp, err := s.client.Get(t.URL)
	if err != nil {
		log.Printf("[Polling] Error connecting for certificate check to target %s (%s): %v", t.Name, t.URL, err)
		return time.Time{}, false
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("[Polling] Error closing certificate check response for target %s: %v", t.Name, err)
		}
	}()
	if resp.TLS == nil || len(resp.TLS.PeerCertificates) == 0 {
		return time.Time{}, false
	}
	return resp.TLS.PeerCertificates[0].NotAfter.UTC(), true
}

// checkCertificates is the daily certificate expiry sweep. It runs on a fixed
// daily schedule, independent of each target's poll interval, and covers every
// target at once. For each HTTPS target it reads the current certificate, logs
// a warning as its expiry approaches, sends ntfy reminders (once within 30
// days, then daily within 10 days), and persists the observed expiry and
// notification state.
func (s *PollingService) checkCertificates() {
	targets, err := s.repo.GetTargets()
	if err != nil {
		log.Printf("[Polling] Error loading targets for certificate check: %v", err)
		return
	}
	for _, t := range targets {
		notAfter, ok := s.checkTargetCert(t)
		if !ok {
			continue
		}
		logCertExpiry(t.URL, notAfter)
		expiresAt := notAfter.Format(time.RFC3339)
		daysLeft := int(time.Until(notAfter).Hours() / 24)

		notified30dAt, notified10dDate := s.certNotify(t, daysLeft, expiresAt)

		if t.CertExpiresAt != nil && *t.CertExpiresAt == expiresAt &&
			equalStringPtr(t.CertNotified30dAt, notified30dAt) &&
			equalStringPtr(t.CertNotified10dDate, notified10dDate) {
			continue
		}
		if err := s.repo.UpdateCertState(t.ID, expiresAt, notified30dAt, notified10dDate); err != nil {
			log.Printf("[Polling] Error saving cert state for target %s (%s): %v", t.Name, t.URL, err)
		}
	}
}

// certNotify sends ntfy reminders about an upcoming certificate expiry and
// returns the desired persisted notification state. A one-time notification is
// sent when the certificate has certNotify30dDays (30) or fewer days remaining,
// and a daily reminder is sent once per calendar day when it has
// certExpiryWarningDays (10) or fewer. Renewing the certificate resets the
// daily state. It is a no-op when no ntfy URL is configured; send failures
// leave the state unchanged so the reminder is retried on the next poll.
func (s *PollingService) certNotify(t database.Target, daysLeft int, expiresAt string) (notified30dAt, notified10dDate *string) {
	notified30dAt = t.CertNotified30dAt
	notified10dDate = t.CertNotified10dDate

	certChanged := t.CertExpiresAt == nil || *t.CertExpiresAt != expiresAt
	if certChanged {
		notified30dAt = nil
		notified10dDate = nil
	}

	if s.ntfyURL == "" {
		return notified30dAt, notified10dDate
	}

	already30d := t.CertNotified30dAt != nil && *t.CertNotified30dAt == expiresAt
	if daysLeft <= certNotify30dDays && !already30d && s.sendCertNotification(t, daysLeft, expiresAt) {
		notified30dAt = &expiresAt
	}

	now := time.Now
	if s.now != nil {
		now = s.now
	}
	today := now().UTC().Format("2006-01-02")
	if daysLeft <= certExpiryWarningDays &&
		(notified10dDate == nil || *notified10dDate != today) &&
		s.sendCertNotification(t, daysLeft, expiresAt) {
		notified10dDate = &today
	}
	return notified30dAt, notified10dDate
}

// sendCertNotification posts a certificate expiry reminder for the target to
// the configured ntfy URL and reports whether it succeeded.
func (s *PollingService) sendCertNotification(t database.Target, daysLeft int, expiresAt string) bool {
	expiry, err := time.Parse(time.RFC3339, expiresAt)
	expiryDate := expiresAt
	if err == nil {
		expiryDate = expiry.Format("2006-01-02")
	}
	subject := fmt.Sprintf("expires in %d days", daysLeft)
	if daysLeft <= 0 {
		subject = "has expired"
	}
	message := fmt.Sprintf("🔐 Cert %s: %s\nURL: %s\nExpires: %s", subject, t.Name, t.URL, expiryDate)
	if err := s.postNotification("Uptime Monitor: cert expiry", message); err != nil {
		log.Printf("[Polling] Error sending cert expiry notification for target %s (%s): %v", t.Name, t.URL, err)
		return false
	}
	log.Printf("[Polling] Sent cert expiry notification for target %s (%s): cert %s", t.Name, t.URL, subject)
	return true
}

func (s *PollingService) Start(ctx context.Context) {
	if err := s.sync(); err != nil {
		log.Printf("[Polling] Error resyncing targets: %v", err)
		return
	}

	s.cron.Start()

	// Certificate expiry is checked once a day for all targets, on its own
	// schedule, independent of the per-target health check intervals.
	if _, err := s.cron.AddFunc("@daily", s.checkCertificates); err != nil {
		log.Printf("[Polling] Error scheduling certificate check: %v", err)
	}

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

// shouldNotify reports whether a check result warrants an ntfy notification
// given the previous check for the same target. A notification is sent on every
// state transition (up -> down, down -> up) and while a target stays down, but
// not while it stays up. The very first check for a target has no previous
// state and therefore never notifies.
func shouldNotify(prev *database.Check, isUp bool) bool {
	return prev != nil && !(prev.IsUp && isUp)
}

// postNotification posts a plain-text message to the configured ntfy URL. The
// caller decides whether the notification is warranted and logs any failure.
func (s *PollingService) postNotification(title, message string) error {
	req, err := http.NewRequest(http.MethodPost, s.ntfyURL, strings.NewReader(message))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("X-Title", title)
	resp, err := s.notifyClient.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			log.Printf("[Polling] Error closing ntfy response: %v", err)
		}
	}(resp.Body)
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

// notifyTarget posts a status notification for the target to the configured
// ntfy URL. It is a no-op when no URL is configured or the transition does not
// warrant a notification. Failures are logged and never break the poll loop.
func (s *PollingService) notifyTarget(t database.Target, prev *database.Check, cur *database.Check) {
	if s.ntfyURL == "" || !shouldNotify(prev, cur.IsUp) {
		return
	}
	state := "🔴 DOWN"
	if cur.IsUp {
		state = "🟢 UP"
	}
	message := fmt.Sprintf("%s: %s\nURL: %s", state, t.Name, t.URL)
	if cur.StatusCode != nil {
		message += fmt.Sprintf("\nHTTP status: %d", *cur.StatusCode)
	}
	if cur.ErrorMessage != nil && *cur.ErrorMessage != "" {
		message += fmt.Sprintf("\nError: %s", *cur.ErrorMessage)
	}
	if err := s.postNotification("Uptime Monitor", message); err != nil {
		log.Printf("[Polling] Error sending ntfy notification for target %s: %v", t.Name, err)
		return
	}
	log.Printf("[Polling] Sent ntfy notification for target %s: %s (%s)", t.Name, state, t.URL)
}

func (s *PollingService) pingTarget(t database.Target) {
	prev, err := s.repo.GetLastCheckByTargetID(t.ID)
	if err != nil {
		log.Printf("[Polling] Error reading last check for target %s (%s): %v", t.Name, t.URL, err)
	}

	start := time.Now()
	resp, err := s.client.Get(t.URL)
	duration := time.Since(start).Milliseconds()
	durInt := int(duration)

	if err != nil {
		errMsg := err.Error()
		log.Printf("[Polling] Target %s (%s) - Error: %v (took %dms)", t.Name, t.URL, err, duration)
		check := &database.Check{
			TargetID:       t.ID,
			ResponseTimeMS: &durInt,
			IsUp:           false,
			ErrorMessage:   &errMsg,
		}
		_ = s.repo.CreateCheck(check)
		s.notifyTarget(t, prev, check)
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

	if isUp {
		log.Printf("[Polling] Target %s (%s) - Reachable: status %d, took %dms", t.Name, t.URL, statusCode, duration)
	} else {
		log.Printf("[Polling] Target %s (%s) - Unreachable (Server Error): status %d, took %dms", t.Name, t.URL, statusCode, duration)
	}

	check := &database.Check{
		TargetID:       t.ID,
		StatusCode:     &statusCode,
		ResponseTimeMS: &durInt,
		IsUp:           isUp,
	}
	_ = s.repo.CreateCheck(check)
	s.notifyTarget(t, prev, check)
}

func equalStringPtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
