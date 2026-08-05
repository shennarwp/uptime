package service

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"log"
	"math/big"
	"net"
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

// newTLSTestServer starts an HTTPS test server whose certificate chain is signed
// by a test CA. It returns the server, a root pool containing that CA (inject
// into PollingService.rootPool so the poller trusts it), and the leaf cert's
// NotAfter.
func newTLSTestServer(t *testing.T, handler http.Handler) (*httptest.Server, *x509.CertPool, time.Time) {
	t.Helper()
	caCert, caKey := generateTestCA(t)

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate leaf key: %v", err)
	}
	notAfter := time.Now().Add(30 * 24 * time.Hour)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "uptime test server"},
		NotBefore:    time.Now().Add(-24 * time.Hour),
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create leaf cert: %v", err)
	}
	leafCert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("failed to parse leaf cert: %v", err)
	}

	ts := httptest.NewUnstartedServer(handler)
	ts.TLS = &tls.Config{
		Certificates: []tls.Certificate{{
			Certificate: [][]byte{leafCert.Raw, caCert.Raw},
			PrivateKey:  leafKey,
		}},
	}
	ts.StartTLS()

	pool := x509.NewCertPool()
	pool.AddCert(caCert)
	return ts, pool, leafCert.NotAfter
}

func TestPollingService_PersistsCertExpiry(t *testing.T) {
	ts, pool, expected := newTLSTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	svc.rootPool = pool

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

	wantExpiry := expected.UTC().Format(time.RFC3339)
	stored, err := repo.GetTargetByID(tlsTarget.ID)
	if err != nil {
		t.Fatalf("failed to get target: %v", err)
	}
	if stored.CertExpiresAt == nil {
		t.Fatalf("expected cert_expires_at to be set after TLS check, got nil")
	}
	if *stored.CertExpiresAt != wantExpiry {
		t.Errorf("expected cert_expires_at %s, got %s", wantExpiry, *stored.CertExpiresAt)
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

func generateTestCA(t *testing.T) (*x509.Certificate, *ecdsa.PrivateKey) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate CA key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "uptime test CA"},
		NotBefore:             time.Now().Add(-24 * time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("failed to create CA cert: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("failed to parse CA cert: %v", err)
	}
	return cert, key
}

func generateTestLeaf(t *testing.T, ca *x509.Certificate, caKey *ecdsa.PrivateKey, dnsName string, notBefore, notAfter time.Time) *x509.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate leaf key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "uptime test leaf"},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{dnsName},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca, &key.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create leaf cert: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("failed to parse leaf cert: %v", err)
	}
	return cert
}

func TestPollingService_CheckCert(t *testing.T) {
	caCert, caKey := generateTestCA(t)

	pool := x509.NewCertPool()
	pool.AddCert(caCert)
	svc := &PollingService{rootPool: pool}

	day := 24 * time.Hour
	now := time.Now()

	tests := []struct {
		name      string
		dnsName   string
		notBefore time.Time
		notAfter  time.Time
		wantErr   bool
		wantLog   string
		wantWarn  bool
	}{
		{
			name: "no peer certificates",
		},
		{
			name:      "far from expiry",
			dnsName:   "example.com",
			notBefore: now.Add(-24 * time.Hour),
			notAfter:  now.Add(30*day + 2*time.Hour),
			wantLog:   "expires in 30 days",
		},
		{
			name:      "within warning threshold",
			dnsName:   "example.com",
			notBefore: now.Add(-24 * time.Hour),
			notAfter:  now.Add(5*day + 2*time.Hour),
			wantLog:   "expires in 5 days",
			wantWarn:  true,
		},
		{
			name:      "already expired",
			dnsName:   "example.com",
			notBefore: now.Add(-30 * day),
			notAfter:  now.Add(-3*day - 2*time.Hour),
			wantLog:   "expired 3 days ago",
			wantWarn:  true,
		},
		{
			name:      "not yet valid",
			dnsName:   "example.com",
			notBefore: now.Add(day),
			notAfter:  now.Add(30*day + 2*time.Hour),
			wantLog:   "expires in 30 days",
		},
		{
			name:      "hostname mismatch",
			dnsName:   "wrong.example",
			notBefore: now.Add(-24 * time.Hour),
			notAfter:  now.Add(30 * day),
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			log.SetOutput(&buf)
			defer log.SetOutput(os.Stderr)

			var cs tls.ConnectionState
			if tc.dnsName != "" {
				leaf := generateTestLeaf(t, caCert, caKey, tc.dnsName, tc.notBefore, tc.notAfter)
				cs = tls.ConnectionState{
					ServerName:       "example.com",
					PeerCertificates: []*x509.Certificate{leaf, caCert},
				}
			}

			err := svc.checkCert(cs)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected checkCert to return an error, got nil")
				}
				return
			}
			if err != nil {
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

	t.Run("untrusted certificate", func(t *testing.T) {
		otherCA, otherKey := generateTestCA(t)
		leaf := generateTestLeaf(t, otherCA, otherKey, "example.com", now.Add(-24*time.Hour), now.Add(30*day))
		cs := tls.ConnectionState{
			ServerName:       "example.com",
			PeerCertificates: []*x509.Certificate{leaf, otherCA},
		}
		if err := svc.checkCert(cs); err == nil {
			t.Fatalf("expected checkCert to reject an untrusted certificate, got nil")
		}
	})
}
