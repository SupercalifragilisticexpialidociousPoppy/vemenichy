package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"vemenichy-server/internal/player"
	"vemenichy-server/internal/state"
	"vemenichy-server/pkg/youtube"
)

// 1. PING HANDLER (Status Check)
func HandlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	state.Global.Mutex.Lock()
	status := fmt.Sprintf(`{"status":"online", "current":"%s", "next":"%s", "preloading":%t}`,
		state.Global.CurrentSong, state.Global.NextSong, state.Global.IsPreloading)
	state.Global.Mutex.Unlock()

	w.Write([]byte(status))
}

// 2. ADD HANDLER (Queue a song)
func HandleAdd(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	url := r.URL.Query().Get("url")
	if url != "" {
		state.Global.Playlist <- url
		w.Write([]byte(`{"status": "Added to queue"}`))
	}
}

// 3. SEARCH HANDLER (Soundcloud yt-dlp)
func HandleSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query().Get("q")
	source := r.URL.Query().Get("source")

	if query == "" {
		w.Write([]byte("[]"))
		return
	}

	var results []map[string]string

	if source == "sc" {
		// Scraping SoundCloud with yt-dlp
		fmt.Printf("🔍 Searching SoundCloud for: '%s'\n", query)

		cmd := exec.Command("yt-dlp",
			"--dump-json",
			"--flat-playlist",
			"--skip-download",
			"scsearch10:"+query)

		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("🚨 YT-DLP Error: %v\n", err)
			w.Write([]byte("[]"))
			return
		}

		// yt-dlp outputs one standalone JSON object per line
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")

		for _, line := range lines {
			if line == "" {
				continue
			}

			// Create a temporary struct to hold the exact JSON fields we care about
			var trackData struct {
				Title    string  `json:"title"`
				Uploader string  `json:"uploader"`
				Duration float64 `json:"duration"` // yt-dlp outputs duration in raw seconds
				ID       string  `json:"id"`
				URL      string  `json:"url"`
			}

			// Unmarshal the JSON string into our struct
			if err := json.Unmarshal([]byte(line), &trackData); err != nil {
				fmt.Printf("Skipping unparseable JSON line: %v\n", err)
				continue
			}

			// Convert raw seconds (e.g., 205) to mm:ss format (e.g., 3:25)
			durSec := int(trackData.Duration)
			durStr := fmt.Sprintf("%d:%02d", durSec/60, durSec%60)

			entry := map[string]string{
				"title":    trackData.Title,
				"artist":   trackData.Uploader,
				"duration": durStr,
				"id":       trackData.ID,
				"url":      trackData.URL,
				"source":   "sc",
			}
			results = append(results, entry)
		}
	} else if source == "yt" {
		// --- YOUTUBE: Official API ---
		fmt.Printf("🔍 Searching YouTube for: '%s'\n", query)

		ytResults, err := youtube.Search(query)
		if err != nil {
			fmt.Printf("YouTube API Error: %v\n", err)
			w.Write([]byte("[]"))
			return
		}
		results = ytResults
	}

	if results == nil {
		results = []map[string]string{}
	}

	json.NewEncoder(w).Encode(results)
}

// 4. Download that bitch.
func HandleDownload(w http.ResponseWriter, r *http.Request) {
	targetURL := r.URL.Query().Get("url")
	trackTitle := r.URL.Query().Get("title") // Grabbing the real title!

	if trackTitle == "" || trackTitle == "undefined" {
		trackTitle = "Unknown Track"
	}

	trackID := r.URL.Query().Get("v")
	if trackID == "" && strings.Contains(targetURL, "v=") {
		parts := strings.Split(targetURL, "v=")
		trackID = strings.Split(parts[1], "&")[0]
	}

	// 🚨 THE DISGUISE & COOKIE BYPASS
	cmd := exec.Command("yt-dlp",
		"--cookies", "cookies.txt",
		"-x",
		"--audio-format", "mp3",
		"-o", "sessions/%(id)s.%(ext)s",
		targetURL,
	)

	go func() {
		player.WebLog("📥 Downloading: %s", trackTitle)

		output, err := cmd.CombinedOutput()
		if err != nil {
			player.WebLog("🚨 Download failed: %v\n📜 RAW ERROR:\n%s", err, string(output))
			return
		}

		player.WebLog("✅ Download complete! Sending to DJ...")

		newTrack := player.Track{
			ID:       trackID,
			Title:    trackTitle, // 🚨 The Ghost is dead. Real title is injected here.
			Filepath: fmt.Sprintf("sessions/%s.mp3", trackID),
		}
		player.AddToQueue(newTrack)
	}()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "success", "message": "Download started"}`))
}

// HandleSkip aggressively kills the current track to trigger the next one
func HandleSkip(w http.ResponseWriter, r *http.Request) {
	// if r.Method != http.MethodPost {
	// 	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	// 	return
	// }

	player.Skip()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "success", "message": "Track skipped"}`))
}

// HandleGetQueue returns the upcoming songs in JSON format
func HandleGetQueue(w http.ResponseWriter, r *http.Request) {
	// if r.Method != http.MethodGet {
	// 	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	// 	return
	// }

	currentQueue := player.GetQueue()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"queue":  currentQueue,
	})
}

// HandlePause toggles the playback state of the current track
func HandlePause(w http.ResponseWriter, r *http.Request) {
	// if r.Method != http.MethodPost {
	// 	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	// 	return
	// }

	player.TogglePause()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "success", "message": "Playback toggled"}`))
}

// HandleStatus returns the currently playing track and the queue
func HandleStatus(w http.ResponseWriter, r *http.Request) {
	nowPlaying, currentQueue := player.GetStatus()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"now_playing": nowPlaying,
		"queue":       currentQueue,
	})
}

// HandleVolume changes the player volume
func HandleVolume(w http.ResponseWriter, r *http.Request) {
	level := r.URL.Query().Get("v")
	if level != "" {
		player.SetVolume(level)
	}
	w.WriteHeader(http.StatusOK)
}

func HandleLogs(w http.ResponseWriter, r *http.Request) {
	logs := player.GetLogs()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"logs": logs})
}
