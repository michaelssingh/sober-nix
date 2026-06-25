#!/usr/bin/env bash
# bin/deploy.sh — Remote build on sprite.dev VM, copy to otus, and activate.
# Run this on otus.
set -euo pipefail

start_time=$(date +%s)
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
out_path=$(ssh -p "$VM_PORT" $SSH_OPTS "$VM_HOST" "export SSH_AUTH_SOCK=/home/sprite/.ssh-agent.sock && cd \"$FLAKE_DIR\" && git pull >&2 && NIX_REMOTE=daemon nix build .#nixosConfigurations.otus.config.system.build.toplevel --print-out-paths --no-link --extra-experimental-features 'nix-command flakes'")

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
old_path=$(readlink -f /nix/var/nix/profiles/system 2>/dev/null || true)
sudo nix-env --profile /nix/var/nix/profiles/system --set "$out_path"
sudo "$out_path/bin/switch-to-configuration" switch

# Step 5 intentionally omitted — git pull already done at step 1

# Send detailed notification on success if notify-send is available
if command -v notify-send >/dev/null 2>&1; then
    end_time=$(date +%s)
    duration=$((end_time - start_time))
    if [[ $duration -ge 60 ]]; then
        duration_str="$((duration / 60))m $((duration % 60))s"
    else
        duration_str="${duration}s"
    fi

    gen_id=$(readlink /nix/var/nix/profiles/system | cut -d'-' -f2 || echo "unknown")
    system_id=$(basename "$out_path" | cut -d'-' -f1 || echo "unknown")
    commit_hash=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    commit_msg=$(git log -1 --pretty=%s 2>/dev/null || echo "unknown")

    sys_size=$(nix path-info -Sh "$out_path" --extra-experimental-features 'nix-command' 2>/dev/null | awk '{print $2, $3}' || echo "unknown")

    if [[ -n "${old_path:-}" && "$old_path" != "$out_path" ]]; then
        diff_output=$(nix store diff-closures "$old_path" "$out_path" --extra-experimental-features 'nix-command' 2>/dev/null || echo "")
    else
        diff_output=""
    fi

    if [[ -n "$diff_output" ]]; then
        formatted_diff=$(echo "$diff_output" | sed 's/^/  • /' | head -n 12)
        diff_section="\n\nPackage Changes:\n$formatted_diff"
        if [[ $(echo "$diff_output" | wc -l) -gt 12 ]]; then
            diff_section="$diff_section\n  • ... (and more)"
        fi
    else
        diff_section="\n\nPackage Changes: None"
    fi

    notify-send -a "System Deploy" -u normal \
        "Deployment Successful (Generation #$gen_id)" \
        "System successfully built, copied, and activated.\n\nGeneration: #$gen_id\nSystem ID: $system_id ($sys_size)\nDuration: $duration_str\nCommit: $commit_hash - $commit_msg\nPath: $out_path\nUser: $(whoami)\nDate: $(date)$diff_section" || true
fi

echo -e "\n${BOLD}${GREEN}✔ Deployment completed successfully!${RESET}"
