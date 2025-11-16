package dns

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"time"
)

// Proxy ist der DNS-Proxy-Service, der Registry, Blacklist und Cache nutzt
type Proxy struct {
	registry      *Registry
	blacklist     *Blacklist
	cache         *Cache
	timeout       time.Duration
	serverIndex   uint32 // Für Round-Robin
	useRoundRobin bool
}

// NewProxy erstellt einen neuen DNS-Proxy ohne Cache
func NewProxy(registry *Registry, blacklist *Blacklist) *Proxy {
	return &Proxy{
		registry:      registry,
		blacklist:     blacklist,
		cache:         nil,
		timeout:       5 * time.Second,
		useRoundRobin: false,
	}
}

// NewProxyWithCache erstellt einen neuen DNS-Proxy mit Cache
func NewProxyWithCache(registry *Registry, blacklist *Blacklist, cache *Cache) *Proxy {
	return &Proxy{
		registry:      registry,
		blacklist:     blacklist,
		cache:         cache,
		timeout:       5 * time.Second,
		useRoundRobin: true, // Mit Cache nutzen wir Round-Robin
	}
}

// SetTimeout setzt das Timeout für DNS-Abfragen
func (p *Proxy) SetTimeout(timeout time.Duration) {
	p.timeout = timeout
}

// Lookup führt eine DNS-Abfrage für eine Domain durch
// Blockierte Domains geben spezielle IPs zurück (0.0.0.0 / ::)
// Nutzt Cache falls vorhanden, sonst DNS-Server (Round-Robin oder Fallback)
func (p *Proxy) Lookup(domain string) ([]string, error) {
	if domain == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}

	// Prüfe Blacklist - gebe spezielle IPs zurück statt Fehler
	if p.blacklist.IsBlocked(domain) {
		return []string{"0.0.0.0", "::"}, nil
	}

	// Prüfe Cache
	if p.cache != nil {
		if cached := p.cache.Get(domain); cached != nil {
			return cached, nil
		}
	}

	// Hole alle verfügbaren Server
	servers := p.registry.GetAllServers()
	if len(servers) == 0 {
		return nil, fmt.Errorf("no DNS servers configured")
	}

	var ips []string
	var err error

	if p.useRoundRobin {
		// Round-Robin: Versuche Server nacheinander, beginnend mit nächstem
		ips, err = p.lookupRoundRobin(domain, servers)
	} else {
		// Fallback: Versuche alle Server bis einer erfolgreich ist
		ips, err = p.lookupFallback(domain, servers)
	}

	if err != nil {
		return nil, err
	}

	// Speichere erfolgreiches Ergebnis im Cache
	if p.cache != nil && len(ips) > 0 {
		p.cache.Set(domain, ips)
	}

	return ips, nil
}

// lookupRoundRobin versucht Server im Round-Robin-Verfahren
func (p *Proxy) lookupRoundRobin(domain string, servers []DNSServer) ([]string, error) {
	if len(servers) == 0 {
		return nil, fmt.Errorf("no servers available")
	}

	// Hole nächsten Server-Index (atomic für Thread-Safety)
	index := atomic.AddUint32(&p.serverIndex, 1) % uint32(len(servers))

	// Versuche alle Server, beginnend mit dem gewählten
	var lastErr error
	for i := 0; i < len(servers); i++ {
		serverIdx := (int(index) + i) % len(servers)
		ips, err := p.lookupWithServer(domain, servers[serverIdx])
		if err == nil {
			return ips, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("all DNS servers failed, last error: %w", lastErr)
}

// lookupFallback versucht Server nacheinander (alte Methode)
func (p *Proxy) lookupFallback(domain string, servers []DNSServer) ([]string, error) {
	var lastErr error
	for _, server := range servers {
		ips, err := p.lookupWithServer(domain, server)
		if err == nil {
			return ips, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("all DNS servers failed, last error: %w", lastErr)
}

// lookupWithServer führt eine DNS-Abfrage mit einem bestimmten Server durch
func (p *Proxy) lookupWithServer(domain string, server DNSServer) ([]string, error) {
	dnsAddress := server.GetAddress()

	// Erstelle einen benutzerdefinierten Resolver
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: p.timeout,
			}
			return d.DialContext(ctx, "udp", dnsAddress)
		},
	}

	// Führe die DNS-Abfrage aus
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	ipAddrs, err := r.LookupIP(ctx, "ip", domain)
	if err != nil {
		return nil, fmt.Errorf("lookup failed for server %s: %w", server.GetName(), err)
	}

	// Konvertiere zu String-Slice
	var ips []string
	for _, ip := range ipAddrs {
		ips = append(ips, ip.String())
	}

	return ips, nil
}

// GetRegistry gibt die Registry zurück
func (p *Proxy) GetRegistry() *Registry {
	return p.registry
}

// GetBlacklist gibt die Blacklist zurück
func (p *Proxy) GetBlacklist() *Blacklist {
	return p.blacklist
}

// GetCache gibt den Cache zurück
func (p *Proxy) GetCache() *Cache {
	return p.cache
}

// SetRoundRobin aktiviert oder deaktiviert Round-Robin
func (p *Proxy) SetRoundRobin(enabled bool) {
	p.useRoundRobin = enabled
}
