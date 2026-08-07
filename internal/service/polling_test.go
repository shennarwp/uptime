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
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
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

	pollingSvc := NewPollingService(repo, "")

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

	pollingSvc := NewPollingService(repo, "")

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
	return newTLSTestServerWithExpiry(t, handler, time.Now().Add(30*24*time.Hour))
}

// newTLSTestServerWithExpiry is like newTLSTestServer but with a configurable
// leaf certificate expiry, for exercising certificate expiry notifications.
func newTLSTestServerWithExpiry(t *testing.T, handler http.Handler, notAfter time.Time) (*httptest.Server, *x509.CertPool, time.Time) {
	t.Helper()
	caCert, caKey := generateTestCA(t)

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate leaf key: %v", err)
	}
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

// removeSeededTargets deletes every target whose URL is not in keep. A freshly
// migrated test database ships with seeded targets pointing at real HTTPS
// endpoints, which the daily certificate sweep would otherwise dial.
func removeSeededTargets(t *testing.T, repo *database.TargetRepository, keep ...string) {
	t.Helper()
	targets, err := repo.GetTargets()
	if err != nil {
		t.Fatalf("failed to get targets: %v", err)
	}
	for _, target := range targets {
		kept := false
		for _, u := range keep {
			if target.URL == u {
				kept = true
				break
			}
		}
		if kept {
			continue
		}
		if err := repo.DeleteTarget(target.ID); err != nil {
			t.Fatalf("failed to delete seeded target %d: %v", target.ID, err)
		}
	}
}

func TestPollingService_CheckCertificates_PersistsExpiry(t *testing.T) {
	ts, _, expected := newTLSTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	removeSeededTargets(t, repo, ts.URL)

	svc := NewPollingService(repo, "")
	svc.checkCertificates()

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

	wantExpiry := expected.UTC().Format(time.RFC3339)
	if tlsTarget.CertExpiresAt == nil {
		t.Fatalf("expected cert_expires_at to be set after certificate check, got nil")
	}
	if *tlsTarget.CertExpiresAt != wantExpiry {
		t.Errorf("expected cert_expires_at %s, got %s", wantExpiry, *tlsTarget.CertExpiresAt)
	}
}

func TestPollingService_CheckCertificates_SkipsHTTP(t *testing.T) {
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
	removeSeededTargets(t, repo, ts.URL)

	svc := NewPollingService(repo, "")
	svc.checkCertificates()

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

	if httpTarget.CertExpiresAt != nil {
		t.Errorf("expected cert_expires_at to stay nil for HTTP target, got %v", *httpTarget.CertExpiresAt)
	}
}

func TestShouldNotify(t *testing.T) {
	up := &database.Check{IsUp: true}
	down := &database.Check{IsUp: false}

	tests := []struct {
		name string
		prev *database.Check
		isUp bool
		want bool
	}{
		{name: "no previous state", prev: nil, isUp: true, want: false},
		{name: "no previous state and down", prev: nil, isUp: false, want: false},
		{name: "up to down", prev: up, isUp: false, want: true},
		{name: "down to down", prev: down, isUp: false, want: true},
		{name: "down to up", prev: down, isUp: true, want: true},
		{name: "up to up", prev: up, isUp: true, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldNotify(tc.prev, tc.isUp); got != tc.want {
				t.Errorf("shouldNotify(prevUp=%v, isUp=%v) = %v, want %v", tc.prev != nil && tc.prev.IsUp, tc.isUp, got, tc.want)
			}
		})
	}
}

func TestPollingService_NotifiesOnStateTransition(t *testing.T) {
	var mu sync.Mutex
	var bodies []string
	ntfy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, string(body))
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer ntfy.Close()

	var isDown atomic.Bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isDown.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	tmpDir, err := os.MkdirTemp("", "uptime_poll_ntfy_*")
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
		Name:     "Notify Target",
		URL:      ts.URL,
		Schedule: "@every 1m",
	}); err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	svc := NewPollingService(repo, ntfy.URL)

	targets, err := repo.GetTargets()
	if err != nil {
		t.Fatalf("failed to get targets: %v", err)
	}
	var target *database.Target
	for i := range targets {
		if targets[i].URL == ts.URL {
			target = &targets[i]
			break
		}
	}
	if target == nil {
		t.Fatalf("created target not found in GetTargets result")
	}

	count := func() int {
		mu.Lock()
		defer mu.Unlock()
		return len(bodies)
	}

	svc.pingTarget(*target) // first check (up), no previous state -> no notification
	if got := count(); got != 0 {
		t.Fatalf("expected 0 notifications after first up check, got %d", got)
	}

	isDown.Store(true)
	svc.pingTarget(*target) // up -> down -> notify
	svc.pingTarget(*target) // down -> down -> notify
	if got := count(); got != 2 {
		t.Fatalf("expected 2 notifications while down, got %d", got)
	}

	isDown.Store(false)
	svc.pingTarget(*target) // down -> up -> notify
	svc.pingTarget(*target) // up -> up -> no notification
	if got := count(); got != 3 {
		t.Fatalf("expected 3 notifications after recovery, got %d", got)
	}

	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(bodies[0], "🔴 DOWN") || !strings.Contains(bodies[0], target.URL) {
		t.Errorf("expected down notification to include status and URL, got %q", bodies[0])
	}
	if !strings.Contains(bodies[0], "HTTP status: 500") {
		t.Errorf("expected down notification to include status code, got %q", bodies[0])
	}
	if !strings.Contains(bodies[2], "🟢 UP") || !strings.Contains(bodies[2], target.URL) {
		t.Errorf("expected recovery notification to include status and URL, got %q", bodies[2])
	}
}

