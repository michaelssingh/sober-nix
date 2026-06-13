#!/usr/bin/env bash
# Soju Maintenance: Apply settings to all channels except ##init
set -euo pipefail

APP_NAME="sober-athene"
USER_NAME="init"

echo "Connecting to $APP_NAME and updating channel settings..."

# Execute a single bash command inside the container using fly ssh console
fly ssh console -a "$APP_NAME" -C "bash -c '
  sojuctl user run \"$USER_NAME\" channel status | while read -r line; do
    # Extract channel/network part
    parts=(\$line)
    chan=\"\${parts[0]}\"

    # Skip ##init channel (assuming ##init/libera)
    if [[ \"\$chan\" == *\"##init/\"* ]]; then
      echo \"Skipping \$chan...\"
      continue
    fi

    echo \"Updating \$chan...\"
    sojuctl user run \"$USER_NAME\" channel update \"\$chan\" -detached true -reattach-on message -detach-after 1h
  done
'"
