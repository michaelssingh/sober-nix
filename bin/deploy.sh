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

# Configure SSH options to bypass host key prompts and fail fast on timeouts
SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=10"
export NIX_SSHOPTS="$SSH_OPTS"

echo "${BOLD}${CYAN}==> 1. Pulling latest changes on otus...${RESET}"
git_out=$(git pull 2>&1 || echo "Warning: git pull failed")
echo "$git_out"

if echo "$git_out" | grep -q "bin/deploy.sh"; then
	echo "  [deploy] Script updated on disk. Re-executing..."
	exec "$0" "$@"
fi

echo -e "\n${BOLD}${CYAN}==> 2. Triggering Nix build on remote VM ($VM_HOST:$VM_PORT) and pushing to Cachix...${RESET}"
# Saves the raw built path to a temporary file on the worker, outputs it cleanly, and silences Cachix stdout
raw_output=$(ssh -p "$VM_PORT" $SSH_OPTS "$VM_HOST" "export SSH_AUTH_SOCK=/home/sprite/.ssh-agent.sock && cd \"$FLAKE_DIR\" && git pull >&2 && env PATH=/home/sprite/.nix-profile/bin:\$PATH NIX_REMOTE=daemon GOTELEMETRY=off GODEBUG=telemetry=off nix build .#nixosConfigurations.otus.config.system.build.toplevel --print-out-paths --no-link --extra-experimental-features 'nix-command flakes' > /tmp/build_path.txt && cat /tmp/build_path.txt && cat /tmp/build_path.txt | xargs -r env PATH=/home/sprite/.nix-profile/bin:\$PATH NIX_REMOTE=daemon cachix push sober-nix >&2")

# Extract the nix store path robustly and strip all trailing spaces/newlines
out_path=$(echo "$raw_output" | grep -E '^/nix/store/' | head -n 1 | tr -d '[:space:]' || true)

if [[ -z "$out_path" ]]; then
    echo "${RED}Error: Failed to obtain build path from VM.${RESET}" >&2
    echo "Raw remote output was:" >&2
    echo "$raw_output" >&2
    exit 1
fi
echo "  Built: $out_path"

echo -e "\n${BOLD}${CYAN}==> 3. Copying built system ($out_path) from VM to local store...${RESET}"
# Direct execution path to the single-user nix binary on the remote box to entirely sidestep the remote shell environment.
NIX_REMOTE=daemon nix copy \
    --no-check-sigs \
    --from "ssh://$VM_HOST:$VM_PORT?remote-program=/home/sprite/.nix-profile/bin/nix-store" \
    "$out_path" \
    --extra-experimental-features 'nix-command flakes'

echo -e "\n${BOLD}${CYAN}==> 4. Activating new configuration...${RESET}"
old_path=$(readlink -f /nix/var/nix/profiles/system 2>/dev/null || true)
sudo nix-env --profile /nix/var/nix/profiles/system --set "$out_path"
sudo "$out_path/bin/switch-to-configuration" switch

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
        # Strip ANSI escape sequences
        clean_diff=$(echo "$diff_output" | sed 's/\x1b\[[0-9;]*[a-zA-Z]//g')
        formatted_diff=$(echo "$clean_diff" | sed 's/^/  • /' | head -n 12)
        diff_section="\n\nPackage Changes:\n$formatted_diff"
        if [[ $(echo "$clean_diff" | wc -l) -gt 12 ]]; then
            diff_section="$diff_section\n  • ... (and more)"
        fi
    else
        diff_section="\n\nPackage Changes: None"
    fi

    notify-send -t 0 -a "System Deploy" -u normal \
        "Deployment Successful (Generation #$gen_id)" \
        "System successfully built, copied, and activated.\n\nGeneration: #$gen_id\nSystem ID: $system_id ($sys_size)\nDuration: $duration_str\nCommit: $commit_hash - $commit_msg\nPath: $out_path\nUser: $(whoami)\nDate: $(date)$diff_section" || true
fi

echo -e "\n${BOLD}${GREEN}✔ Deployment completed successfully!${RESET}"
