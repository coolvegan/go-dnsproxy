package dns

import (
	"fmt"
	"strings"
	"sync"
)

// Blacklist verwaltet blockierte Domains
type Blacklist struct {
	domains  map[string]bool
	wildcards map[string]bool
	mu       sync.RWMutex
}

// NewBlacklist erstellt eine neue leere Blacklist
func NewBlacklist() *Blacklist {
	return &Blacklist{
		domains:   make(map[string]bool),
		wildcards: make(map[string]bool),
	}
}

// AddDomain fügt eine Domain zur Blacklist hinzu
// Unterstützt Wildcards (z.B. "*.ads.com")
func (b *Blacklist) AddDomain(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	// Normalisiere Domain zu lowercase
	domain = strings.ToLower(strings.TrimSpace(domain))

	b.mu.Lock()
	defer b.mu.Unlock()

	// Prüfe ob es ein Wildcard ist (beginnt mit *.)
	if strings.HasPrefix(domain, "*.") {
		// Entferne *. und speichere als Wildcard
		suffix := domain[2:]
		if suffix == "" {
			return fmt.Errorf("invalid wildcard domain: %s", domain)
		}
		b.wildcards[suffix] = true
	} else {
		b.domains[domain] = true
	}

	return nil
}

// RemoveDomain entfernt eine Domain aus der Blacklist
func (b *Blacklist) RemoveDomain(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	domain = strings.ToLower(strings.TrimSpace(domain))

	b.mu.Lock()
	defer b.mu.Unlock()

	// Prüfe ob es ein Wildcard ist
	if strings.HasPrefix(domain, "*.") {
		suffix := domain[2:]
		delete(b.wildcards, suffix)
	} else {
		delete(b.domains, domain)
	}

	return nil
}

// IsBlocked prüft, ob eine Domain blockiert ist
// Berücksichtigt exakte Matches und Wildcard-Regeln
func (b *Blacklist) IsBlocked(domain string) bool {
	if domain == "" {
		return false
	}

	domain = strings.ToLower(strings.TrimSpace(domain))

	b.mu.RLock()
	defer b.mu.RUnlock()

	// Prüfe exakte Domain
	if b.domains[domain] {
		return true
	}

	// Prüfe Wildcards
	// z.B. "ads.example.com" matched "*.example.com"
	for suffix := range b.wildcards {
		if strings.HasSuffix(domain, "."+suffix) || domain == suffix {
			return true
		}
	}

	return false
}

// GetAllDomains gibt alle blockierten Domains zurück (ohne Wildcards)
func (b *Blacklist) GetAllDomains() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	domains := make([]string, 0, len(b.domains))
	for domain := range b.domains {
		domains = append(domains, domain)
	}

	return domains
}

// GetAllWildcards gibt alle Wildcard-Regeln zurück (mit *. Präfix)
func (b *Blacklist) GetAllWildcards() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	wildcards := make([]string, 0, len(b.wildcards))
	for suffix := range b.wildcards {
		wildcards = append(wildcards, "*."+suffix)
	}

	return wildcards
}

// Count gibt die Gesamtanzahl der Einträge zurück (Domains + Wildcards)
func (b *Blacklist) Count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.domains) + len(b.wildcards)
}

// Clear entfernt alle Einträge aus der Blacklist
func (b *Blacklist) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.domains = make(map[string]bool)
	b.wildcards = make(map[string]bool)
}
