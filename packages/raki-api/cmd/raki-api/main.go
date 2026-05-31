package main

import (
	"flag"
	"log"
	"net/http"
	"raki/adminapi"
	"strings"
)

func main() {
	socketPath := flag.String("socket", "/run/soju/admin", "Path to Soju admin socket")
	listenAddr := flag.String("listen", ":8080", "HTTP listen address")
	apiKeys := flag.String("api-keys", "", "Comma-separated list of API keys")
	flag.Parse()

	keys := strings.Split(*apiKeys, ",")
	if *apiKeys == "" || len(keys) == 0 {
		log.Fatal("at least one API key must be provided via -api-keys")
	}

	server := adminapi.NewServer(*socketPath, keys)

	log.Printf("raki-api listening on %s", *listenAddr)
	if err := http.ListenAndServe(*listenAddr, server); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
