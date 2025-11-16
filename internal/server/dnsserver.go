package server

import (
	"fmt"
	"net"

	"github.com/miekg/dns"
	dnsinternal "gittea.kittel.dev/go-dnsproxy/internal/dns"
)

// DNSServer ist ein echter DNS-Server, der auf Port 53 lauscht
type DNSServer struct {
	proxy  *dnsinternal.Proxy
	server *dns.Server
	addr   string
}

// NewDNSServer erstellt einen neuen DNS-Server
// addr: Adresse zum Lauschen (z.B. ":53" oder "127.0.0.1:5353")
func NewDNSServer(addr string, proxy *dnsinternal.Proxy) (*DNSServer, error) {
	if addr == "" {
		return nil, fmt.Errorf("address cannot be empty")
	}
	if proxy == nil {
		return nil, fmt.Errorf("proxy cannot be nil")
	}

	s := &DNSServer{
		proxy: proxy,
		addr:  addr,
	}

	// Erstelle DNS-Server mit UDP
	s.server = &dns.Server{
		Addr: addr,
		Net:  "udp",
		Handler: dns.HandlerFunc(s.handleDNSRequest),
	}

	return s, nil
}

// Start startet den DNS-Server
func (s *DNSServer) Start() error {
	// Prüfe ob Port verfügbar ist
	conn, err := net.ListenPacket("udp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to bind to %s: %w", s.addr, err)
	}
	conn.Close()

	// Starte Server in Goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != nil {
			// Server wurde gestoppt oder Fehler
			fmt.Printf("DNS Server stopped: %v\n", err)
		}
	}()

	return nil
}

// Stop stoppt den DNS-Server
func (s *DNSServer) Stop() error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown()
}

// handleDNSRequest behandelt eingehende DNS-Anfragen
func (s *DNSServer) handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	msg := new(dns.Msg)
	msg.SetReply(r)
	msg.Authoritative = true

	// Erstmal nur eine einfache Antwort für Tests
	// Wird später mit Proxy-Integration erweitert
	
	w.WriteMsg(msg)
}

// GetAddr gibt die Server-Adresse zurück
func (s *DNSServer) GetAddr() string {
	return s.addr
}
