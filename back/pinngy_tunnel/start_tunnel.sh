#!/bin/bash

# 1. Dynamically load the .env file
ENV_FILE="$HOME/vemenichy/back/go_stuff/.env"

if [ -f "$ENV_FILE" ]; then
    export $(grep -v '^#' "$ENV_FILE" | xargs)
else
    echo "❌ Cannot find .env file at $ENV_FILE"
    exit 1
fi

# 2. Define the Cleanup Function (The "Offline" Webhook)
cleanup() {
    trap - EXIT SIGINT SIGTERM

    echo -e "\n🛑 Script terminating! Sending offline webhook..."

    curl -s -H "Content-Type: application/json" \
         -d "{\"content\": \":headstone: Vemenichy Global is offline now.\nThe Pinggy tunnel is closed.\"}" \
         $webhookURL

    # Ensure the background SSH process is actually killed,
    # just in case the user hit Ctrl+C to stop the script.
    pkill -f "$PINGGY_TOKEN@a.pinggy.io"
    echo "Done. Goodbye!"
    exit 0
}

# 3. SET THE TRAP
# This watches for EXIT (natural script end), SIGINT (Ctrl+C), and SIGTERM (kill command)
trap cleanup EXIT SIGINT SIGTERM

echo "Starting Pinggy Tunnel (TUI Disabled)..."

# 4. Use a flag to ensure we don't spam the "Online" webhook
WEBHOOK_SENT=0

# Removed 'break' from the loop so it intentionally hangs and monitors!
ssh -T -o StrictHostKeyChecking=no -o ServerAliveInterval=30 -p 443 -R0:localhost:8080 $sshToken@a.pinggy.io 2>&1 | while read -r line; do

    echo "Pinggy: $line"

    if [[ "$line" == *"pinggy-free.link"* ]] && [ $WEBHOOK_SENT -eq 0 ]; then

        TUNNEL_URL=$(echo "$line" | grep -oE 'https://[a-zA-Z0-9.-]+\.pinggy-free\.link' | head -n 1 | tr -d '\r' | tr -d '\n')

        if [ ! -z "$TUNNEL_URL" ]; then
            echo "✅ Successfully caught clean URL: $TUNNEL_URL"

            curl -s -H "Content-Type: application/json" \
                 -d "{\"content\": \":cd: **Vemenichy Global is Online!**\nAccess Dashboard: $TUNNEL_URL\nExpires in 1 hour unless killed manually.\"}" \
                 $webhookURL

            echo "🚀 Payload fired! Script is now HANGING to monitor tunnel state..."

            # Set flag to 1 so we don't send the "Online" payload again
            WEBHOOK_SENT=1
        fi
    fi
done
