package dns

import (
	"sync"
	"testing"
)

func TestNewBlacklist(t *testing.T) {
	bl := NewBlacklist()
	if bl == nil {
		t.Fatal("NewBlacklist() returned nil")
	}
	if bl.Count() != 0 {
		t.Errorf("New blacklist should be empty, got count = %d", bl.Count())
	}
}

func TestBlacklist_AddDomain(t *testing.T) {
	bl := NewBlacklist()

	tests := []struct {
		name      string
		domain    string
		wantError bool
	}{
		{
			name:      "Valid domain",
			domain:    "ads.example.com",
			wantError: false,
		},
		{
			name:      "Valid wildcard",
			domain:    "*.ads.com",
			wantError: false,
		},
		{
			name:      "Empty domain",
			domain:    "",
			wantError: true,
		},
		{
			name:      "Invalid wildcard",
			domain:    "*.",
			wantError: true,
		},
		{
			name:      "Domain with spaces",
			domain:    "  example.com  ",
			wantError: false,
		},
		{
			name:      "Uppercase domain",
			domain:    "ADS.EXAMPLE.COM",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := bl.AddDomain(tt.domain)
			if tt.wantError && err == nil {
				t.Error("AddDomain() expected error, got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("AddDomain() unexpected error: %v", err)
			}
		})
	}
}

func TestBlacklist_RemoveDomain(t *testing.T) {
	bl := NewBlacklist()

	// Füge Domains hinzu
	bl.AddDomain("ads.example.com")
	bl.AddDomain("*.tracker.com")

	initialCount := bl.Count()
	if initialCount != 2 {
		t.Errorf("Initial count = %d, want 2", initialCount)
	}

	// Test: Remove reguläre Domain
	err := bl.RemoveDomain("ads.example.com")
	if err != nil {
		t.Errorf("RemoveDomain() unexpected error: %v", err)
	}
	if bl.Count() != 1 {
		t.Errorf("Count after remove = %d, want 1", bl.Count())
	}

	// Test: Remove Wildcard
	err = bl.RemoveDomain("*.tracker.com")
	if err != nil {
		t.Errorf("RemoveDomain() unexpected error: %v", err)
	}
	if bl.Count() != 0 {
		t.Errorf("Count after remove wildcard = %d, want 0", bl.Count())
	}

	// Test: Remove nicht existierende Domain (sollte keinen Fehler geben)
	err = bl.RemoveDomain("nonexistent.com")
	if err != nil {
		t.Errorf("RemoveDomain() for non-existent domain: %v", err)
	}

	// Test: Empty domain
	err = bl.RemoveDomain("")
	if err == nil {
		t.Error("RemoveDomain() expected error for empty domain, got none")
	}
}

func TestBlacklist_IsBlocked(t *testing.T) {
	bl := NewBlacklist()

	// Setup: Füge verschiedene Domains hinzu
	bl.AddDomain("ads.example.com")
	bl.AddDomain("tracker.evil.com")
	bl.AddDomain("*.doubleclick.net")
	bl.AddDomain("*.googlesyndication.com")

	tests := []struct {
		name    string
		domain  string
		blocked bool
	}{
		{
			name:    "Exact match - blocked",
			domain:  "ads.example.com",
			blocked: true,
		},
		{
			name:    "Exact match - not blocked",
			domain:  "good.example.com",
			blocked: false,
		},
		{
			name:    "Wildcard match - subdomain",
			domain:  "stats.doubleclick.net",
			blocked: true,
		},
		{
			name:    "Wildcard match - deep subdomain",
			domain:  "a.b.c.doubleclick.net",
			blocked: true,
		},
		{
			name:    "Wildcard match - exact domain",
			domain:  "doubleclick.net",
			blocked: true,
		},
		{
			name:    "Wildcard no match - different domain",
			domain:  "doubleclick.com",
			blocked: false,
		},
		{
			name:    "Wildcard no match - partial",
			domain:  "notdoubleclick.net",
			blocked: false,
		},
		{
			name:    "Case insensitive - uppercase",
			domain:  "ADS.EXAMPLE.COM",
			blocked: true,
		},
		{
			name:    "Case insensitive - mixed",
			domain:  "AdS.ExAmPlE.cOm",
			blocked: true,
		},
		{
			name:    "Empty domain",
			domain:  "",
			blocked: false,
		},
		{
			name:    "Domain with spaces",
			domain:  "  ads.example.com  ",
			blocked: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bl.IsBlocked(tt.domain)
			if got != tt.blocked {
				t.Errorf("IsBlocked(%q) = %v, want %v", tt.domain, got, tt.blocked)
			}
		})
	}
}

