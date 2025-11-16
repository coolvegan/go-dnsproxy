package server

import (
	"testing"
	"time"

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
