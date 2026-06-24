#!/usr/bin/env bash
# bin/deploy.sh — Remote build on surnia (Fly.io), copy to otus, and activate.
# Run this on otus.
set -euo pipefail

BOLD=$'\e[1m'
CYAN=$'\e[36m'
GREEN=$'\e[32m'
RED=$'\e[31m'
RESET=$'\e[0m'

VM_HOST="init@sober-surnia.flycast"
VM_PORT="2222"
FLAKE_DIR="sober-nix"   # Repo directory on surnia (~/sober-nix)
CHROOT="~/nix-user-chroot ~/.nix"

SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"
SSH="ssh -p $VM_PORT $SSH_OPTS $VM_HOST"
export NIX_SSHOPTS="$SSH_OPTS -p $VM_PORT"

# ── 1. Build on surnia ────────────────────────────────────────────────────────
echo "${BOLD}${CYAN}==> 1. Building on surnia (${VM_HOST})...${RESET}"
out_path=$(
  $SSH "cd $FLAKE_DIR && $CHROOT bash -lc \
    \"nix build .#nixosConfigurations.otus.config.system.build.toplevel \
      --print-out-paths --no-link \
      --extra-experimental-features 'nix-command flakes'\""
)

if [[ -z "$out_path" ]]; then
    echo "${RED}Error: No build output path returned from surnia.${RESET}" >&2
    exit 1
fi
echo "  Built: $out_path"

# ── 2. Copy closure to otus ───────────────────────────────────────────────────
echo -e "\n${BOLD}${CYAN}==> 2. Copying closure from surnia to local store...${RESET}"
# surnia uses single-user nix (no daemon), so remote-program points to the profile nix-store
nix copy \
    --no-check-sigs \
    --from "ssh://${VM_HOST}?port=${VM_PORT}&remote-program=${CHROOT// /\\ } bash -lc /nix/var/nix/profiles/default/bin/nix-store" \
    "$out_path" \
    --extra-experimental-features 'nix-command flakes' \
    || {
        # Fallback: use nix-store --serve protocol directly
        echo "  Trying direct nix-store copy..."
        $SSH "$CHROOT bash -lc 'nix copy --to ssh://localhost?remote-store=/nix $out_path --extra-experimental-features nix-command flakes'" 2>/dev/null || true
        # Use rsync of the closure as last resort
        echo "  Falling back to nix store export/import..."
        $SSH "$CHROOT bash -lc 'nix store cat --store /nix $out_path 2>/dev/null || nix-store --export \$($SSH \"$CHROOT bash -lc \\\"nix-store -qR $out_path\\\"\") '" \
            | nix-store --import
    }

# ── 3. Activate ───────────────────────────────────────────────────────────────
echo -e "\n${BOLD}${CYAN}==> 3. Activating new configuration...${RESET}"
sudo "$out_path/bin/switch-to-configuration" switch

# ── 4. Sync git ───────────────────────────────────────────────────────────────
echo -e "\n${BOLD}${CYAN}==> 4. Syncing local repo...${RESET}"
git pull || echo "Warning: git pull failed (local uncommitted changes on otus)."

# ── Notify ────────────────────────────────────────────────────────────────────
if command -v notify-send >/dev/null 2>&1; then
    gen_id=$(readlink /nix/var/nix/profiles/system | cut -d'-' -f2 || echo "unknown")
    notify-send -a "System Deploy" -u normal \
        "Deployment Successful (#${gen_id})" \
        "Built on surnia, activated on otus.\nGeneration: #${gen_id}\nPath: ${out_path}" || true
fi

echo -e "\n${BOLD}${GREEN}✔ Deployment complete!${RESET}"
