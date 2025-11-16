package dns

import (
	"strings"
	"testing"
	"time"
)

func TestNewProxy(t *testing.T) {
	registry := NewRegistry()
	blacklist := NewBlacklist()

	proxy := NewProxy(registry, blacklist)
	if proxy == nil {
		t.Fatal("NewProxy() returned nil")
	}
	if proxy.GetRegistry() != registry {
		t.Error("GetRegistry() returned wrong registry")
	}
	if proxy.GetBlacklist() != blacklist {
		t.Error("GetBlacklist() returned wrong blacklist")
	}
}

func TestProxy_SetTimeout(t *testing.T) {
	registry := NewRegistry()
	blacklist := NewBlacklist()
	proxy := NewProxy(registry, blacklist)

	customTimeout := 10 * time.Second
	proxy.SetTimeout(customTimeout)

	if proxy.timeout != customTimeout {
		t.Errorf("SetTimeout() failed, got %v, want %v", proxy.timeout, customTimeout)
	}
}

func TestProxy_Lookup_EmptyDomain(t *testing.T) {
	registry := NewRegistry()
	blacklist := NewBlacklist()
	proxy := NewProxy(registry, blacklist)

	_, err := proxy.Lookup("")
	if err == nil {
		t.Error("Lookup() with empty domain should return error")
	}
}

func TestProxy_Lookup_BlockedDomain(t *testing.T) {
	registry := NewRegistry()
	blacklist := NewBlacklist()
	proxy := NewProxy(registry, blacklist)

	// Füge einen Server hinzu
	server, _ := NewServer("Test", "8.8.8.8", "", 53)
	registry.AddServer(server)

	// Blockiere eine Domain
	blacklist.AddDomain("blocked.com")

	_, err := proxy.Lookup("blocked.com")
	if err == nil {
		t.Error("Lookup() should fail for blocked domain")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("Error message should mention 'blocked', got: %v", err)
	}
}

func TestProxy_Lookup_NoServers(t *testing.T) {
	registry := NewRegistry()
	blacklist := NewBlacklist()
	proxy := NewProxy(registry, blacklist)

	_, err := proxy.Lookup("example.com")
	if err == nil {
		t.Error("Lookup() should fail when no servers configured")
	}
	if !strings.Contains(err.Error(), "no DNS servers") {
		t.Errorf("Error should mention 'no DNS servers', got: %v", err)
	}
}

func TestProxy_Lookup_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}

	registry := NewRegistry()
	blacklist := NewBlacklist()
	proxy := NewProxy(registry, blacklist)

	// Füge bekannten öffentlichen DNS-Server hinzu
	server, _ := NewServer("Cloudflare", "1.1.1.1", "", 53)
	registry.AddServer(server)

	// Teste mit einer bekannten Domain
	ips, err := proxy.Lookup("example.com")
	if err != nil {
		t.Errorf("Lookup() unexpected error: %v", err)
	}
	if len(ips) == 0 {
		t.Error("Lookup() should return at least one IP")
	}

	// Validiere, dass IPs zurückgegeben wurden
	for _, ip := range ips {
		if ip == "" {
			t.Error("Lookup() returned empty IP")
		}
	}
}

func TestProxy_Lookup_WithWildcardBlock(t *testing.T) {
	registry := NewRegistry()
	blacklist := NewBlacklist()
	proxy := NewProxy(registry, blacklist)

	// Füge einen Server hinzu
	server, _ := NewServer("Test", "8.8.8.8", "", 53)
	registry.AddServer(server)

	// Blockiere mit Wildcard
	blacklist.AddDomain("*.ads.com")

	// Teste blockierte Subdomain
	_, err := proxy.Lookup("tracker.ads.com")
	if err == nil {
		t.Error("Lookup() should fail for wildcard-blocked domain")
	}

	// Teste nicht blockierte Domain
	_, err = proxy.Lookup("ads.net")
	if err != nil && strings.Contains(err.Error(), "blocked") {
		t.Error("Lookup() should not block similar but different domain")
	}
}

func TestProxy_Lookup_Fallback(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}

	registry := NewRegistry()
	blacklist := NewBlacklist()
	proxy := NewProxy(registry, blacklist)
	proxy.SetTimeout(2 * time.Second)

	// Füge einen ungültigen Server hinzu (sollte fehlschlagen)
	invalidServer, _ := NewServer("Invalid", "192.0.2.1", "", 53)
	registry.AddServer(invalidServer)

	// Füge einen gültigen Server hinzu (sollte als Fallback funktionieren)
	validServer, _ := NewServer("Cloudflare", "1.1.1.1", "", 53)
	registry.AddServer(validServer)

	// Die Abfrage sollte trotzdem erfolgreich sein (Fallback)
	ips, err := proxy.Lookup("example.com")
	if err != nil {
		t.Errorf("Lookup() should succeed with fallback, got error: %v", err)
	}
	if len(ips) == 0 {
		t.Error("Lookup() should return IPs via fallback")
	}
}

func TestProxy_Lookup_AllServersFail(t *testing.T) {
	registry := NewRegistry()
	blacklist := NewBlacklist()
	proxy := NewProxy(registry, blacklist)
	proxy.SetTimeout(1 * time.Second)

	// Füge nur ungültige Server hinzu
	server1, _ := NewServer("Invalid1", "192.0.2.1", "", 53)
	server2, _ := NewServer("Invalid2", "192.0.2.2", "", 53)
	registry.AddServer(server1)
	registry.AddServer(server2)

	_, err := proxy.Lookup("example.com")
	if err == nil {
		t.Error("Lookup() should fail when all servers fail")
	}
	if !strings.Contains(err.Error(), "all DNS servers failed") {
		t.Errorf("Error should mention 'all DNS servers failed', got: %v", err)
	}
}

func TestProxy_Lookup_CaseInsensitiveBlocking(t *testing.T) {
	registry := NewRegistry()
	blacklist := NewBlacklist()
	proxy := NewProxy(registry, blacklist)

	server, _ := NewServer("Test", "8.8.8.8", "", 53)
	registry.AddServer(server)

	// Blockiere lowercase
	blacklist.AddDomain("blocked.com")

	// Teste mit verschiedenen Cases
	testCases := []string{
		"blocked.com",
		"BLOCKED.COM",
		"Blocked.Com",
		"bLoCkEd.CoM",
	}

	for _, domain := range testCases {
		_, err := proxy.Lookup(domain)
		if err == nil {
			t.Errorf("Lookup(%q) should be blocked", domain)
		}
		if !strings.Contains(err.Error(), "blocked") {
			t.Errorf("Error for %q should mention 'blocked', got: %v", domain, err)
		}
	}
}

func TestProxy_GetRegistry(t *testing.T) {
	registry := NewRegistry()
	blacklist := NewBlacklist()
	proxy := NewProxy(registry, blacklist)

	got := proxy.GetRegistry()
	if got != registry {
		t.Error("GetRegistry() returned wrong registry")
	}
}

func TestProxy_GetBlacklist(t *testing.T) {
	registry := NewRegistry()
	blacklist := NewBlacklist()
	proxy := NewProxy(registry, blacklist)

	got := proxy.GetBlacklist()
	if got != blacklist {
		t.Error("GetBlacklist() returned wrong blacklist")
	}
}
