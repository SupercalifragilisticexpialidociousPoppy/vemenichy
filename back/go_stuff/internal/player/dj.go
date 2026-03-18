package player

import (
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// Track holds the data for the song we want to play
type Track struct {
	ID       string
	Title    string
	Filepath string
}

var (
	queue      []Track
	mu         sync.Mutex // The bouncer that protects the queue
	currentCmd *exec.Cmd  // Keeps track of the active mpv process
)

// AddToQueue is called by your API when a download finishes
func AddToQueue(track Track) {
	mu.Lock() // Lock the queue so no one else can touch it
	defer mu.Unlock()

	queue = append(queue, track)
	fmt.Printf("🎵 Added to queue: %s (Total in queue: %d)\n", track.Title, len(queue))
}

// StartDJ runs forever in the background
func StartDJ() {
	fmt.Println("🎧 DJ Module Loaded. Waiting for tracks...")

	for {
		mu.Lock()
		// If queue is empty, unlock and chill for 1 second
		if len(queue) == 0 {
			mu.Unlock()
			time.Sleep(1 * time.Second)
			continue
		}

		// Pop the first track off the queue
		nowPlaying := queue[0]
		queue = queue[1:] // Reslice to remove the first element
		mu.Unlock()

		fmt.Printf("▶️ Now Playing: %s\n", nowPlaying.Title)

		// 🚨 THE HACK: Boot mpv and force it to open a secret Windows Named Pipe
		currentCmd = exec.Command("mpv",
			"--no-video",
			"--input-ipc-server=\\\\.\\pipe\\vemenichy", // Windows specific pipe
			nowPlaying.Filepath,
		)

		// Run() blocks the loop until the song finishes naturally (or is killed)
		err := currentCmd.Run()
		if err != nil {
			// This will trigger when we intentionally kill the process to skip
			fmt.Printf("⏹️ Track ended or skipped.\n")
		}

		currentCmd = nil // Clear the command
	}
}

// Skip cleanly shuts down the current mpv instance via IPC so the loop grabs the next song
func Skip() {
	mu.Lock()
	defer mu.Unlock()

	if currentCmd != nil {
		fmt.Println("⏭️ Skipping track...")

		// 🚨 THE CLEAN KILL: Use the IPC pipe to tell the engine to suicide
		err := exec.Command("cmd", "/c", "echo quit > \\\\.\\pipe\\vemenichy").Run()
		if err != nil {
			fmt.Printf("🚨 Failed to send skip command: %v\n", err)
		}
	} else {
		fmt.Println("⚠️ Nothing is currently playing.")
	}
}

// GetQueue returns a safe copy of the current queue
func GetQueue() []Track {
	mu.Lock()
	defer mu.Unlock()

	// Return a copy so the API doesn't accidentally modify the real queue
	queueCopy := make([]Track, len(queue))
	copy(queueCopy, queue)
	return queueCopy
}

// TogglePause sends the spacebar equivalent to mpv via IPC
func TogglePause() {
	mu.Lock()
	defer mu.Unlock()

	if currentCmd != nil && currentCmd.Process != nil {
		fmt.Println("⏯️ Toggling Play/Pause...")

		// 🚨 THE INJECTION: Use Windows CMD to echo the command into mpv's pipe
		err := exec.Command("cmd", "/c", "echo cycle pause > \\\\.\\pipe\\vemenichy").Run()
		if err != nil {
			fmt.Printf("🚨 Failed to send pause command: %v\n", err)
		}
	} else {
		fmt.Println("⚠️ Nothing is currently playing.")
	}
}
