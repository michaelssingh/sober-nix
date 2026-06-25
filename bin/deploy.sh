#!/usr/bin/env bash
# bin/deploy.sh — Remote build on sprite.dev VM, copy to otus, and activate.
# Run this on otus.
set -euo pipefail

BOLD=$'\e[1m'
CYAN=$'\e[36m'
GREEN=$'\e[32m'
RED=$'\e[31m'
RESET=$'\e[0m'

VM_HOST="sprite@127.0.0.1"
VM_PORT="2222"
FLAKE_DIR="sober-nix" # Directory on the VM

# Configure SSH options to bypass host key prompts
SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"
export NIX_SSHOPTS="$SSH_OPTS"

echo "${BOLD}${CYAN}==> 1. Pulling latest changes on otus...${RESET}"
git pull || echo "Warning: git pull failed (you may have uncommitted local changes on otus)."

echo -e "\n${BOLD}${CYAN}==> 2. Triggering Nix build on remote VM ($VM_HOST:$VM_PORT)...${RESET}"
out_path=$(ssh -p "$VM_PORT" $SSH_OPTS "$VM_HOST" "cd \"$FLAKE_DIR\" && git pull && NIX_REMOTE=daemon nix build .#nixosConfigurations.otus.config.system.build.toplevel --print-out-paths --no-link --extra-experimental-features 'nix-command flakes'")

if [[ -z "$out_path" ]]; then
    echo "${RED}Error: Failed to obtain build path from VM.${RESET}" >&2
    exit 1
fi
echo "  Built: $out_path"

echo -e "\n${BOLD}${CYAN}==> 3. Copying built system ($out_path) from VM to local store...${RESET}"
NIX_REMOTE=daemon nix copy \
    --no-check-sigs \
    --from "ssh://$VM_HOST:$VM_PORT?remote-program=/nix/var/nix/profiles/default/bin/nix-store" \
    "$out_path" \
    --extra-experimental-features 'nix-command flakes'

echo -e "\n${BOLD}${CYAN}==> 4. Activating new configuration...${RESET}"
sudo nix-env --profile /nix/var/nix/profiles/system --set "$out_path"
sudo "$out_path/bin/switch-to-configuration" switch

# Step 5 intentionally omitted — git pull already done at step 1

# Send detailed notification on success if notify-send is available
if command -v notify-send >/dev/null 2>&1; then
    gen_id=$(readlink /nix/var/nix/profiles/system | cut -d'-' -f2 || echo "unknown")
    notify-send -a "System Deploy" -u normal \
        "Deployment Successful (Generation #$gen_id)" \
        "System successfully built, copied, and activated.\n\nGeneration: #$gen_id\nPath: $out_path\nUser: $(whoami)\nDate: $(date)" || true
fi

echo -e "\n${BOLD}${GREEN}✔ Deployment completed successfully!${RESET}"
