package main

import (
	"fmt"
	"log"

	"gittea.kittel.dev/go-dnsproxy/internal/dns"
)

func main() {
	// Initialisiere Registry und füge DNS-Server hinzu
	registry := dns.NewRegistry()

	// Füge bekannte öffentliche DNS-Server hinzu
	cloudflare, err := dns.NewServer("Cloudflare", "1.1.1.1", "2606:4700:4700::1111", 53)
	if err != nil {
		log.Fatalf("Fehler beim Erstellen des Cloudflare-Servers: %v", err)
	}
	registry.AddServer(cloudflare)

	google, err := dns.NewServer("Google DNS", "8.8.8.8", "2001:4860:4860::8888", 53)
	if err != nil {
		log.Fatalf("Fehler beim Erstellen des Google-Servers: %v", err)
	}
	registry.AddServer(google)

	quad9, err := dns.NewServer("Quad9", "9.9.9.9", "2620:fe::fe", 53)
	if err != nil {
		log.Fatalf("Fehler beim Erstellen des Quad9-Servers: %v", err)
	}
	registry.AddServer(quad9)

	// Initialisiere Blacklist und füge blockierte Domains hinzu
	blacklist := dns.NewBlacklist()

	// Beispiel: Blockiere bekannte Werbe- und Tracking-Domains
	blacklist.AddDomain("*.doubleclick.net")
	blacklist.AddDomain("*.googlesyndication.com")
	blacklist.AddDomain("*.googleadservices.com")
	blacklist.AddDomain("ads.example.com")

	// Erstelle DNS-Proxy
	proxy := dns.NewProxy(registry, blacklist)

	fmt.Println("=== GO-DNSPROXY Demo ===")
	fmt.Printf("Konfigurierte DNS-Server: %d\n", registry.Count())
	fmt.Printf("Blockierte Domains/Regeln: %d\n\n", blacklist.Count())

	// Test-Domains für Abfragen
	testDomains := []string{
		"heise.de",
		"example.com",
		"google.com",
		"ads.example.com",         // Sollte blockiert sein
		"tracker.doubleclick.net", // Sollte blockiert sein (Wildcard)
	}

	// Führe DNS-Abfragen durch
	for _, domain := range testDomains {
		fmt.Printf("Lookup: %s\n", domain)
		ips, err := proxy.Lookup(domain)
		if err != nil {
			fmt.Printf("  ❌ Fehler: %v\n\n", err)
			continue
		}

		fmt.Printf("  ✅ IP-Adressen:\n")
		for _, ip := range ips {
			fmt.Printf("     - %s\n", ip)
		}
		fmt.Println()
	}

	// Statistik
	fmt.Println("=== Statistik ===")
	fmt.Printf("Aktive DNS-Server: %d\n", registry.Count())
	servers := registry.GetAllServers()
	for _, server := range servers {
		fmt.Printf("  - %s (%s)\n", server.GetName(), server.GetAddress())
	}

	fmt.Printf("\nBlockierte Domain-Regeln: %d\n", blacklist.Count())
	wildcards := blacklist.GetAllWildcards()
	if len(wildcards) > 0 {
		fmt.Println("  Wildcard-Regeln:")
		for _, wc := range wildcards {
			fmt.Printf("    - %s\n", wc)
		}
	}
	domains := blacklist.GetAllDomains()
	if len(domains) > 0 {
		fmt.Println("  Exakte Domains:")
		for _, d := range domains {
			fmt.Printf("    - %s\n", d)
		}
	}
}
