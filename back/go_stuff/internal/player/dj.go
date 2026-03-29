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
	Artist   string `json:"artist"`
	Index    int    `json:"index"`
	Duration string `json:"duration"`
}

var (
	queue            []Track
	mu               sync.Mutex // Protects the music queue and player state
	logMu            sync.Mutex // Protects ONLY the logs (Prevents Deadlocks!)
	currentCmd       *exec.Cmd
	currentTrack     *Track
	ServerLogs       []string
	globalTrackIndex = 1
)

// --- UNIFIED LOGGING SYSTEM ---
func WebLog(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Println(msg) // Keep printing to the physical Pi terminal

	logMu.Lock()
	defer logMu.Unlock()

	ServerLogs = append(ServerLogs, msg)
	if len(ServerLogs) > 60 { // Keeps RAM perfectly safe
		ServerLogs = ServerLogs[1:]
	}
}

func GetLogs() []string {
	logMu.Lock()
	defer logMu.Unlock()
	logsCopy := make([]string, len(ServerLogs))
	copy(logsCopy, ServerLogs)
	return logsCopy
}

// --- DJ & QUEUE LOGIC ---
func AddToQueue(track Track) {
	mu.Lock()
	track.Index = globalTrackIndex // Stamp the track with its permanent number
	globalTrackIndex++             // Increment for the next song
	queue = append(queue, track)
	mu.Unlock()

	// 🚨 TRACER 5: Did the struct survive entering the queue?
	WebLog("📦 [Step 5: Queue] Added: [%d] %s by %s (Dur: %s)", track.Index, track.Title, track.Artist, track.Duration)
}

func StartDJ() {
	WebLog("🎧 DJ Module Loaded. Waiting for tracks...")

	for {
		mu.Lock()
		if len(queue) == 0 {
			mu.Unlock()
			time.Sleep(1 * time.Second)
			continue
		}

		// Pop track and SET GLOBAL VARIABLE (Fixes the blank UI bug!)
		nowPlaying := queue[0]
		queue = queue[1:]
		currentTrack = &nowPlaying

		// 🚨 TRACER 6: Did the metadata survive until playback?
		WebLog("▶️ [Step 6: Player] Popped for playback: %s by %s", nowPlaying.Title, nowPlaying.Artist)

		// Boot mpv
		currentCmd = exec.Command("mpv",
			"--no-video",
			"--input-ipc-server=/tmp/vemenichy.sock",
			nowPlaying.Filepath,
		)
		mu.Unlock()

		WebLog("▶️ Now Playing: %s", nowPlaying.Title)

		err := currentCmd.Run()
		if err != nil {
			WebLog("⏹️ Track ended or skipped naturally.")
		}

		mu.Lock()
		currentCmd = nil
		currentTrack = nil
		mu.Unlock()
	}
}

// --- IPC CONTROLS ---
func Skip() {
	mu.Lock()
	defer mu.Unlock()

	if currentCmd != nil {
		sendIPC("quit")
	} else {
		WebLog("⚠️ Attempted skip, but nothing is playing.")
	}
}

func TogglePause() {
	mu.Lock()
	defer mu.Unlock()

	if currentCmd != nil && currentCmd.Process != nil {
		sendIPC("cycle pause")
	} else {
		WebLog("⚠️ Attempted pause, but nothing is playing.")
	}
}

func SetVolume(level string) {
	mu.Lock()
	defer mu.Unlock()

	if currentCmd != nil {
		sendIPC(fmt.Sprintf("set volume %s", level))
	}
}

func GetStatus() (*Track, []Track) {
	mu.Lock()
	defer mu.Unlock()

	queueCopy := make([]Track, len(queue))
	copy(queueCopy, queue)

	return currentTrack, queueCopy
}

func GetQueue() []Track {
	mu.Lock()
	defer mu.Unlock()

	queueCopy := make([]Track, len(queue))
	copy(queueCopy, queue)
	return queueCopy
}

func sendIPC(cmd string) {
	conn, err := net.Dial("unix", "/tmp/vemenichy.sock")
	if err != nil {
		WebLog("🚨 IPC connection failed: %v", err)
		return
	}
	defer conn.Close()
	conn.Write([]byte(cmd + "\n"))
}
