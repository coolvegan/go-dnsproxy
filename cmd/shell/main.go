package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gittea.kittel.dev/go-dnsproxy/internal/dns"
	"gittea.kittel.dev/go-dnsproxy/internal/server"
)

func main() {
	fmt.Println("=== GO-DNSPROXY - DNS Server with Blacklist & Cache ===")
	fmt.Println()

	// Initialisiere Registry und füge DNS-Server hinzu
	registry := dns.NewRegistry()

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

	// Initialisiere Blacklist
	blacklist := dns.NewBlacklist()

	// Lade externe Hosts-Datei (Steven Black)
	// Für Demo nutzen wir die kleinste Variante
	fmt.Println("📥 Lade externe Blacklist von GitHub...")
	hostsURL := "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts"
	added, err := blacklist.LoadFromURL(hostsURL)
	if err != nil {
		log.Printf("⚠️  Warnung: Konnte externe Blacklist nicht laden: %v", err)
		log.Println("   Fahre mit manuellen Regeln fort...")
		// Fallback zu manuellen Regeln
		blacklist.AddDomain("*.doubleclick.net")
		blacklist.AddDomain("*.googlesyndication.com")
		blacklist.AddDomain("*.googleadservices.com")
		blacklist.AddDomain("*.google-analytics.com")
	} else {
		fmt.Printf("✅ %d Domains von externer Blacklist geladen\n\n", added)
	}

	// Füge zusätzliche manuelle Regeln hinzu
	blacklist.AddDomain("ads.example.com")
	blacklist.AddDomain("tracker.example.com")

	// Initialisiere Cache (2 Stunden TTL, 5 Minuten Cleanup)
	cache := dns.NewCache(2*time.Hour, 5*time.Minute)
	defer cache.Stop()

	// Erstelle Proxy mit Cache und Round-Robin
	proxy := dns.NewProxyWithCache(registry, blacklist, cache)

	// Konfiguration ausgeben
	fmt.Printf("📋 Konfiguration:\n")
	fmt.Printf("   DNS-Server (Round-Robin): %d\n", registry.Count())
	servers := registry.GetAllServers()
	for _, s := range servers {
		fmt.Printf("     • %s (%s)\n", s.GetName(), s.GetAddress())
	}
	fmt.Printf("   Blacklist-Regeln: %d\n", blacklist.Count())
	fmt.Printf("   Cache TTL: 2 Stunden\n")
	fmt.Printf("   Cache Cleanup: alle 5 Minuten\n\n")

	// Starte DNS-Server auf Port 15353 (nicht-privilegiert für Demo)
	// Für produktiven Betrieb auf Port 53 mit sudo starten
	dnsAddr := ":15353"
	dnsServer, err := server.NewDNSServer(dnsAddr, proxy)
	if err != nil {
		log.Fatalf("Fehler beim Erstellen des DNS-Servers: %v", err)
	}

	fmt.Printf("🚀 Starte DNS-Server auf %s...\n", dnsAddr)
	err = dnsServer.Start()
	if err != nil {
		log.Fatalf("Fehler beim Starten des DNS-Servers: %v", err)
	}

	fmt.Println("✅ DNS-Server läuft!")
	fmt.Println("\n📖 Nutzung:")
	fmt.Println("   dig @127.0.0.1 -p 15353 example.com")
	fmt.Println("   nslookup example.com 127.0.0.1 -port=15353")
	fmt.Println("\n🧪 Test blockierte Domain:")
	fmt.Println("   dig @127.0.0.1 -p 15353 ads.example.com")
	fmt.Println("\n⏹  Beenden mit Ctrl+C")
	fmt.Println()

	// Warte auf Signal zum Beenden
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	fmt.Println("\n\n🛑 Beende DNS-Server...")

	// Statistik vor dem Beenden
	fmt.Printf("\n📊 Statistik:\n")
	fmt.Printf("   Cache-Einträge: %d\n", cache.Count())
	fmt.Printf("   Aktive DNS-Server: %d\n", registry.Count())
	fmt.Printf("   Blockierte Regeln: %d\n", blacklist.Count())

	// Server stoppen
	err = dnsServer.Stop()
	if err != nil {
		log.Printf("Fehler beim Stoppen: %v", err)
	}

	fmt.Println("✅ DNS-Server beendet.")
}
