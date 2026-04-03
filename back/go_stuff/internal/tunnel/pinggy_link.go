package tunnel

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"

	"vemenichy-server/internal/player"
)

// Start kicks off the Pinggy SSH tunnel in a background goroutine
func Start() {
	go func() {
		// Fetch token inside the function, AFTER main.go loads the .env file!
		sshToken := os.Getenv("sshToken") // Make sure this matches exactly what is in your .env
		if sshToken == "" {
			player.WebLog("[Tunnel] 🚨 Error: PINGGY_TOKEN is missing from .env!")
			return
		}

		player.WebLog("[Tunnel] Starting Pinggy Tunnel...")

		// Set up the SSH command
		cmd := exec.Command("ssh", "-tt", "-o", "StrictHostKeyChecking=no", "-o", "ServerAliveInterval=30", "-p", "443", "-R0:localhost:8080", sshToken+"@a.pinggy.io")

		// Create a pipe to stream both stdout and stderr (equivalent to 2>&1)
		pr, pw := io.Pipe()
		cmd.Stdout = pw
		cmd.Stderr = pw

		// Start the SSH command
		if err := cmd.Start(); err != nil {
			player.WebLog("[Tunnel] Failed to start SSH command: %v\n", err)
			return
		}

		// Ensure the writer closes when the command finishes
		go func() {
			cmd.Wait()
			pw.Close()
		}()

		// Set up a scanner to read the output
		scanner := bufio.NewScanner(pr)

		// Custom split function to handle newlines, carriage returns, and backticks.
		splitFunc := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
			if atEOF && len(data) == 0 {
				return 0, nil, nil
			}
			if i := bytes.IndexAny(data, "\n\r`"); i >= 0 {
				return i + 1, data[0:i], nil
			}
			if atEOF {
				return len(data), data, nil
			}
			return 0, nil, nil
		}
		scanner.Split(splitFunc)

		// Exact regex from your bash script
		urlRegex := regexp.MustCompile(`https?://[a-zA-Z0-9.-]+\.pinggy\.link`)
		webhookSent := false

		// Stream the output live
		for scanner.Scan() {
			text := scanner.Text()

			// Uncomment to debug raw Pinggy output
			// player.WebLog("[Tunnel Debug] %s\n", text)

			// Look for the URL and fire the webhook once
			if !webhookSent {
				match := urlRegex.FindString(text)
				if match != "" {
					player.WebLog("[Tunnel] ✅ Successfully caught clean URL: %s\n", match)
					sendWebhook(match)
					webhookSent = true
				}
			}
		}

		if err := scanner.Err(); err != nil {
			player.WebLog("[Tunnel] Error reading SSH output stream: %v\n", err)
		}

		player.WebLog("[Tunnel] SSH command exited.")
	}()
}

// sendWebhook handles formatting and posting the JSON payload to Discord
func sendWebhook(tunnelURL string) {
	webhookURL := os.Getenv("webhookURL") // Grab the webhook URL here too
	if webhookURL == "" {
		player.WebLog("[Tunnel] 🚨 Error: WEBHOOK_URL is missing from .env!")
		return
	}

	payload := map[string]string{
		"content": fmt.Sprintf(":cd: **Vemenichy is Online!**\nAccess Dashboard: %s", tunnelURL),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		player.WebLog("[Tunnel] Failed to marshal webhook payload: %v\n", err)
		return
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		player.WebLog("[Tunnel] Failed to send webhook: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		player.WebLog("[Tunnel] 🚀 Payload fired to Discord!")
	} else {
		player.WebLog("[Tunnel] Webhook returned non-200 status: %d\n", resp.StatusCode)
	}
}
