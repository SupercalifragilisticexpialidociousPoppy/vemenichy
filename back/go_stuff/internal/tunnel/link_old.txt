package tunnel

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
)

var webhookURL = os.Getenv("webhookURL")
var sshToken = os.Getenv("sshToken")

// Start kicks off the Pinggy SSH tunnel in a background goroutine
func Start() {
	go func() {
		log.Println("[Tunnel] Starting Pinggy Tunnel...")

		// Set up the SSH command
		cmd := exec.Command("ssh", "-tt", "-o", "StrictHostKeyChecking=no", "-o", "ServerAliveInterval=30", "-p", "443", "-R0:localhost:8080", sshToken+"@a.pinggy.io")

		// Create a pipe to stream both stdout and stderr (equivalent to 2>&1)
		pr, pw := io.Pipe()
		cmd.Stdout = pw
		cmd.Stderr = pw

		// Start the SSH command
		if err := cmd.Start(); err != nil {
			log.Printf("[Tunnel] Failed to start SSH command: %v\n", err)
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
		// This mimics your bash script's: tr '`' '\n'
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

			// Uncomment this line to see the raw Pinggy logs in your console for debugging:
			// log.Printf("[Tunnel Debug] %s\n", text)

			// Look for the URL and fire the webhook once
			if !webhookSent {
				match := urlRegex.FindString(text)
				if match != "" {
					log.Printf("[Tunnel] ✅ Successfully caught clean URL: %s\n", match)
					sendWebhook(match)
					webhookSent = true
				}
			}
		}

		if err := scanner.Err(); err != nil {
			log.Printf("[Tunnel] Error reading SSH output stream: %v\n", err)
		}

		log.Println("[Tunnel] SSH command exited.")
	}()
}

// sendWebhook handles formatting and posting the JSON payload to Discord
func sendWebhook(tunnelURL string) {
	payload := map[string]string{
		"content": fmt.Sprintf(":cd: **Vemenichy is Online!**\nAccess Dashboard: %s", tunnelURL),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Tunnel] Failed to marshal webhook payload: %v\n", err)
		return
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("[Tunnel] Failed to send webhook: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Println("[Tunnel] 🚀 Payload fired to Discord!")
	} else {
		log.Printf("[Tunnel] Webhook returned non-200 status: %d\n", resp.StatusCode)
	}
}