func TestNotifyTarget_MessageIncludesError(t *testing.T) {
	var mu sync.Mutex
	var bodies []string
	ntfy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, string(body))
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer ntfy.Close()

	svc := &PollingService{
		ntfyURL:      ntfy.URL,
		notifyClient: &http.Client{Timeout: 10 * time.Second},
	}
	errMsg := "dial tcp: connect: connection refused"
	prevUp := &database.Check{IsUp: true}
	cur := &database.Check{IsUp: false, ErrorMessage: &errMsg}

	svc.notifyTarget(
		database.Target{Name: "Broken", URL: "http://127.0.0.1:1"},
		prevUp,
		cur,
	)

	mu.Lock()
	defer mu.Unlock()
	if len(bodies) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(bodies))
	}
	body := bodies[0]
	if !strings.Contains(body, "🔴 DOWN") {
		t.Errorf("expected 🔴 DOWN in message, got %q", body)
	}
	if !strings.Contains(body, "URL:") {
		t.Errorf("expected URL in message, got %q", body)
	}
	if !strings.Contains(body, errMsg) {
		t.Errorf("expected error info in message, got %q", body)
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

type certNotifyHarness struct {
	svc    *PollingService
	bodies func() []string
	count  func() int
}

// newCertNotifyHarness returns a PollingService wired to a capturing ntfy
// server, for exercising certificate expiry notifications with a controllable
// clock.
func newCertNotifyHarness(t *testing.T, now func() time.Time) certNotifyHarness {
	t.Helper()
	var mu sync.Mutex
	var bodies []string
	ntfy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, string(body))
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(ntfy.Close)
	return certNotifyHarness{
		svc: &PollingService{
			ntfyURL:      ntfy.URL,
			notifyClient: &http.Client{Timeout: 10 * time.Second},
			now:          now,
		},
		bodies: func() []string {
			mu.Lock()
			defer mu.Unlock()
			return append([]string(nil), bodies...)
		},
		count: func() int {
			mu.Lock()
			defer mu.Unlock()
			return len(bodies)
		},
	}
}

