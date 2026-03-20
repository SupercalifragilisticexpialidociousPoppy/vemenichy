package player

import (
	"fmt"
	"net"
	"os/exec"
	"sync"
	"time"
)

type Track struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Filepath string `json:"filepath"`
}

var (
	queue        []Track
	mu           sync.Mutex // The bouncer that protects the queue
	currentCmd   *exec.Cmd  // Keeps track of the active mpv process
	currentTrack *Track
	ServerLogs   []string
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

		// Boot mpv
		currentCmd = exec.Command("mpv",
			"--no-video",
			"--input-ipc-server=/tmp/vemenichy.sock", // 🚨 LINUX SOCKET
			nowPlaying.Filepath,
		)

		// Run() blocks the loop until the song finishes naturally (or is killed)
		err := currentCmd.Run()
		if err != nil {
			// This will trigger when we intentionally kill the process to skip
			fmt.Printf("⏹️ Track ended or skipped.\n")
		}

		mu.Lock()
		currentCmd = nil
		currentTrack = nil
		mu.Unlock()
	}
}

// Skip cleanly shuts down the current mpv instance via IPC so the loop grabs the next song
func Skip() {
	mu.Lock()
	defer mu.Unlock()

	if currentCmd != nil {
		fmt.Println("⏭️ Skipping track...")
		sendIPC("quit") // 🚨 NATIVE GO CALL
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
		sendIPC("cycle pause")
	} else {
		fmt.Println("⚠️ Nothing is currently playing.")
	}
}

// To get current song
func GetStatus() (*Track, []Track) {
	mu.Lock()
	defer mu.Unlock()

	queueCopy := make([]Track, len(queue))
	copy(queueCopy, queue)

	return currentTrack, queueCopy
}

// SetVolume uses the IPC pipe to change mpv's volume (0 to 100)
func SetVolume(level string) {
	mu.Lock()
	defer mu.Unlock()

	if currentCmd != nil {
		sendIPC(fmt.Sprintf("set volume %s", level))
	}
}

// WebLog prints to the console AND saves it for the frontend terminal
func WebLog(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Println(msg) // Keep it printing in the physical terminal

	mu.Lock()
	defer mu.Unlock()
	ServerLogs = append(ServerLogs, msg)
	// Keep only the last 50 logs so we don't eat all your RAM
	if len(ServerLogs) > 50 {
		ServerLogs = ServerLogs[1:]
	}
}

// GetLogs safely passes the log history to the API
func GetLogs() []string {
	mu.Lock()
	defer mu.Unlock()
	logsCopy := make([]string, len(ServerLogs))
	copy(logsCopy, ServerLogs)
	return logsCopy
}

// sendIPC natively connects to mpv's Unix socket
func sendIPC(cmd string) {
	conn, err := net.Dial("unix", "/tmp/vemenichy.sock")
	if err != nil {
		WebLog("🚨 IPC connection failed: %v", err)
		return
	}
	defer conn.Close()

	// mpv requires a newline character to register the command
	conn.Write([]byte(cmd + "\n"))
}
