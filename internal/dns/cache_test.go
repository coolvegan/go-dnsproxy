package dns

import (
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	cache := NewCache(2*time.Hour, 5*time.Minute)
	defer cache.Stop()

	if cache == nil {
		t.Fatal("NewCache() returned nil")
	}
	if cache.Count() != 0 {
		t.Errorf("New cache should be empty, got count = %d", cache.Count())
	}
	if cache.GetTTL() != 2*time.Hour {
		t.Errorf("TTL = %v, want %v", cache.GetTTL(), 2*time.Hour)
	}
}

func TestCache_SetAndGet(t *testing.T) {
	cache := NewCache(2*time.Hour, 5*time.Minute)
	defer cache.Stop()

	// Test: Set und Get
	domain := "example.com"
	ips := []string{"1.2.3.4", "5.6.7.8"}

	cache.Set(domain, ips)

	got := cache.Get(domain)
	if got == nil {
		t.Fatal("Get() returned nil for existing entry")
	}
	if len(got) != len(ips) {
		t.Errorf("Get() returned %d IPs, want %d", len(got), len(ips))
	}
	for i, ip := range ips {
		if got[i] != ip {
			t.Errorf("Get()[%d] = %v, want %v", i, got[i], ip)
		}
	}
}

func TestCache_GetNonExistent(t *testing.T) {
	cache := NewCache(2*time.Hour, 5*time.Minute)
	defer cache.Stop()

	got := cache.Get("nonexistent.com")
	if got != nil {
		t.Error("Get() should return nil for non-existent entry")
	}
}

func TestCache_Expiration(t *testing.T) {
	// Kurze TTL für schnellen Test
	cache := NewCache(100*time.Millisecond, 1*time.Second)
	defer cache.Stop()

	domain := "example.com"
	ips := []string{"1.2.3.4"}

	cache.Set(domain, ips)

	// Sollte sofort verfügbar sein
	got := cache.Get(domain)
	if got == nil {
		t.Error("Get() should return entry immediately after Set()")
	}

	// Warte bis TTL abläuft
	time.Sleep(150 * time.Millisecond)

	// Sollte jetzt abgelaufen sein
	got = cache.Get(domain)
	if got != nil {
		t.Error("Get() should return nil for expired entry")
	}
}

func TestCache_Count(t *testing.T) {
	cache := NewCache(2*time.Hour, 5*time.Minute)
	defer cache.Stop()

	if cache.Count() != 0 {
		t.Errorf("Initial Count() = %d, want 0", cache.Count())
	}

	cache.Set("domain1.com", []string{"1.1.1.1"})
	if cache.Count() != 1 {
		t.Errorf("Count() after one set = %d, want 1", cache.Count())
	}

	cache.Set("domain2.com", []string{"2.2.2.2"})
	if cache.Count() != 2 {
		t.Errorf("Count() after two sets = %d, want 2", cache.Count())
	}

	// Überschreibe bestehenden Eintrag
	cache.Set("domain1.com", []string{"3.3.3.3"})
	if cache.Count() != 2 {
		t.Errorf("Count() after overwrite = %d, want 2", cache.Count())
	}
}

func TestCache_Clear(t *testing.T) {
	cache := NewCache(2*time.Hour, 5*time.Minute)
	defer cache.Stop()

	cache.Set("domain1.com", []string{"1.1.1.1"})
	cache.Set("domain2.com", []string{"2.2.2.2"})

	if cache.Count() != 2 {
		t.Errorf("Count() before clear = %d, want 2", cache.Count())
	}

	cache.Clear()

	if cache.Count() != 0 {
		t.Errorf("Count() after clear = %d, want 0", cache.Count())
	}

	got := cache.Get("domain1.com")
	if got != nil {
		t.Error("Get() should return nil after clear")
	}
}

