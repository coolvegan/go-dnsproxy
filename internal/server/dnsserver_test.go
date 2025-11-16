package server

import (
	"testing"
	"time"

	"github.com/miekg/dns"
	dnsinternal "gittea.kittel.dev/go-dnsproxy/internal/dns"
)

func TestNewDNSServer(t *testing.T) {
	registry := dnsinternal.NewRegistry()
	blacklist := dnsinternal.NewBlacklist()
	proxy := dnsinternal.NewProxy(registry, blacklist)

	// Test: Erfolgreiche Erstellung
	server, err := NewDNSServer("127.0.0.1:15353", proxy)
	if err != nil {
		t.Fatalf("NewDNSServer() unexpected error: %v", err)
	}
	if server == nil {
		t.Fatal("NewDNSServer() returned nil")
	}
	if server.GetAddr() != "127.0.0.1:15353" {
		t.Errorf("GetAddr() = %s, want 127.0.0.1:15353", server.GetAddr())
	}
}

func TestNewDNSServer_EmptyAddr(t *testing.T) {
	registry := dnsinternal.NewRegistry()
	blacklist := dnsinternal.NewBlacklist()
	proxy := dnsinternal.NewProxy(registry, blacklist)

	_, err := NewDNSServer("", proxy)
	if err == nil {
		t.Error("NewDNSServer() with empty address should return error")
	}
}

func TestNewDNSServer_NilProxy(t *testing.T) {
	_, err := NewDNSServer("127.0.0.1:15353", nil)
	if err == nil {
		t.Error("NewDNSServer() with nil proxy should return error")
	}
}

func TestDNSServer_StartAndStop(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping server test in short mode")
	}

	registry := dnsinternal.NewRegistry()
	blacklist := dnsinternal.NewBlacklist()
	proxy := dnsinternal.NewProxy(registry, blacklist)

	// Nutze einen nicht-privilegierten Port für Tests
	server, err := NewDNSServer("127.0.0.1:15353", proxy)
	if err != nil {
		t.Fatalf("NewDNSServer() failed: %v", err)
	}

	// Test: Start
	err = server.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Kurz warten damit Server hochfährt
	time.Sleep(100 * time.Millisecond)

	// Test: Stop
	err = server.Stop()
	if err != nil {
		t.Errorf("Stop() failed: %v", err)
	}

	// Kurz warten damit Server runterfährt
	time.Sleep(100 * time.Millisecond)
}

func TestDNSServer_DoubleStart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping server test in short mode")
	}

	registry := dnsinternal.NewRegistry()
	blacklist := dnsinternal.NewBlacklist()
	proxy := dnsinternal.NewProxy(registry, blacklist)

	server, _ := NewDNSServer("127.0.0.1:15354", proxy)

	// Erster Start
	err := server.Start()
	if err != nil {
		t.Fatalf("First Start() failed: %v", err)
	}
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	// Zweiter Start sollte fehlschlagen (Port schon belegt)
	server2, _ := NewDNSServer("127.0.0.1:15354", proxy)
	err = server2.Start()
	if err == nil {
		defer server2.Stop()
		t.Error("Second Start() on same port should fail")
	}
}

func TestDNSServer_StopWithoutStart(t *testing.T) {
	registry := dnsinternal.NewRegistry()
	blacklist := dnsinternal.NewBlacklist()
	proxy := dnsinternal.NewProxy(registry, blacklist)

	server, _ := NewDNSServer("127.0.0.1:15355", proxy)

	// Stop ohne Start gibt einen Fehler zurück - das ist ok
	err := server.Stop()
	if err == nil {
		t.Log("Stop() without Start() returns nil (ok)")
	} else {
		t.Logf("Stop() without Start() returns error: %v (also ok)", err)
	}
}

func TestDNSServer_HandleDNSRequest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping DNS query test in short mode")
	}

	// Setup
	registry := dnsinternal.NewRegistry()
	blacklist := dnsinternal.NewBlacklist()
	cache := dnsinternal.NewCache(2*time.Hour, 5*time.Minute)
	defer cache.Stop()

	proxy := dnsinternal.NewProxyWithCache(registry, blacklist, cache)

	// Füge DNS-Server hinzu
	cloudflare, _ := dnsinternal.NewServer("Cloudflare", "1.1.1.1", "", 53)
	registry.AddServer(cloudflare)

	// Starte DNS-Server
	server, err := NewDNSServer("127.0.0.1:15356", proxy)
	if err != nil {
		t.Fatalf("NewDNSServer() failed: %v", err)
	}

	err = server.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	// Teste DNS-Abfrage für A Record
	testDNSQuery(t, "127.0.0.1:15356", "example.com", "A")
}

func TestDNSServer_BlockedDomain(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping DNS query test in short mode")
	}

	// Setup
	registry := dnsinternal.NewRegistry()
	blacklist := dnsinternal.NewBlacklist()
	proxy := dnsinternal.NewProxy(registry, blacklist)

	// Blockiere Domain
	blacklist.AddDomain("blocked.example.com")

	// Füge DNS-Server hinzu (wird nicht gebraucht für blockierte Domains)
	cloudflare, _ := dnsinternal.NewServer("Cloudflare", "1.1.1.1", "", 53)
	registry.AddServer(cloudflare)

	// Starte DNS-Server
	server, err := NewDNSServer("127.0.0.1:15357", proxy)
	if err != nil {
		t.Fatalf("NewDNSServer() failed: %v", err)
	}

	err = server.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	// Teste DNS-Abfrage für blockierte Domain
	// Sollte 0.0.0.0 zurückgeben
	testDNSQueryBlocked(t, "127.0.0.1:15357", "blocked.example.com")
}

// Hilfsfunktion für DNS-Abfragen
func testDNSQuery(t *testing.T, serverAddr, domain, qtype string) {
	c := new(dns.Client)
	m := new(dns.Msg)

	var queryType uint16
	switch qtype {
	case "A":
		queryType = dns.TypeA
	case "AAAA":
		queryType = dns.TypeAAAA
	default:
		t.Fatalf("Unknown query type: %s", qtype)
	}

	m.SetQuestion(dns.Fqdn(domain), queryType)
	m.RecursionDesired = true

	r, _, err := c.Exchange(m, serverAddr)
	if err != nil {
		t.Fatalf("DNS query failed: %v", err)
	}

	if len(r.Answer) == 0 {
		t.Errorf("No answers received for %s", domain)
	}

	t.Logf("Received %d answers for %s (%s)", len(r.Answer), domain, qtype)
	for _, ans := range r.Answer {
		t.Logf("  Answer: %s", ans.String())
	}
}

// Hilfsfunktion für blockierte Domains
func testDNSQueryBlocked(t *testing.T, serverAddr, domain string) {
	c := new(dns.Client)
	m := new(dns.Msg)

	m.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	m.RecursionDesired = true

	r, _, err := c.Exchange(m, serverAddr)
	if err != nil {
		t.Fatalf("DNS query failed: %v", err)
	}

	if len(r.Answer) == 0 {
		t.Error("Expected answer for blocked domain (should return 0.0.0.0)")
		return
	}

	// Prüfe ob 0.0.0.0 zurückgegeben wurde
	foundBlocked := false
	for _, ans := range r.Answer {
		if a, ok := ans.(*dns.A); ok {
			if a.A.String() == "0.0.0.0" {
				foundBlocked = true
				t.Logf("Blocked domain correctly returned: %s", a.A.String())
			}
		}
	}

	if !foundBlocked {
		t.Error("Blocked domain should return 0.0.0.0")
	}
}
