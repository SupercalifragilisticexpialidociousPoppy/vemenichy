package main

import (
	"log"
	"net/http"

	"github.com/joho/godotenv"

	"vemenichy-server/internal/api"
	"vemenichy-server/internal/player"
	"vemenichy-server/internal/state"
	"vemenichy-server/internal/tunnel"
)

func main() {
	// Get environment variables.
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("\n[!] No .env file found, looking for system variables")
	}
	state.Initialize()

	// Start the dj
	go player.StartDJ()

	// Setup the router
	mux := api.NewRouter()

	// Start the global pinggy tunnel.
	player.WebLog("Attempting pinggy-link...")
	tunnel.StartTunnel()

	// Start the server.
	port := ":8080"
	player.WebLog("🚀 Vemenichy Server v0.5 started on %s\n\tReady to accept commands...", port)

	err = http.ListenAndServe(port, mux)
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}

	// Adding new bootup features below http.ListenAndServe will never execute as this is an infinite loop. So have everything above it. Learned the hard way.
}