func TestCertNotify(t *testing.T) {
	day := 24 * time.Hour
	fixedNow := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	expiresIn20d := fixedNow.Add(20*day + 2*time.Hour).UTC().Format(time.RFC3339)
	expiresIn8d := fixedNow.Add(8*day + 2*time.Hour).UTC().Format(time.RFC3339)
	renewedAt := fixedNow.Add(90*day + 2*time.Hour).UTC().Format(time.RFC3339)

	t.Run("notifies once within 30 days", func(t *testing.T) {
		h := newCertNotifyHarness(t, func() time.Time { return fixedNow })
		target := database.Target{Name: "Cert Target", URL: "https://example.com"}

		n30, n10 := h.svc.certNotify(target, 20, expiresIn20d)
		if got := h.count(); got != 1 {
			t.Fatalf("expected 1 notification, got %d: %q", got, h.bodies())
		}
		if n30 == nil || *n30 != expiresIn20d {
			t.Errorf("expected notified30dAt %s, got %v", expiresIn20d, n30)
		}
		if n10 != nil {
			t.Errorf("expected no daily flag outside 10 days, got %v", n10)
		}
		if b := h.bodies()[0]; !strings.Contains(b, "🔐") || !strings.Contains(b, "expires in 20 days") {
			t.Errorf("unexpected notification body: %q", b)
		}

		// second poll for the same certificate: nothing new
		target.CertExpiresAt = &expiresIn20d
		target.CertNotified30dAt = n30
		n30b, n10b := h.svc.certNotify(target, 20, expiresIn20d)
		if got := h.count(); got != 1 {
			t.Fatalf("expected still 1 notification, got %d", got)
		}
		if n30b == nil || *n30b != expiresIn20d {
			t.Errorf("expected persisted notified30dAt %s, got %v", expiresIn20d, n30b)
		}
		if n10b != nil {
			t.Errorf("expected nil daily flag, got %v", n10b)
		}
	})

	t.Run("notifies daily within 10 days", func(t *testing.T) {
		clock := fixedNow
		h := newCertNotifyHarness(t, func() time.Time { return clock })
		target := database.Target{Name: "Cert Target", URL: "https://example.com"}

		n30, n10 := h.svc.certNotify(target, 8, expiresIn8d)
		if got := h.count(); got != 2 {
			t.Fatalf("expected 30d + daily notification on first poll, got %d: %q", got, h.bodies())
		}
		today := clock.Format("2006-01-02")
		if n10 == nil || *n10 != today {
			t.Errorf("expected daily flag %s, got %v", today, n10)
		}

		// same day: no repeat
		target.CertExpiresAt = &expiresIn8d
		target.CertNotified30dAt = n30
		target.CertNotified10dDate = n10
		_, n10b := h.svc.certNotify(target, 8, expiresIn8d)
		if got := h.count(); got != 2 {
			t.Fatalf("expected no repeat same day, got %d", got)
		}
		if n10b == nil || *n10b != today {
			t.Errorf("expected daily flag to persist %s, got %v", today, n10b)
		}

		// next day: daily reminder again
		clock = clock.Add(24 * time.Hour)
		target.CertNotified10dDate = n10b
		_, n10c := h.svc.certNotify(target, 8, expiresIn8d)
		if got := h.count(); got != 3 {
			t.Fatalf("expected daily reminder on next day, got %d", got)
		}
		nextDay := clock.Format("2006-01-02")
		if n10c == nil || *n10c != nextDay {
			t.Errorf("expected daily flag %s, got %v", nextDay, n10c)
		}
	})

	t.Run("renewed certificate resets notification state", func(t *testing.T) {
		clock := fixedNow
		h := newCertNotifyHarness(t, func() time.Time { return clock })
		yesterday := clock.Add(-24 * time.Hour).Format("2006-01-02")
		target := database.Target{
			Name:                "Cert Target",
			URL:                 "https://example.com",
			CertExpiresAt:       &expiresIn8d,
			CertNotified30dAt:   &expiresIn8d,
			CertNotified10dDate: &yesterday,
		}

		n30, n10 := h.svc.certNotify(target, 60, renewedAt)
		if got := h.count(); got != 0 {
			t.Fatalf("expected no notifications after renewal, got %d", got)
		}
		if n30 != nil || n10 != nil {
			t.Errorf("expected flags cleared after renewal, got 30d=%v 10d=%v", n30, n10)
		}
	})

	t.Run("disabled without ntfy URL", func(t *testing.T) {
		svc := &PollingService{notifyClient: &http.Client{}, now: func() time.Time { return fixedNow }}
		n30, n10 := svc.certNotify(database.Target{Name: "Cert Target", URL: "https://example.com"}, 8, expiresIn8d)
		if n30 != nil || n10 != nil {
			t.Errorf("expected no flags without ntfy URL, got 30d=%v 10d=%v", n30, n10)
		}
	})

	t.Run("expired certificate message", func(t *testing.T) {
		h := newCertNotifyHarness(t, func() time.Time { return fixedNow })
		h.svc.certNotify(database.Target{Name: "Cert Target", URL: "https://example.com"}, -2, expiresIn20d)
		if got := h.count(); got != 2 {
			t.Fatalf("expected 30d + daily notifications for expired cert, got %d", got)
		}
		if !strings.Contains(h.bodies()[0], "has expired") {
			t.Errorf("expected 'has expired' in message, got %q", h.bodies()[0])
		}
	})
}

func TestPollingService_CheckCertificates_NotifiesOnExpiry(t *testing.T) {
	var mu sync.Mutex
	var bodies []string
	ntfy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, string(body))
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer ntfy.Close()

	ts, _, _ := newTLSTestServerWithExpiry(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), time.Now().Add(5*24*time.Hour+2*time.Hour))
	defer ts.Close()

	tmpDir, err := os.MkdirTemp("", "uptime_poll_certnotify_*")
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
		Name:     "Expiring Cert Target",
		URL:      ts.URL,
		Schedule: "@every 1m",
	}); err != nil {
		t.Fatalf("failed to create target: %v", err)
	}
	removeSeededTargets(t, repo, ts.URL)

	svc := NewPollingService(repo, ntfy.URL)

	count := func() int {
		mu.Lock()
		defer mu.Unlock()
		return len(bodies)
	}

	svc.checkCertificates()
	svc.checkCertificates()

	if got := count(); got != 2 {
		t.Fatalf("expected 2 certificate notifications (30d once + daily), got %d: %q", got, bodies)
	}
	mu.Lock()
	for _, b := range bodies {
		if !strings.Contains(b, "🔐") {
			t.Errorf("expected certificate emoji in notification, got %q", b)
		}
	}
	mu.Unlock()

	targets, err := repo.GetTargets()
	if err != nil {
		t.Fatalf("failed to get targets: %v", err)
	}
	var target *database.Target
	for i := range targets {
		if targets[i].URL == ts.URL {
			target = &targets[i]
			break
		}
	}
	if target == nil {
		t.Fatalf("created target not found in GetTargets result")
	}
	if target.CertExpiresAt == nil {
		t.Fatalf("expected cert_expires_at to be persisted, got nil")
	}
	if target.CertNotified30dAt == nil || target.CertNotified10dDate == nil {
		t.Fatalf("expected cert notification flags to be persisted, got 30d=%v 10d=%v", target.CertNotified30dAt, target.CertNotified10dDate)
	}
}
