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
	// 1. INITIALIZE THE BRAIN
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("\n[!] No .env file found, looking for system variables")
	}
	state.Initialize()

	// 2. START THE MUSCLES (The DJ)
	go player.StartDJ()

	// 4. SETUP THE MOUTH (The HTTP Router)
	mux := api.NewRouter()

	// 5. START THE SERVER
	port := ":8080"
	player.WebLog("🚀 Vemenichy Server v0.5 started on %s\n\tReady to accept commands...", port)

	err = http.ListenAndServe(port, mux)
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}

	// 3. PINNGY LINK
	tunnel.Start() // <-- Uncomment this
}
