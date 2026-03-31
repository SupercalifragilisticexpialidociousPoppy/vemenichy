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

	player.WebLog("🔎 Search Request: %s for '%s'", strings.ToUpper(source), query)

	var results []map[string]string

	if source == "sc" {
		// Get global track index.

		// Clean json dump:
		// Windows: yt-dlp --skip-download --flat-playlist --print "{\`"title\`": %(title)j, \`"uploader\`": %(uploader)j, \`"duration\`": %(duration)j, \`"webpage_url\`": %(webpage_url)j}" "scsearch10:troyboi afterhours"
		//Linux: yt-dlp --skip-download --flat-playlist --print '{"title": %(title)j, "uploader": %(uploader)j, "duration": %(duration)j, "webpage_url": %(webpage_url)j}' "scsearch10:troyboi afterhours"

		printFormat := `{"title": %(title)j, "uploader": %(uploader)j, "duration": %(duration)j, "webpage_url": %(webpage_url)j, "id": %(id)j}`
		searchArg := "scsearch10:" + query

		cmd := exec.Command("yt-dlp",
			"--skip-download",
			"--flat-playlist",
			"--print", printFormat,
			searchArg,
		)
		output, err := cmd.CombinedOutput()
		if err != nil {
			player.WebLog("🚨 YT-DLP Error: %v", err)
			w.Write([]byte("[]"))
			return
		}

		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			var trackData struct {
				Title    string  `json:"title"`
				Uploader string  `json:"uploader"`
				Duration float64 `json:"duration"`
				ID       string  `json:"id"`
				URL      string  `json:"webpage_url"`
			}
			if err := json.Unmarshal([]byte(line), &trackData); err != nil {
				continue
			}

			durSec := int(trackData.Duration)
			durStr := fmt.Sprintf("%d:%02d", durSec/60, durSec%60)

			results = append(results, map[string]string{
				"title": trackData.Title, "artist": trackData.Uploader,
				"duration": durStr, "id": trackData.ID, "url": trackData.URL, "source": "sc",
			})
		}
	} else if source == "yt" {
		ytResults, err := youtube.Search(query)
		if err != nil {
			player.WebLog("🚨 YouTube API Error: %v", err)
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
	trackTitle := r.URL.Query().Get("title")
	trackArtist := r.URL.Query().Get("artist")
	trackDuration := r.URL.Query().Get("duration")
	trackSource := r.URL.Query().Get("source")
	trackID := r.URL.Query().Get("id")

	if trackTitle == "" || trackTitle == "undefined" {
		trackTitle = "Unknown Track"
	}
	if trackArtist == "" || trackArtist == "undefined" {
		trackArtist = "Unknown Artist"
	}
	if trackDuration == "" || trackDuration == "undefined" {
		trackDuration = "--:--"
	}

	// Fallback just in case the UI fails to send an ID for YouTube
	if trackID == "" && trackSource == "yt" && strings.Contains(targetURL, "v=") {
		parts := strings.Split(targetURL, "v=")
		trackID = strings.Split(parts[1], "&")[0]
	}

	player.WebLog("📥 [Step 3: Handler] Source: %s | Title: '%s' | ID: '%s'", strings.ToUpper(trackSource), trackTitle, trackID)

	var cmd *exec.Cmd

	if trackSource == "sc" {
		cmd = exec.Command("yt-dlp",
			"-x",
			"--audio-format", "mp3",
			"-o", "sessions/%(id)s.%(ext)s",
			targetURL,
		)
	} else {
		// To download age-restricted audio, YT DRM needs us to solve JS puzzles. cookies.txt and nodejs required for this.
		cmd = exec.Command("yt-dlp",
			"--cookies", "cookies.txt",
			"--js-runtimes", "node",
			"-x",
			"--audio-format", "mp3",
			"-o", "sessions/%(id)s.%(ext)s",
			targetURL,
		)
	}

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
			Title:    trackTitle,
			Artist:   trackArtist,
			Duration: trackDuration,
			Filepath: fmt.Sprintf("sessions/%s.mp3", trackID),
		}

		player.AddToQueue(newTrack)
	}()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "success", "message": "Download started"}`))
}

// HandleSkip aggressively kills the current track to trigger the next one
func HandleSkip(w http.ResponseWriter, r *http.Request) {
	player.WebLog("📲 UI Request: Skip Track")
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
	player.WebLog("📲 UI Request: Toggle Pause")
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
		player.WebLog("📲 UI Request: Set Volume to %s%%", level)
		player.SetVolume(level)
	}
	w.WriteHeader(http.StatusOK)
}

func HandleLogs(w http.ResponseWriter, r *http.Request) {
	logs := player.GetLogs()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"logs": logs})
}
