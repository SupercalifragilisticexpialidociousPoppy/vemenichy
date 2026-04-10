package state

import (
	"os"
	"vemenichy-server/internal/player"
)

// All the downloaded songs are stored in a sessions folder in the  same directory as the binary file. To not murder my pi's SD Card, all songs are deleted at the start of every session.
func Initialize() {
	player.WebLog("🧹 Sweeping old session data...")

	// 1. Nuke the entire sessions folder and everything inside it
	os.RemoveAll("sessions")

	// 2. Create a fresh, empty folder (0755 are standard safe folder permissions)
	err := os.MkdirAll("sessions", 0755)
	if err != nil {
		player.WebLog("🚨 Failed to create sessions folder: %v", err)
	}
}

// This can be moved to the shutdown sequence, but I like to keep the songs for offline listening lol
