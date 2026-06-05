#!/usr/bin/env bash
# Soju Maintenance: Attach all detached channels for a given user
set -euo pipefail

APP_NAME="sober-athene"
USER_NAME="init"

echo "Connecting to $APP_NAME and attaching all detached channels..."

# Execute a single bash command inside the container using fly ssh console
fly ssh console -a "$APP_NAME" -C "bash -c '
  # List channels and attach them using default config path (/etc/soju/config)
  sojuctl user run \"$USER_NAME\" channel status | while read -r line; do
    if [[ \"\$line\" == *detached* ]]; then
      parts=(\$line)
      chan=\"\${parts[0]}\"
      echo \"Attaching \$chan...\"
      sojuctl user run \"$USER_NAME\" channel update \"\$chan\" -detached false
    fi
  done
'"
