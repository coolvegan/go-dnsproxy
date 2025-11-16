package dns

import (
	"fmt"
	"sync"
)

// Registry verwaltet eine Liste von DNS-Servern
type Registry struct {
	servers map[string]DNSServer
	mu      sync.RWMutex
}

// NewRegistry erstellt eine neue leere Registry
func NewRegistry() *Registry {
	return &Registry{
		servers: make(map[string]DNSServer),
	}
}

// AddServer fügt einen Server zur Registry hinzu
// Gibt einen Fehler zurück, wenn ein Server mit dem Namen bereits existiert
func (r *Registry) AddServer(server DNSServer) error {
	if server == nil {
		return fmt.Errorf("server cannot be nil")
	}

	name := server.GetName()
	if name == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.servers[name]; exists {
		return fmt.Errorf("server with name '%s' already exists", name)
	}

	r.servers[name] = server
	return nil
}

// RemoveServer entfernt einen Server aus der Registry anhand des Namens
// Gibt einen Fehler zurück, wenn der Server nicht existiert
func (r *Registry) RemoveServer(name string) error {
	if name == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.servers[name]; !exists {
		return fmt.Errorf("server with name '%s' not found", name)
	}

	delete(r.servers, name)
	return nil
}

// GetServer gibt einen Server anhand des Namens zurück
// Gibt nil zurück, wenn der Server nicht existiert
func (r *Registry) GetServer(name string) DNSServer {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.servers[name]
}

// GetAllServers gibt eine Liste aller registrierten Server zurück
func (r *Registry) GetAllServers() []DNSServer {
	r.mu.RLock()
	defer r.mu.RUnlock()

	servers := make([]DNSServer, 0, len(r.servers))
	for _, server := range r.servers {
		servers = append(servers, server)
	}

	return servers
}

// Count gibt die Anzahl der registrierten Server zurück
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.servers)
}

// Clear entfernt alle Server aus der Registry
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.servers = make(map[string]DNSServer)
}
