package player

import (
	"fmt"
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
	mu           sync.Mutex
	currentCmd   *exec.Cmd
	currentTrack *Track
	ServerLogs   []string
)

// WebLog prints to the console AND saves it for the frontend terminal
func WebLog(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Println(msg)

	mu.Lock()
	defer mu.Unlock()
	ServerLogs = append(ServerLogs, msg)
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

// AddToQueue is called by your API when a download finishes
func AddToQueue(track Track) {
	mu.Lock()
	queue = append(queue, track)
	qLen := len(queue)
	mu.Unlock() // Unlock early!

	WebLog("🎵 Added to queue: %s (Total in queue: %d)", track.Title, qLen)
}

// StartDJ runs forever in the background
func StartDJ() {
	WebLog("🎧 DJ Module Loaded. Waiting for tracks...")

	for {
		mu.Lock()
		if len(queue) == 0 {
			mu.Unlock()
			time.Sleep(1 * time.Second)
			continue
		}

		nowPlaying := queue[0]
		queue = queue[1:]

		// 🚨 THE FIX: Actually save the current track so the API can see it!
		trackCopy := nowPlaying
		currentTrack = &trackCopy
		mu.Unlock()

		WebLog("▶️ Now Playing: %s", nowPlaying.Title)

		currentCmd = exec.Command("mpv",
			"--no-video",
			"--input-ipc-server=\\\\.\\pipe\\vemenichy",
			nowPlaying.Filepath,
		)

		err := currentCmd.Run()
		if err != nil {
			WebLog("⏹️ Track ended or skipped.")
		}

		mu.Lock()
		currentCmd = nil
		currentTrack = nil
		mu.Unlock()
	}
}

// Skip cleanly shuts down the current mpv instance
func Skip() {
	mu.Lock()
	active := currentCmd != nil
	mu.Unlock() // 🚨 FIX: Unlock before running commands/logs to prevent deadlocks

	if active {
		WebLog("⏭️ Skipping track...")
		exec.Command("cmd", "/c", "echo quit > \\\\.\\pipe\\vemenichy").Run()
	} else {
		WebLog("⚠️ Nothing is currently playing.")
	}
}

// GetQueue returns a safe copy of the current queue (Restored!)
func GetQueue() []Track {
	mu.Lock()
	defer mu.Unlock()

	queueCopy := make([]Track, len(queue))
	copy(queueCopy, queue)
	return queueCopy
}

// TogglePause sends the spacebar equivalent to mpv via IPC
func TogglePause() {
	mu.Lock()
	active := currentCmd != nil && currentCmd.Process != nil
	mu.Unlock() // 🚨 FIX: Unlock before running commands/logs

	if active {
		WebLog("⏯️ Toggling Play/Pause...")
		exec.Command("cmd", "/c", "echo cycle pause > \\\\.\\pipe\\vemenichy").Run()
	} else {
		WebLog("⚠️ Nothing is currently playing.")
	}
}

// GetStatus returns current song and queue
func GetStatus() (*Track, []Track) {
	mu.Lock()
	defer mu.Unlock()

	queueCopy := make([]Track, len(queue))
	copy(queueCopy, queue)

	return currentTrack, queueCopy
}

// SetVolume uses the IPC pipe to change mpv's volume
func SetVolume(level string) {
	mu.Lock()
	active := currentCmd != nil
	mu.Unlock()

	if active {
		cmdStr := fmt.Sprintf("echo set volume %s > \\\\.\\pipe\\vemenichy", level)
		exec.Command("cmd", "/c", cmdStr).Run()
	}
}
