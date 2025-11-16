package dns

import (
	"fmt"
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
	if proxy.GetCache() != nil {
		t.Error("NewProxy() should not have cache")
	}
	if proxy.useRoundRobin {
		t.Error("NewProxy() should not use Round-Robin by default")
	}
}

func TestNewProxyWithCache(t *testing.T) {
	registry := NewRegistry()
	blacklist := NewBlacklist()
	cache := NewCache(2*time.Hour, 5*time.Minute)
	defer cache.Stop()

	proxy := NewProxyWithCache(registry, blacklist, cache)
	if proxy == nil {
		t.Fatal("NewProxyWithCache() returned nil")
	}
	if proxy.GetCache() != cache {
		t.Error("GetCache() returned wrong cache")
	}
	if !proxy.useRoundRobin {
		t.Error("Proxy with cache should use Round-Robin")
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

	// Blockierte Domains sollten jetzt spezielle IPs zurückgeben
	ips, err := proxy.Lookup("blocked.com")
	if err != nil {
		t.Errorf("Lookup() for blocked domain should not error, got: %v", err)
	}
	if len(ips) != 2 {
		t.Errorf("Lookup() for blocked domain should return 2 IPs, got %d", len(ips))
	}
	if ips[0] != "0.0.0.0" || ips[1] != "::" {
		t.Errorf("Lookup() for blocked domain should return ['0.0.0.0', '::'], got %v", ips)
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

	// Teste blockierte Subdomain - sollte jetzt spezielle IPs zurückgeben
	ips, err := proxy.Lookup("tracker.ads.com")
	if err != nil {
		t.Errorf("Lookup() for blocked domain should not error, got: %v", err)
	}
	if len(ips) != 2 || ips[0] != "0.0.0.0" {
		t.Errorf("Lookup() for wildcard-blocked domain should return ['0.0.0.0', '::'], got %v", ips)
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

	// Teste mit verschiedenen Cases - sollte immer blockierte IPs zurückgeben
	testCases := []string{
		"blocked.com",
		"BLOCKED.COM",
		"Blocked.Com",
		"bLoCkEd.CoM",
	}

	for _, domain := range testCases {
		ips, err := proxy.Lookup(domain)
		if err != nil {
			t.Errorf("Lookup(%q) should not error for blocked domain, got: %v", domain, err)
		}
		if len(ips) != 2 || ips[0] != "0.0.0.0" {
			t.Errorf("Lookup(%q) should return blocked IPs, got: %v", domain, ips)
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

func TestProxy_GetCache(t *testing.T) {
	registry := NewRegistry()
	blacklist := NewBlacklist()
	cache := NewCache(2*time.Hour, 5*time.Minute)
	defer cache.Stop()

	proxy := NewProxyWithCache(registry, blacklist, cache)
	got := proxy.GetCache()
	if got != cache {
		t.Error("GetCache() returned wrong cache")
	}

	// Proxy ohne Cache
	proxyNoCache := NewProxy(registry, blacklist)
	if proxyNoCache.GetCache() != nil {
		t.Error("GetCache() should return nil for proxy without cache")
	}
}

func TestProxy_Lookup_WithCache(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}

	registry := NewRegistry()
	blacklist := NewBlacklist()
	cache := NewCache(2*time.Hour, 5*time.Minute)
	defer cache.Stop()

	proxy := NewProxyWithCache(registry, blacklist, cache)

	server, _ := NewServer("Cloudflare", "1.1.1.1", "", 53)
	registry.AddServer(server)

	domain := "example.com"

	// Erste Abfrage - sollte DNS nutzen und cachen
	ips1, err := proxy.Lookup(domain)
	if err != nil {
		t.Fatalf("First Lookup() failed: %v", err)
	}
	if len(ips1) == 0 {
		t.Error("First Lookup() should return IPs")
	}

	// Prüfe ob im Cache
	cached := cache.Get(domain)
	if cached == nil {
		t.Error("Domain should be cached after first lookup")
	}

	// Zweite Abfrage - sollte aus Cache kommen
	ips2, err := proxy.Lookup(domain)
	if err != nil {
		t.Fatalf("Second Lookup() failed: %v", err)
	}

	// Sollte identische IPs sein
	if len(ips1) != len(ips2) {
		t.Errorf("Cached lookup returned different number of IPs: %d vs %d", len(ips1), len(ips2))
	}
}

func TestProxy_Lookup_CacheHit(t *testing.T) {
	registry := NewRegistry()
	blacklist := NewBlacklist()
	cache := NewCache(2*time.Hour, 5*time.Minute)
	defer cache.Stop()

	proxy := NewProxyWithCache(registry, blacklist, cache)

	// Kein Server nötig, wenn Cache-Hit
	domain := "cached.example.com"
	expectedIPs := []string{"1.2.3.4", "5.6.7.8"}

	// Setze direkt in Cache
	cache.Set(domain, expectedIPs)

	// Lookup sollte aus Cache kommen, ohne DNS-Server
	ips, err := proxy.Lookup(domain)
	if err != nil {
		t.Errorf("Lookup() failed: %v", err)
	}
	if len(ips) != len(expectedIPs) {
		t.Errorf("Cached Lookup() returned %d IPs, want %d", len(ips), len(expectedIPs))
	}
	for i, ip := range expectedIPs {
		if ips[i] != ip {
			t.Errorf("Cached IP[%d] = %v, want %v", i, ips[i], ip)
		}
	}
}

func TestProxy_SetRoundRobin(t *testing.T) {
	registry := NewRegistry()
	blacklist := NewBlacklist()
	proxy := NewProxy(registry, blacklist)

	if proxy.useRoundRobin {
		t.Error("Default proxy should not use Round-Robin")
	}

	proxy.SetRoundRobin(true)
	if !proxy.useRoundRobin {
		t.Error("SetRoundRobin(true) failed")
	}

	proxy.SetRoundRobin(false)
	if proxy.useRoundRobin {
		t.Error("SetRoundRobin(false) failed")
	}
}

func TestProxy_RoundRobin(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}

	registry := NewRegistry()
	blacklist := NewBlacklist()
	cache := NewCache(2*time.Hour, 5*time.Minute)
	defer cache.Stop()

	proxy := NewProxyWithCache(registry, blacklist, cache)

	// Füge mehrere Server hinzu
	server1, _ := NewServer("Cloudflare", "1.1.1.1", "", 53)
	server2, _ := NewServer("Google", "8.8.8.8", "", 53)
	registry.AddServer(server1)
	registry.AddServer(server2)

	// Mehrere Lookups sollten Round-Robin nutzen
	for i := 0; i < 5; i++ {
		domain := fmt.Sprintf("test%d.example.com", i)
		_, err := proxy.Lookup(domain)
		if err != nil {
			t.Logf("Lookup %d failed: %v (expected for some test domains)", i, err)
		}
	}

	// Server-Index sollte sich geändert haben
	if proxy.serverIndex == 0 {
		t.Error("Round-Robin should have incremented serverIndex")
	}
}
