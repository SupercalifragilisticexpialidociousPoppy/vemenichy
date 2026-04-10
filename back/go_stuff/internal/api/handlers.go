package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
	"vemenichy-server/internal/player"
	"vemenichy-server/internal/tunnel"
	"vemenichy-server/pkg/youtube"
)

// To execute system level commands - currently handling the pinggy global tunnel and shutting down the server and the pi - directly from the UI, we've kept a password to prevent abuse.
type GlobalReq struct {
	Password string `json:"password"`
}

// ====================================
//  SEARCH HANDLER (Soundcloud, yt-dlp)
// ====================================

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
		// Use yt-dlp to scrape soundcloud search.

		// Clean json dump to scrape manually:
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
		// Use Youtube's search api (Google Cloud Console)

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

// ======================
//  Download that bitch!
// ======================

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
	// If it STILL doesn't work, terminate the process.
	if trackID == "" {
		player.WebLog("📥 [FATAL ERROR] Failed to get song id. Terminating download sequence...")
		return
	}

	player.WebLog("📥 [Step 3: Handler] Source: %s | Title: '%s' | ID: '%s'", strings.ToUpper(trackSource), trackTitle, trackID)

	var cmd *exec.Cmd

	if trackSource == "sc" {
		// yt-dlp scraper again, soundcloud is very cute.
		cmd = exec.Command("yt-dlp",
			"-x",
			"--audio-format", "mp3",
			"-o", "sessions/%(id)s.%(ext)s",
			targetURL,
		)
	} else {
		// Requirements: nodejs and cookies.txt

		// YT DRM needs us to solve JS puzzles. nodejs required for this. You can use other things like deno or bun just fine.
		// get node js in your environment and MAKE SURE yt-dlp can see it. This was a massive headache when trying to run this server on my new raspi - yt-dlp couldn't see nodejs's file location and therefore failed DRM's JS puzzles.

		// To get age restricted audio, youtube checks if you're legit using cookies. You can extract your personal youtube accounts ccookies using browser extensions like 'Get Cookies.txt LOCALLY'
		cmd = exec.Command("yt-dlp",
			"--cookies", "cookies.txt",
			"--js-runtimes", "node",
			"-x",
			"--audio-format", "mp3",
			"-o", "sessions/%(id)s.%(ext)s",
			targetURL,
		)

		// If you don't want to give your personal cookies here, use this alternate command:
		// yt-dlp -x --audio-format mp3 -o sessions/%(id)s.%(ext)s
		// Note that this will throw fatal errors when downloading mature songs.
	}

	// Report to frontend on how the download is going, and add a newly downloaded song to queue.
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

// ===================
//  UI Controls relay
// ===================

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

// Send logs
func HandleLogs(w http.ResponseWriter, r *http.Request) {
	logs := player.GetLogs()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"logs": logs})
}

// =================
//  SYSTEM COMMANDS
// =================

// Activates global pinggy tunnel.
func HandleEnableGlobal(w http.ResponseWriter, r *http.Request) {
	var req GlobalReq
	json.NewDecoder(r.Body).Decode(&req)

	if req.Password != os.Getenv("GLOBAL_PASSWORD") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if already active before attempting to start
	if tunnel.IsActive() {
		http.Error(w, "Tunnel is already running", http.StatusConflict)
		return
	}

	if err := tunnel.StartTunnel(); err != nil {
		http.Error(w, "Failed to start tunnel", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Deactivate the tunnel.
func HandleDisableGlobal(w http.ResponseWriter, r *http.Request) {
	var req GlobalReq
	json.NewDecoder(r.Body).Decode(&req)

	if req.Password != os.Getenv("GLOBAL_PASSWORD") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if actually active before attempting to kill
	if !tunnel.IsActive() {
		http.Error(w, "Tunnel is not running", http.StatusConflict)
		return
	}

	tunnel.StopTunnel()
	w.WriteHeader(http.StatusOK)
}

// Shutdown Sequence
func HandlePowerOff(w http.ResponseWriter, r *http.Request) {
	var req GlobalReq
	json.NewDecoder(r.Body).Decode(&req)

	if req.Password != os.Getenv("GLOBAL_PASSWORD") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	player.WebLog("[MAINFRAME] Initiating system shutdown...")

	// Cleanly kill the tunnel so Discord gets the "Offline" webhook
	player.WebLog("[MAINFRAME] Checking if global tunnel is active.")
	if tunnel.IsActive() {
		player.WebLog("[MAINFRAME] Closing global tunnel.")
		tunnel.StopTunnel()
	}

	// Tell the frontend we are shutting down
	w.WriteHeader(http.StatusOK)
	player.WebLog("[MAINFRAME] All processes safe to terminate.")

	go func() {
		player.WebLog("[MAINFRAME] Shutting down in 10 seconds...")
		time.Sleep(7 * time.Second)
		player.WebLog("[MAINFRAME] Take care of yourself.")
		time.Sleep(3 * time.Second)
		exec.Command("bash", "../system/shutdown.sh").Run()
	}()
}
