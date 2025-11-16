package dns

import (
	"sync"
	"time"
)

// CacheEntry repräsentiert einen Cache-Eintrag mit Timestamp
type CacheEntry struct {
	IPs       []string
	Timestamp time.Time
}

// Cache ist ein Memory-Cache für DNS-Abfragen
type Cache struct {
	entries  map[string]*CacheEntry
	mu       sync.RWMutex
	ttl      time.Duration
	stopChan chan struct{}
}

// NewCache erstellt einen neuen Cache mit automatischer Reinigung
// ttl: Time-To-Live für Cache-Einträge (z.B. 2 Stunden)
// cleanupInterval: Intervall für die automatische Reinigung (z.B. 5 Minuten)
func NewCache(ttl time.Duration, cleanupInterval time.Duration) *Cache {
	c := &Cache{
		entries:  make(map[string]*CacheEntry),
		ttl:      ttl,
		stopChan: make(chan struct{}),
	}

	// Starte automatische Reinigung in Hintergrund-Goroutine
	go c.cleanupLoop(cleanupInterval)

	return c
}

// Get holt einen Eintrag aus dem Cache
// Gibt nil zurück, wenn der Eintrag nicht existiert oder abgelaufen ist
func (c *Cache) Get(domain string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[domain]
	if !exists {
		return nil
	}

	// Prüfe ob Eintrag abgelaufen ist
	if time.Since(entry.Timestamp) > c.ttl {
		return nil
	}

	return entry.IPs
}

// Set speichert einen Eintrag im Cache
func (c *Cache) Set(domain string, ips []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[domain] = &CacheEntry{
		IPs:       ips,
		Timestamp: time.Now(),
	}
}

// Clear entfernt alle Einträge aus dem Cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*CacheEntry)
}

// Count gibt die Anzahl der Einträge im Cache zurück
func (c *Cache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.entries)
}

// CleanExpired entfernt alle abgelaufenen Einträge
func (c *Cache) CleanExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	removed := 0
	now := time.Now()

	for domain, entry := range c.entries {
		if now.Sub(entry.Timestamp) > c.ttl {
			delete(c.entries, domain)
			removed++
		}
	}

	return removed
}

// cleanupLoop führt die automatische Reinigung in regelmäßigen Abständen durch
func (c *Cache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.CleanExpired()
		case <-c.stopChan:
			return
		}
	}
}

// Stop stoppt die automatische Reinigung
func (c *Cache) Stop() {
	close(c.stopChan)
}

// GetTTL gibt die konfigurierte TTL zurück
func (c *Cache) GetTTL() time.Duration {
	return c.ttl
}
