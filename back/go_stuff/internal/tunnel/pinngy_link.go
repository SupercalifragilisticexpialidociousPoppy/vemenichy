package tunnel

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

func Start() {
	fmt.Println("🚇 Booting up global Pinggy tunnel...")

	// We include the bypass flags here so it NEVER asks for your SSH passphrase
	// and automatically accepts the host key. Perfect for headless Pi booting!
	cmd := exec.Command("ssh",
		"-p", "443",
		"-R0:localhost:8080",
		"-o", "StrictHostKeyChecking=no",
		"-o", "PubkeyAuthentication=no",
		"a.pinggy.io",
	)

	// 1. Attach a pipe to listen to what SSH prints to the terminal
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("🚨 Tunnel pipe failed: %v\n", err)
		return
	}

	// 2. Start the command (Start runs it in the background, unlike Run/CombinedOutput)
	if err := cmd.Start(); err != nil {
		fmt.Printf("🚨 Failed to start tunnel: %v\n", err)
		return
	}

	// 3. Launch a goroutine to scan the output text while the server keeps booting
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()

			// Pinggy prints a lot of ASCII art, we only care about the actual URLs
			if strings.Contains(line, "http://") || strings.Contains(line, "https://") {
				// Strip whitespace and print it cleanly
				fmt.Printf("🌍 GLOBAL LINK SECURED: %s\n", strings.TrimSpace(line))
			}
		}

		// If the loop finishes, it means the tunnel collapsed
		fmt.Println("⚠️ Global tunnel closed.")
	}()
}
