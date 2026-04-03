package tunnel

import (
	"bufio"
	"os"
	"os/exec"
	"vemenichy-server/internal/player"
)

var (
	tunnelCmd    *exec.Cmd
	tunnelActive bool
)

// StartTunnel runs the shell script and pipes output to WebLog
func StartTunnel() error {
	if tunnelActive {
		return nil // Already running
	}

	// Make sure the path matches where you run vemenichy-bin from!
	tunnelCmd = exec.Command("bash", "../pinngy_tunnel/start_tunnel.sh")

	// Capture output
	stdout, _ := tunnelCmd.StdoutPipe()
	tunnelCmd.Stderr = tunnelCmd.Stdout

	if err := tunnelCmd.Start(); err != nil {
		return err
	}
	tunnelActive = true

	// Read the bash script output live and push to WebLog
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			player.WebLog("[Global] %s", scanner.Text())
		}

		// When the script dies (expires after 1 hour, or manually killed)
		tunnelCmd.Wait()
		tunnelActive = false
		player.WebLog("[Global] Tunnel offline.")
	}()

	return nil
}

// StopTunnel sends the Ctrl+C signal
func StopTunnel() {
	if tunnelCmd != nil && tunnelCmd.Process != nil && tunnelActive {
		// os.Interrupt is the exact equivalent of hitting Ctrl+C
		tunnelCmd.Process.Signal(os.Interrupt)
	}
}

func IsActive() bool {
	return tunnelActive
}
