package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/joho/godotenv"

	"vemenichy-server/internal/api"
	"vemenichy-server/internal/player"
	"vemenichy-server/internal/state"
)

func main() {
	// 1. INITIALIZE THE BRAIN
	// (Cleans up old temp files like 'next_song.mp3' from previous crashes)
	err := godotenv.Load(".env")
	// envMap, _ := godotenv.Read(".env")
	// fmt.Printf("🕵️ Raw Env Map: %v\n", envMap)

	if err != nil {
		log.Println("\n[!] No .env file found, looking for system variables")
	}
	state.Initialize()

	// 2. START THE MUSCLES (The DJ)
	// We run this in a 'go routine' so it runs in the background
	// while the HTTP server listens for requests.
	go player.StartDJ()

	// 3. PINNGY LINK
	//tunnel.Start()

	// 4. SETUP THE MOUTH (The HTTP Router)
	// We ask the API package to give us a configured router with all endpoints
	// (/ping, /add, /search, /devices) registered.
	mux := api.NewRouter()

	// 5. START THE SERVER
	port := ":8080"
	fmt.Printf("Vemenichy Server v0.2 started on %s\n", port)
	fmt.Println("   Ready to accept commands...")

	// This blocks forever (waiting for requests)
	err = http.ListenAndServe(port, mux)
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}
