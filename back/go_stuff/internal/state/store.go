package state

import (
	"fmt"
	"os"
	"sync"
)

// AppState defines all the shared variables in your system.
type AppState struct {
	CurrentSong  string
	NextSong     string
	IsPreloading bool

	// The Playlist is a "Channel".
	// Think of it like a physical pipe:
	// The API drops URLs in one end, and the Player pulls them out the other.
	Playlist chan string

	// Mutex is a "Lock".
	// It prevents the API and Player from fighting over variables at the exact same time.
	Mutex sync.Mutex
}

// Global is the single instance everyone will use.
// We initialize it with safe default values.
var Global = &AppState{
	CurrentSong: "Nothing",
	NextSong:    "Nothing",

	// Make a channel that can hold 100 songs before blocking
	Playlist: make(chan string, 100),
}

// Initialize cleans up any mess left over from the last time the server ran.
func Initialize() {
	fmt.Println("🧹 Sweeping old session data...")

	// 1. Nuke the entire folder and everything inside it
	os.RemoveAll("sessions")

	// 2. Create a fresh, empty folder (0755 are standard safe folder permissions)
	err := os.MkdirAll("sessions", 0755)
	if err != nil {
		fmt.Printf("🚨 Failed to create sessions folder: %v\n", err)
	}
	os.Remove("current_song.mp3")
	os.Remove("next_song.mp3")
}
