package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"uptime/internal/database"
)

func TestPollingService_PingAndPoll(t *testing.T) {
	// Start test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer ts.Close()

	tmpDir, err := os.MkdirTemp("", "uptime_poll_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	repo := database.NewTargetRepository(db)
	err = repo.CreateTarget(&database.Target{
		Name:     "Polled Target",
		URL:      ts.URL,
		Schedule: "@every 1s",
	})
	if err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	pollingSvc := NewPollingService(repo)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Test pingTarget directly
	targets, err := repo.GetTargets()
	if err != nil {
		t.Fatalf("failed to get targets: %v", err)
	}

	pollingSvc.pingTarget(targets[0])

	checks, err := repo.GetRecentChecksByTargetID(targets[0].ID, 10)
	if err != nil {
		t.Fatalf("failed to get checks: %v", err)
	}
	if len(checks) != 1 {
		t.Fatalf("expected 1 check, got %d", len(checks))
	}
	if !checks[0].IsUp {
		t.Errorf("expected check to be up")
	}

	// Test Start cron with canceled context
	go pollingSvc.Start(ctx)
	<-ctx.Done()
}

func TestPollingService_Resync(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "uptime_poll_resync_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	repo := database.NewTargetRepository(db)
	target := &database.Target{
		Name:     "Resync Target",
		URL:      "http://example.com",
		Schedule: "@every 1h",
	}
	if err := repo.CreateTarget(target); err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	pollingSvc := NewPollingService(repo)

	if err := pollingSvc.sync(); err != nil {
		t.Fatalf("failed to sync: %v", err)
	}
	if entry, ok := pollingSvc.entries[target.ID]; !ok || entry.schedule != "@every 1h" {
		t.Fatalf("expected entry for target %d to be registered with @every 1h", target.ID)
	}
	baseCount := len(pollingSvc.entries) - 1

	updater, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open db for update: %v", err)
	}
	_, err = updater.Exec("UPDATE targets SET schedule = ? WHERE id = ?", "@every 5m", target.ID)
	updater.Close()
	if err != nil {
		t.Fatalf("failed to update schedule: %v", err)
	}

	if err := pollingSvc.sync(); err != nil {
		t.Fatalf("failed to sync: %v", err)
	}
	if len(pollingSvc.entries) != baseCount+1 {
		t.Fatalf("expected %d entries after reschedule, got %d", baseCount+1, len(pollingSvc.entries))
	}
	if entry := pollingSvc.entries[target.ID]; entry == nil || entry.schedule != "@every 5m" {
		t.Fatalf("expected rescheduled entry with @every 5m, got %+v", pollingSvc.entries[target.ID])
	}

	if err := repo.DeleteTarget(target.ID); err != nil {
		t.Fatalf("failed to delete target: %v", err)
	}
	if err := pollingSvc.sync(); err != nil {
		t.Fatalf("failed to sync: %v", err)
	}
	if len(pollingSvc.entries) != baseCount {
		t.Fatalf("expected %d entries after deletion, got %d", baseCount, len(pollingSvc.entries))
	}
}

func TestPollingService_PersistsCertExpiry(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	tmpDir, err := os.MkdirTemp("", "uptime_poll_tls_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	repo := database.NewTargetRepository(db)
	if err := repo.CreateTarget(&database.Target{
		Name:     "TLS Target",
		URL:      ts.URL,
		Schedule: "@every 1m",
	}); err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	svc := NewPollingService(repo)
	targets, err := repo.GetTargets()
	if err != nil {
		t.Fatalf("failed to get targets: %v", err)
	}
	var tlsTarget *database.Target
	for i := range targets {
		if targets[i].URL == ts.URL {
			tlsTarget = &targets[i]
			break
		}
	}
	if tlsTarget == nil {
		t.Fatalf("created target not found in GetTargets result")
	}

	svc.pingTarget(*tlsTarget)

	expected := ts.Certificate().NotAfter.UTC().Format(time.RFC3339)
	stored, err := repo.GetTargetByID(tlsTarget.ID)
	if err != nil {
		t.Fatalf("failed to get target: %v", err)
	}
	if stored.CertExpiresAt == nil {
		t.Fatalf("expected cert_expires_at to be set after TLS check, got nil")
	}
	if *stored.CertExpiresAt != expected {
		t.Errorf("expected cert_expires_at %s, got %s", expected, *stored.CertExpiresAt)
	}
}

func TestPollingService_DoesNotSetCertExpiryForHTTP(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	tmpDir, err := os.MkdirTemp("", "uptime_poll_http_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	repo := database.NewTargetRepository(db)
	if err := repo.CreateTarget(&database.Target{
		Name:     "HTTP Target",
		URL:      ts.URL,
		Schedule: "@every 1m",
	}); err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	svc := NewPollingService(repo)
	targets, err := repo.GetTargets()
	if err != nil {
		t.Fatalf("failed to get targets: %v", err)
	}
	var httpTarget *database.Target
	for i := range targets {
		if targets[i].URL == ts.URL {
			httpTarget = &targets[i]
			break
		}
	}
	if httpTarget == nil {
		t.Fatalf("created target not found in GetTargets result")
	}

	svc.pingTarget(*httpTarget)

	stored, err := repo.GetTargetByID(httpTarget.ID)
	if err != nil {
		t.Fatalf("failed to get target: %v", err)
	}
	if stored.CertExpiresAt != nil {
		t.Errorf("expected cert_expires_at to stay nil for HTTP target, got %v", *stored.CertExpiresAt)
	}
}

func TestPollingService_CheckCert(t *testing.T) {
	svc := &PollingService{}
	day := 24 * time.Hour

	tests := []struct {
		name     string
		notAfter time.Time
		wantLog  string
		wantWarn bool
	}{
		{
			name:     "no peer certificates",
			notAfter: time.Time{},
			wantLog:  "",
		},
		{
			name:     "far from expiry",
			notAfter: time.Now().Add(30*day + 2*time.Hour),
			wantLog:  "expires in 30 days",
		},
		{
			name:     "within warning threshold",
			notAfter: time.Now().Add(5*day + 2*time.Hour),
			wantLog:  "expires in 5 days",
			wantWarn: true,
		},
		{
			name:     "already expired",
			notAfter: time.Now().Add(-3*day - 2*time.Hour),
			wantLog:  "expired 3 days ago",
			wantWarn: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			log.SetOutput(&buf)
			defer log.SetOutput(os.Stderr)

			cs := tls.ConnectionState{}
			if !tc.notAfter.IsZero() {
				cs = tls.ConnectionState{
					ServerName: "example.com",
					PeerCertificates: []*x509.Certificate{
						{NotAfter: tc.notAfter},
					},
				}
			}

			if err := svc.checkCert(cs); err != nil {
				t.Fatalf("checkCert returned an error: %v", err)
			}

			out := buf.String()
			if tc.wantLog != "" && !strings.Contains(out, tc.wantLog) {
				t.Errorf("expected log to contain %q, got %q", tc.wantLog, out)
			}
			if tc.wantWarn && !strings.Contains(out, "WARNING") {
				t.Errorf("expected a WARNING in log, got %q", out)
			}
			if !tc.wantWarn && strings.Contains(out, "WARNING") {
				t.Errorf("did not expect a WARNING, got %q", out)
			}
		})
	}
}
