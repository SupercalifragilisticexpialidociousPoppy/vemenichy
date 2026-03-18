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
	// 1. Grab the URL from the request
	targetURL := r.URL.Query().Get("url")
	if targetURL == "" {
		http.Error(w, "Missing 'url' parameter", http.StatusBadRequest)
		return
	}

	// 2. Extract the track ID so we know what to name the file
	trackID := r.URL.Query().Get("v")
	if trackID == "" {
		// Quick fallback: if 'v' wasn't passed directly, try to rip it out of the YouTube URL
		if strings.Contains(targetURL, "v=") {
			parts := strings.Split(targetURL, "v=")
			trackID = strings.Split(parts[1], "&")[0]
		} else {
			trackID = "unknown_id"
		}
	}

	// 3. Prepare the yt-dlp command BEFORE the goroutine so it knows what to run
	cmd := exec.Command("yt-dlp",
		"-x",                    // Extract audio only
		"--audio-format", "mp3", // Force it into an mp3
		"-o", "sessions/%(id)s.%(ext)s", // Save it in the sessions folder as ID.mp3
		targetURL,
	)

	// 4. 🚨 Spawn the background worker to handle the heavy lifting!
	go func() {
		fmt.Printf("📥 Downloading track: %s...\n", trackID)

		// Block this specific background thread until yt-dlp AND ffmpeg finish
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("🚨 Download failed: %v\n📜 RAW ERROR:\n%s\n", err, string(output))
			return // Kills this worker, song never queues
		}

		fmt.Println("✅ Download complete! Handing over to DJ...")

		// Now that the file is 100% written to the disk, it is safe to queue
		newTrack := player.Track{
			ID:       trackID,
			Title:    "Track " + trackID, // You can parse the real title later if you want
			Filepath: fmt.Sprintf("sessions/%s.mp3", trackID),
		}
		player.AddToQueue(newTrack)
	}()

	// 5. 🚨 Respond to the frontend IMMEDIATELY so the browser doesn't freeze
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "success", "message": "Download started and will be queued shortly"}`))
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