func TestCache_CleanExpired(t *testing.T) {
	// Kurze TTL für schnellen Test
	cache := NewCache(100*time.Millisecond, 1*time.Hour) // Lange cleanup interval, manuell cleanen
	defer cache.Stop()

	// Füge mehrere Einträge hinzu
	cache.Set("domain1.com", []string{"1.1.1.1"})
	cache.Set("domain2.com", []string{"2.2.2.2"})

	if cache.Count() != 2 {
		t.Errorf("Count() = %d, want 2", cache.Count())
	}

	// Warte bis Einträge ablaufen
	time.Sleep(150 * time.Millisecond)

	// Füge neuen Eintrag hinzu (sollte nicht ablaufen)
	cache.Set("domain3.com", []string{"3.3.3.3"})

	// Clean expired
	removed := cache.CleanExpired()
	if removed != 2 {
		t.Errorf("CleanExpired() removed %d entries, want 2", removed)
	}

	if cache.Count() != 1 {
		t.Errorf("Count() after cleanup = %d, want 1", cache.Count())
	}

	// Domain3 sollte noch vorhanden sein
	got := cache.Get("domain3.com")
	if got == nil {
		t.Error("Get(domain3.com) should not be nil")
	}
}

func TestCache_AutomaticCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping automatic cleanup test in short mode")
	}

	// Kurze TTL und kurzes Cleanup-Intervall
	cache := NewCache(200*time.Millisecond, 300*time.Millisecond)
	defer cache.Stop()

	cache.Set("domain1.com", []string{"1.1.1.1"})
	cache.Set("domain2.com", []string{"2.2.2.2"})

	if cache.Count() != 2 {
		t.Errorf("Count() = %d, want 2", cache.Count())
	}

	// Warte auf automatische Reinigung (TTL + Cleanup-Intervall + Buffer)
	time.Sleep(600 * time.Millisecond)

	// Einträge sollten automatisch entfernt worden sein
	if cache.Count() != 0 {
		t.Errorf("Count() after automatic cleanup = %d, want 0", cache.Count())
	}
}

func TestCache_UpdateEntry(t *testing.T) {
	cache := NewCache(2*time.Hour, 5*time.Minute)
	defer cache.Stop()

	domain := "example.com"
	
	// Erste IPs
	ips1 := []string{"1.1.1.1"}
	cache.Set(domain, ips1)

	got := cache.Get(domain)
	if len(got) != 1 || got[0] != "1.1.1.1" {
		t.Error("First Get() failed")
	}

	// Update mit neuen IPs
	ips2 := []string{"2.2.2.2", "3.3.3.3"}
	cache.Set(domain, ips2)

	got = cache.Get(domain)
	if len(got) != 2 {
		t.Errorf("Updated Get() returned %d IPs, want 2", len(got))
	}
	if got[0] != "2.2.2.2" || got[1] != "3.3.3.3" {
		t.Error("Updated Get() returned wrong IPs")
	}
}

func TestCache_EmptyIPsList(t *testing.T) {
	cache := NewCache(2*time.Hour, 5*time.Minute)
	defer cache.Stop()

	// Set mit leerer IP-Liste
	cache.Set("empty.com", []string{})

	got := cache.Get("empty.com")
	if got == nil {
		t.Error("Get() should not return nil for empty IP list")
	}
	if len(got) != 0 {
		t.Errorf("Get() returned %d IPs, want 0", len(got))
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	cache := NewCache(2*time.Hour, 5*time.Minute)
	defer cache.Stop()

	done := make(chan bool)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(idx int) {
			domain := string(rune('a'+idx)) + ".com"
			cache.Set(domain, []string{"1.2.3.4"})
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func(idx int) {
			domain := string(rune('a'+idx)) + ".com"
			cache.Get(domain)
			done <- true
		}(i)
	}

	// Warte auf Completion
	for i := 0; i < 20; i++ {
		<-done
	}

	// Keine Panic = Success
}

func TestCache_Stop(t *testing.T) {
	cache := NewCache(2*time.Hour, 100*time.Millisecond)
	
	// Stop sollte die Cleanup-Goroutine beenden
	cache.Stop()

	// Kurz warten
	time.Sleep(50 * time.Millisecond)

	// Cache sollte noch funktionieren
	cache.Set("test.com", []string{"1.1.1.1"})
	got := cache.Get("test.com")
	if got == nil {
		t.Error("Cache should still work after Stop()")
	}
}
