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

	// Verarbeite jede Frage in der Anfrage
	for _, question := range r.Question {
		answers := s.processQuestion(question)
		msg.Answer = append(msg.Answer, answers...)
	}

	w.WriteMsg(msg)
}

// processQuestion verarbeitet eine DNS-Frage und gibt Antworten zurück
func (s *DNSServer) processQuestion(q dns.Question) []dns.RR {
	var answers []dns.RR

	// Unterstütze nur A (IPv4) und AAAA (IPv6) Records
	if q.Qtype != dns.TypeA && q.Qtype != dns.TypeAAAA {
		return answers
	}

	// Extrahiere Domain-Namen (entferne trailing dot)
	domain := q.Name
	if len(domain) > 0 && domain[len(domain)-1] == '.' {
		domain = domain[:len(domain)-1]
	}

	// Frage Proxy nach IPs
	ips, err := s.proxy.Lookup(domain)
	if err != nil {
		// Fehler bei Lookup - keine Antworten zurückgeben
		return answers
	}

	// Konvertiere IPs zu DNS-Records
	for _, ip := range ips {
		rr := s.createDNSRecord(q.Name, ip, q.Qtype)
		if rr != nil {
			answers = append(answers, rr)
		}
	}

	return answers
}

// createDNSRecord erstellt einen DNS-Record (A oder AAAA) aus einer IP-Adresse
func (s *DNSServer) createDNSRecord(name string, ip string, qtype uint16) dns.RR {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil
	}

	// IPv4 (A Record)
	if parsedIP.To4() != nil && qtype == dns.TypeA {
		return &dns.A{
			Hdr: dns.RR_Header{
				Name:   name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    300, // 5 Minuten TTL
			},
			A: parsedIP.To4(),
		}
	}

	// IPv6 (AAAA Record)
	if parsedIP.To4() == nil && qtype == dns.TypeAAAA {
		return &dns.AAAA{
			Hdr: dns.RR_Header{
				Name:   name,
				Rrtype: dns.TypeAAAA,
				Class:  dns.ClassINET,
				Ttl:    300, // 5 Minuten TTL
			},
			AAAA: parsedIP,
		}
	}

	return nil
}

// GetAddr gibt die Server-Adresse zurück
func (s *DNSServer) GetAddr() string {
	return s.addr
}
