package youtube

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// --- JSON STRUCTURES ---
type SearchResponse struct {
	Items []struct {
		ID struct {
			VideoID string `json:"videoId"`
		} `json:"id"`
	} `json:"items"`
}

type VideoDetailsResponse struct {
	Items []struct {
		ID      string `json:"id"`
		Snippet struct {
			Title        string `json:"title"`
			ChannelTitle string `json:"channelTitle"`
		} `json:"snippet"`
		ContentDetails struct {
			Duration string `json:"duration"` // Format: PT4M20S
		} `json:"contentDetails"`
	} `json:"items"`
}

// Search queries YouTube and formats the results for Vemenichy
// We use two api calls, one to get video ids, and the other to get video title, channel name, duration and id - first api call doesn't give duration.

func Search(query string) ([]map[string]string, error) {
	// Go to Google Cloud Console to set this up.
	var APIKey = os.Getenv("YOUTUBE_API_KEY")
	// TEMPORARY DEBUG LINE:
	// fmt.Printf("🕵️ Debug: Key length is %d\n", len(APIKey))

	// 1. GET VIDEO IDs (Search Endpoint)
	searchURL := fmt.Sprintf(
		"https://www.googleapis.com/youtube/v3/search?part=id&q=%s&type=video&maxResults=10&key=%s",
		url.QueryEscape(query), APIKey,
	)

	resp, err := http.Get(searchURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Trap Google's hidden HTTP errors
	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Google API Error (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	var searchResult SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		return nil, err
	}

	var videoIDs []string
	for _, item := range searchResult.Items {
		videoIDs = append(videoIDs, item.ID.VideoID)
	}

	if len(videoIDs) == 0 {
		return []map[string]string{}, nil
	}

	// 2. GET TITLES & DURATIONS (Videos Endpoint)
	detailsURL := fmt.Sprintf(
		"https://www.googleapis.com/youtube/v3/videos?part=snippet,contentDetails&id=%s&key=%s",
		strings.Join(videoIDs, ","), APIKey,
	)

	resp2, err := http.Get(detailsURL)
	if err != nil {
		return nil, err
	}
	defer resp2.Body.Close()

	// Trap Google's hidden HTTP errors
	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Google API Error (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	var detailsResult VideoDetailsResponse
	if err := json.NewDecoder(resp2.Body).Decode(&detailsResult); err != nil {
		return nil, err
	}

	// 3. FORMAT THE RESULTS
	var finalResults []map[string]string
	for _, video := range detailsResult.Items {
		entry := map[string]string{
			"title":    video.Snippet.Title,
			"artist":   video.Snippet.ChannelTitle,
			"duration": parseDuration(video.ContentDetails.Duration),
			"id":       video.ID,
			"url":      "https://www.youtube.com/watch?v=" + video.ID,
			"source":   "yt",
		}
		finalResults = append(finalResults, entry)
	}

	return finalResults, nil
}

// Helper to convert video duration to human-friendly format. eg "PT4M20S" to "4:20"
func parseDuration(isoDuration string) string {
	d, _ := time.ParseDuration(strings.ToLower(strings.Replace(isoDuration, "PT", "", 1)))
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}
