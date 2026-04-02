#!/bin/bash

# Configuration
WEBHOOK_URL="https://discord.com/api/webhooks/1488089624344530956/dLLgsTx1ct-aVLskx7ci950CLT5cle2HTQKLa0rUMs5saXo7XgPTP8WYKln1oBiIogsL"
TOKEN="zH8Zzzraxhu"

echo "Starting Pinggy Tunnel..."

# We keep -tt to force the output, but we use a smarter catcher
ssh -tt -o StrictHostKeyChecking=no -o ServerAliveInterval=30 -p 443 -R0:localhost:8080 $TOKEN@a.pinggy.io 2>&1 | tr '`' '\n' | while read -r line; do

    # Check if the line contains a pinggy link
    if [[ "$line" == *"pinggy.link"* ]]; then

        # 🚨 THE FIX: Strict Regex. Only extract exactly 'http://[anything].pinggy.link'
        # We also use 'tr' to delete any invisible carriage returns that break JSON
        TUNNEL_URL=$(echo "$line" | grep -oE 'https?://[a-zA-Z0-9.-]+\.pinggy\.link' | head -n 1 | tr -d '\r' | tr -d '\n')

        # If we successfully grabbed a clean URL, fire it
        if [ ! -z "$TUNNEL_URL" ]; then
            echo "✅ Successfully caught clean URL: $TUNNEL_URL"

            # Fire the payload to Discord
            curl -s -H "Content-Type: application/json" \
                 -d "{\"content\": \":cd: **Vemenichy is Online!**\nAccess Dashboard: $TUNNEL_URL\"}" \
                 $WEBHOOK_URL

            echo "🚀 Payload fired to Discord! Leaving tunnel open in background."
            break
        fi
    fi
done