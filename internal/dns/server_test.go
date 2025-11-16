package dns

import "testing"

func TestNewServer(t *testing.T) {
	tests := []struct {
		name      string
		servName  string
		ipv4      string
		ipv6      string
		port      int
		wantError bool
	}{
		{
			name:      "Valid server with IPv4 only",
			servName:  "Cloudflare",
			ipv4:      "1.1.1.1",
			ipv6:      "",
			port:      53,
			wantError: false,
		},
		{
			name:      "Valid server with IPv4 and IPv6",
			servName:  "Google DNS",
			ipv4:      "8.8.8.8",
			ipv6:      "2001:4860:4860::8888",
			port:      53,
			wantError: false,
		},
		{
			name:      "Empty name",
			servName:  "",
			ipv4:      "1.1.1.1",
			ipv6:      "",
			port:      53,
			wantError: true,
		},
		{
			name:      "Empty IPv4",
			servName:  "Test",
			ipv4:      "",
			ipv6:      "",
			port:      53,
			wantError: true,
		},
		{
			name:      "Invalid port - zero",
			servName:  "Test",
			ipv4:      "1.1.1.1",
			ipv6:      "",
			port:      0,
			wantError: true,
		},
		{
			name:      "Invalid port - negative",
			servName:  "Test",
			ipv4:      "1.1.1.1",
			ipv6:      "",
			port:      -1,
			wantError: true,
		},
		{
			name:      "Invalid port - too high",
			servName:  "Test",
			ipv4:      "1.1.1.1",
			ipv6:      "",
			port:      65536,
			wantError: true,
		},
		{
			name:      "Custom port",
			servName:  "Custom DNS",
			ipv4:      "1.1.1.1",
			ipv6:      "",
			port:      5353,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := NewServer(tt.servName, tt.ipv4, tt.ipv6, tt.port)
			if tt.wantError {
				if err == nil {
					t.Errorf("NewServer() expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("NewServer() unexpected error: %v", err)
				return
			}
			if server.Name != tt.servName {
				t.Errorf("Name = %v, want %v", server.Name, tt.servName)
			}
			if server.IPv4 != tt.ipv4 {
				t.Errorf("IPv4 = %v, want %v", server.IPv4, tt.ipv4)
			}
			if server.IPv6 != tt.ipv6 {
				t.Errorf("IPv6 = %v, want %v", server.IPv6, tt.ipv6)
			}
			if server.Port != tt.port {
				t.Errorf("Port = %v, want %v", server.Port, tt.port)
			}
		})
	}
}

func TestServer_GetMethods(t *testing.T) {
	server, err := NewServer("Test Server", "1.1.1.1", "2606:4700:4700::1111", 53)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if got := server.GetName(); got != "Test Server" {
		t.Errorf("GetName() = %v, want %v", got, "Test Server")
	}

	if got := server.GetIPv4(); got != "1.1.1.1" {
		t.Errorf("GetIPv4() = %v, want %v", got, "1.1.1.1")
	}

	if got := server.GetIPv6(); got != "2606:4700:4700::1111" {
		t.Errorf("GetIPv6() = %v, want %v", got, "2606:4700:4700::1111")
	}
}

func TestServer_GetAddress(t *testing.T) {
	tests := []struct {
		name     string
		servName string
		ipv4     string
		ipv6     string
		port     int
		want     string
	}{
		{
			name:     "IPv4 only",
			servName: "Cloudflare",
			ipv4:     "1.1.1.1",
			ipv6:     "",
			port:     53,
			want:     "1.1.1.1:53",
		},
		{
			name:     "IPv4 with custom port",
			servName: "Custom",
			ipv4:     "8.8.8.8",
			ipv6:     "",
			port:     5353,
			want:     "8.8.8.8:5353",
		},
		{
			name:     "IPv4 and IPv6 - prefers IPv4",
			servName: "Google",
			ipv4:     "8.8.8.8",
			ipv6:     "2001:4860:4860::8888",
			port:     53,
			want:     "8.8.8.8:53",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := NewServer(tt.servName, tt.ipv4, tt.ipv6, tt.port)
			if err != nil {
				t.Fatalf("Failed to create server: %v", err)
			}
			if got := server.GetAddress(); got != tt.want {
				t.Errorf("GetAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServer_ImplementsDNSServerInterface(t *testing.T) {
	var _ DNSServer = (*Server)(nil)
}
