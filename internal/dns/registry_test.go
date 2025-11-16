package dns

import (
	"fmt"
	"sync"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	if registry == nil {
		t.Fatal("NewRegistry() returned nil")
	}
	if registry.Count() != 0 {
		t.Errorf("New registry should be empty, got count = %d", registry.Count())
	}
}

func TestRegistry_AddServer(t *testing.T) {
	registry := NewRegistry()

	server1, _ := NewServer("Cloudflare", "1.1.1.1", "", 53)
	server2, _ := NewServer("Google", "8.8.8.8", "2001:4860:4860::8888", 53)

	// Test: Erfolgreicher Add
	err := registry.AddServer(server1)
	if err != nil {
		t.Errorf("AddServer() unexpected error: %v", err)
	}
	if registry.Count() != 1 {
		t.Errorf("Count() = %d, want 1", registry.Count())
	}

	// Test: Zweiten Server hinzufügen
	err = registry.AddServer(server2)
	if err != nil {
		t.Errorf("AddServer() unexpected error: %v", err)
	}
	if registry.Count() != 2 {
		t.Errorf("Count() = %d, want 2", registry.Count())
	}

	// Test: Duplikat hinzufügen
	err = registry.AddServer(server1)
	if err == nil {
		t.Error("AddServer() expected error for duplicate, got none")
	}
	if registry.Count() != 2 {
		t.Errorf("Count() after duplicate = %d, want 2", registry.Count())
	}

	// Test: Nil Server
	err = registry.AddServer(nil)
	if err == nil {
		t.Error("AddServer() expected error for nil server, got none")
	}
}

func TestRegistry_RemoveServer(t *testing.T) {
	registry := NewRegistry()

	server1, _ := NewServer("Cloudflare", "1.1.1.1", "", 53)
	server2, _ := NewServer("Google", "8.8.8.8", "", 53)

	registry.AddServer(server1)
	registry.AddServer(server2)

	// Test: Erfolgreicher Remove
	err := registry.RemoveServer("Cloudflare")
	if err != nil {
		t.Errorf("RemoveServer() unexpected error: %v", err)
	}
	if registry.Count() != 1 {
		t.Errorf("Count() after remove = %d, want 1", registry.Count())
	}

	// Test: Nicht existierenden Server entfernen
	err = registry.RemoveServer("NonExistent")
	if err == nil {
		t.Error("RemoveServer() expected error for non-existent server, got none")
	}

	// Test: Leeren Namen entfernen
	err = registry.RemoveServer("")
	if err == nil {
		t.Error("RemoveServer() expected error for empty name, got none")
	}
}

func TestRegistry_GetServer(t *testing.T) {
	registry := NewRegistry()

	server1, _ := NewServer("Cloudflare", "1.1.1.1", "", 53)
	registry.AddServer(server1)

	// Test: Existierenden Server abrufen
	got := registry.GetServer("Cloudflare")
	if got == nil {
		t.Fatal("GetServer() returned nil for existing server")
	}
	if got.GetName() != "Cloudflare" {
		t.Errorf("GetServer().GetName() = %v, want Cloudflare", got.GetName())
	}

	// Test: Nicht existierenden Server abrufen
	got = registry.GetServer("NonExistent")
	if got != nil {
		t.Error("GetServer() should return nil for non-existent server")
	}
}

func TestRegistry_GetAllServers(t *testing.T) {
	registry := NewRegistry()

	// Test: Leere Registry
	servers := registry.GetAllServers()
	if len(servers) != 0 {
		t.Errorf("GetAllServers() for empty registry = %d, want 0", len(servers))
	}

	// Test: Mit Servern
	server1, _ := NewServer("Cloudflare", "1.1.1.1", "", 53)
	server2, _ := NewServer("Google", "8.8.8.8", "", 53)
	server3, _ := NewServer("Quad9", "9.9.9.9", "", 53)

	registry.AddServer(server1)
	registry.AddServer(server2)
	registry.AddServer(server3)

	servers = registry.GetAllServers()
	if len(servers) != 3 {
		t.Errorf("GetAllServers() = %d, want 3", len(servers))
	}

	// Überprüfe, dass alle Server vorhanden sind
	names := make(map[string]bool)
	for _, s := range servers {
		names[s.GetName()] = true
	}
	expectedNames := []string{"Cloudflare", "Google", "Quad9"}
	for _, name := range expectedNames {
		if !names[name] {
			t.Errorf("GetAllServers() missing server: %s", name)
		}
	}
}

func TestRegistry_Count(t *testing.T) {
	registry := NewRegistry()

	if registry.Count() != 0 {
		t.Errorf("Initial Count() = %d, want 0", registry.Count())
	}

	server1, _ := NewServer("Server1", "1.1.1.1", "", 53)
	registry.AddServer(server1)

	if registry.Count() != 1 {
		t.Errorf("Count() after one add = %d, want 1", registry.Count())
	}

	server2, _ := NewServer("Server2", "2.2.2.2", "", 53)
	registry.AddServer(server2)

	if registry.Count() != 2 {
		t.Errorf("Count() after two adds = %d, want 2", registry.Count())
	}

	registry.RemoveServer("Server1")

	if registry.Count() != 1 {
		t.Errorf("Count() after remove = %d, want 1", registry.Count())
	}
}

func TestRegistry_Clear(t *testing.T) {
	registry := NewRegistry()

	server1, _ := NewServer("Server1", "1.1.1.1", "", 53)
	server2, _ := NewServer("Server2", "2.2.2.2", "", 53)
	registry.AddServer(server1)
	registry.AddServer(server2)

	if registry.Count() != 2 {
		t.Errorf("Count() before clear = %d, want 2", registry.Count())
	}

	registry.Clear()

	if registry.Count() != 0 {
		t.Errorf("Count() after clear = %d, want 0", registry.Count())
	}

	servers := registry.GetAllServers()
	if len(servers) != 0 {
		t.Errorf("GetAllServers() after clear = %d, want 0", len(servers))
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewRegistry()
	var wg sync.WaitGroup

	// Concurrent Adds
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			server, _ := NewServer(
				fmt.Sprintf("Server%d", idx),
				fmt.Sprintf("1.1.1.%d", idx),
				"",
				53,
			)
			registry.AddServer(server)
		}(i)
	}

	wg.Wait()

	if registry.Count() != 10 {
		t.Errorf("Count() after concurrent adds = %d, want 10", registry.Count())
	}

	// Concurrent Reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			registry.GetAllServers()
		}()
	}

	wg.Wait()
}
