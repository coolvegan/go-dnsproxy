package dns

import "fmt"

// DNSServer definiert das Interface für DNS-Server
type DNSServer interface {
	GetName() string
	GetIPv4() string
	GetIPv6() string
	GetAddress() string
}

// Server repräsentiert einen DNS-Server mit seinen Eigenschaften
type Server struct {
	Name string
	IPv4 string
	IPv6 string
	Port int
}

// NewServer erstellt eine neue Server-Instanz mit Validierung
func NewServer(name, ipv4, ipv6 string, port int) (*Server, error) {
	if name == "" {
		return nil, fmt.Errorf("server name cannot be empty")
	}
	if ipv4 == "" {
		return nil, fmt.Errorf("IPv4 address cannot be empty")
	}
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("port must be between 1 and 65535")
	}

	return &Server{
		Name: name,
		IPv4: ipv4,
		IPv6: ipv6,
		Port: port,
	}, nil
}

// GetName gibt den Namen des Servers zurück
func (s *Server) GetName() string {
	return s.Name
}

// GetIPv4 gibt die IPv4-Adresse zurück
func (s *Server) GetIPv4() string {
	return s.IPv4
}

// GetIPv6 gibt die IPv6-Adresse zurück (kann leer sein)
func (s *Server) GetIPv6() string {
	return s.IPv6
}

// GetAddress gibt die bevorzugte Adresse zurück
// Nutzt IPv4 wenn vorhanden, sonst IPv6
func (s *Server) GetAddress() string {
	if s.IPv4 != "" {
		return fmt.Sprintf("%s:%d", s.IPv4, s.Port)
	}
	if s.IPv6 != "" {
		return fmt.Sprintf("[%s]:%d", s.IPv6, s.Port)
	}
	return ""
}
