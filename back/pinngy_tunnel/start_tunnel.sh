#!/bin/bash

# Configuration
WEBHOOK_URL="https://discord.com/api/webhooks/1488089624344530956/dLLgsTx1ct-aVLskx7ci950CLT5cle2HTQKLa0rUMs5saXo7XgPTP8WYKln1oBiIogsL"
TOKEN="zH8Zzzraxhu"

echo "Starting Pinggy Tunnel..."

# Open the tunnel using the token. 
# ServerAliveInterval keeps the connection from falling asleep!
ssh -o StrictHostKeyChecking=no -o ServerAliveInterval=30 -p 443 -R0:localhost:8080 $TOKEN@a.pinggy.io 2>&1 | while read -r line; do
    
    # Print the invisible background logs to the terminal just in case we need to debug
    echo "$line"
    
    # Look for the line containing the HTTP link
    if [[ "$line" == *"http://"*".pinggy.link"* ]]; then
        
        # Slice out ONLY the URL from the text block
        TUNNEL_URL=$(echo "$line" | grep -o 'http://[^ ]*')
        
        echo "Successfully caught URL: $TUNNEL_URL"
        
        # Fire the payload to Discord
        curl -H "Content-Type: application/json" \
             -d "{\"content\": \"🎧 **Vemenichy is Online!**\nAccess Dashboard: $TUNNEL_URL\"}" \
             $WEBHOOK_URL
             
        # Break the loop so it stops reading lines, but leaves the tunnel running!
        break
    fi
done