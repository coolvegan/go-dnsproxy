package dns

import (
	"context"
	"fmt"
	"net"
	"time"
)

// Proxy ist der DNS-Proxy-Service, der Registry und Blacklist nutzt
type Proxy struct {
	registry  *Registry
	blacklist *Blacklist
	timeout   time.Duration
}

// NewProxy erstellt einen neuen DNS-Proxy
func NewProxy(registry *Registry, blacklist *Blacklist) *Proxy {
	return &Proxy{
		registry:  registry,
		blacklist: blacklist,
		timeout:   5 * time.Second,
	}
}

// SetTimeout setzt das Timeout für DNS-Abfragen
func (p *Proxy) SetTimeout(timeout time.Duration) {
	p.timeout = timeout
}

// Lookup führt eine DNS-Abfrage für eine Domain durch
// Gibt einen Fehler zurück, wenn die Domain blockiert ist
// Versucht mehrere Server bei Fehlern (Fallback)
func (p *Proxy) Lookup(domain string) ([]string, error) {
	if domain == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}

	// Prüfe Blacklist
	if p.blacklist.IsBlocked(domain) {
		return nil, fmt.Errorf("domain '%s' is blocked", domain)
	}

	// Hole alle verfügbaren Server
	servers := p.registry.GetAllServers()
	if len(servers) == 0 {
		return nil, fmt.Errorf("no DNS servers configured")
	}

	// Versuche jeden Server, bis einer erfolgreich ist
	var lastErr error
	for _, server := range servers {
		ips, err := p.lookupWithServer(domain, server)
		if err == nil {
			return ips, nil
		}
		lastErr = err
	}

	// Alle Server haben fehlgeschlagen
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