func TestBlacklist_GetAllDomains(t *testing.T) {
	bl := NewBlacklist()

	// Test: Leere Blacklist
	domains := bl.GetAllDomains()
	if len(domains) != 0 {
		t.Errorf("GetAllDomains() for empty blacklist = %d, want 0", len(domains))
	}

	// Test: Mit Domains
	bl.AddDomain("ads.com")
	bl.AddDomain("tracker.com")
	bl.AddDomain("*.wildcard.com") // Sollte nicht in GetAllDomains erscheinen

	domains = bl.GetAllDomains()
	if len(domains) != 2 {
		t.Errorf("GetAllDomains() = %d, want 2", len(domains))
	}

	// Überprüfe Inhalt
	domainMap := make(map[string]bool)
	for _, d := range domains {
		domainMap[d] = true
	}
	if !domainMap["ads.com"] || !domainMap["tracker.com"] {
		t.Error("GetAllDomains() missing expected domains")
	}
}

func TestBlacklist_GetAllWildcards(t *testing.T) {
	bl := NewBlacklist()

	// Test: Leere Blacklist
	wildcards := bl.GetAllWildcards()
	if len(wildcards) != 0 {
		t.Errorf("GetAllWildcards() for empty blacklist = %d, want 0", len(wildcards))
	}

	// Test: Mit Wildcards
	bl.AddDomain("*.ads.com")
	bl.AddDomain("*.tracker.com")
	bl.AddDomain("regular.com") // Sollte nicht in GetAllWildcards erscheinen

	wildcards = bl.GetAllWildcards()
	if len(wildcards) != 2 {
		t.Errorf("GetAllWildcards() = %d, want 2", len(wildcards))
	}

	// Überprüfe Inhalt
	wildcardMap := make(map[string]bool)
	for _, w := range wildcards {
		wildcardMap[w] = true
	}
	if !wildcardMap["*.ads.com"] || !wildcardMap["*.tracker.com"] {
		t.Error("GetAllWildcards() missing expected wildcards")
	}
}

func TestBlacklist_Count(t *testing.T) {
	bl := NewBlacklist()

	if bl.Count() != 0 {
		t.Errorf("Initial Count() = %d, want 0", bl.Count())
	}

	bl.AddDomain("domain1.com")
	if bl.Count() != 1 {
		t.Errorf("Count() after one domain = %d, want 1", bl.Count())
	}

	bl.AddDomain("*.wildcard.com")
	if bl.Count() != 2 {
		t.Errorf("Count() after adding wildcard = %d, want 2", bl.Count())
	}

	bl.RemoveDomain("domain1.com")
	if bl.Count() != 1 {
		t.Errorf("Count() after remove = %d, want 1", bl.Count())
	}
}

func TestBlacklist_Clear(t *testing.T) {
	bl := NewBlacklist()

	bl.AddDomain("domain1.com")
	bl.AddDomain("domain2.com")
	bl.AddDomain("*.wildcard.com")

	if bl.Count() != 3 {
		t.Errorf("Count() before clear = %d, want 3", bl.Count())
	}

	bl.Clear()

	if bl.Count() != 0 {
		t.Errorf("Count() after clear = %d, want 0", bl.Count())
	}

	if bl.IsBlocked("domain1.com") {
		t.Error("IsBlocked() after clear should return false")
	}
}

func TestBlacklist_ConcurrentAccess(t *testing.T) {
	bl := NewBlacklist()
	var wg sync.WaitGroup

	// Concurrent Adds
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if idx%2 == 0 {
				bl.AddDomain(string(rune('a'+idx%26)) + ".com")
			} else {
				bl.AddDomain("*." + string(rune('a'+idx%26)) + ".net")
			}
		}(i)
	}

	wg.Wait()

	// Concurrent Reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bl.IsBlocked("test.com")
			bl.Count()
		}()
	}

	wg.Wait()

	// Verify count (sollte 100 sein, wenn keine Duplikate)
	count := bl.Count()
	if count == 0 {
		t.Error("Count() after concurrent operations should not be 0")
	}
}

func TestBlacklist_DuplicateAddition(t *testing.T) {
	bl := NewBlacklist()

	bl.AddDomain("duplicate.com")
	bl.AddDomain("duplicate.com")
	bl.AddDomain("duplicate.com")

	if bl.Count() != 1 {
		t.Errorf("Count() after duplicate adds = %d, want 1", bl.Count())
	}

	bl.AddDomain("*.wildcard.com")
	bl.AddDomain("*.wildcard.com")

	if bl.Count() != 2 {
		t.Errorf("Count() after duplicate wildcard adds = %d, want 2", bl.Count())
	}
}
