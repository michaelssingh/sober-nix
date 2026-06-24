#!/bin/bash
# --- surnia Bootstrapper ---
# Idempotent. Run on sober-surnia.flycast as `init`.
# Sets up: nix-user-chroot → Nix → repo clone → home-manager
set -euo pipefail

NIX_CHROOT="$HOME/nix-user-chroot"
NIX_STORE="$HOME/.nix"
REPO="git@sober-bubo.flycast:init/sober-nix.git"
REPO_DIR="$HOME/sober-nix"
FLAKE_URI="github:michaelssingh/sober-nix#init@surnia"
NIX_USER_CHROOT_URL="https://github.com/nix-community/nix-user-chroot/releases/download/1.2.2/nix-user-chroot-bin-1.2.2-x86_64-unknown-linux-musl"

GREEN='\033[0;32m'; CYAN='\033[0;36m'; RESET='\033[0m'
step() { echo -e "\n${CYAN}▶ $*${RESET}"; }
ok()   { echo -e "${GREEN}✓ $*${RESET}"; }

step "Starting surnia bootstrap..."

# ── 1. nix-user-chroot ────────────────────────────────────────────────────────
if [ ! -f "$NIX_CHROOT" ]; then
    step "Downloading nix-user-chroot..."
    curl -L -o "$NIX_CHROOT" "$NIX_USER_CHROOT_URL"
    chmod +x "$NIX_CHROOT"
fi
ok "nix-user-chroot ready"

# ── 2. Nix store ─────────────────────────────────────────────────────────────
if [ ! -d "$NIX_STORE/store" ]; then
    step "Installing Nix (single-user) inside chroot..."
    mkdir -p "$NIX_STORE"
    "$NIX_CHROOT" "$NIX_STORE" bash -c \
        "curl -L https://nixos.org/nix/install | sh -s -- --no-daemon --yes"
    ok "Nix installed"
else
    ok "Nix store already present"
fi

# ── 3. Shell config (host-level, outside chroot) ─────────────────────────────
step "Writing shell config..."
cat > "$HOME/.bashrc" << 'EOF'
# surnia .bashrc
export PS1='\[\033[01;35m\]\u@surnia\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\]\$ '
export HISTSIZE=5000 HISTFILESIZE=10000 HISTCONTROL=ignoreboth:erasedups
shopt -s histappend
alias ls='ls --color=auto' ll='ls -lh' la='ls -A'

# Enter Nix chroot environment
alias nix-env='~/nix-user-chroot ~/.nix bash -lc'
alias hack='~/nix-user-chroot ~/.nix bash -l'

# Re-apply home-manager config
alias hms='~/nix-user-chroot ~/.nix bash -lc "home-manager switch --flake github:michaelssingh/sober-nix#init@surnia --extra-experimental-features '"'"'nix-command flakes'"'"' --refresh"'

# Nix commands via chroot wrapper
nix()  { ~/nix-user-chroot ~/.nix bash -lc "nix $*"; }
export -f nix
EOF
cat > "$HOME/.bash_profile" << 'EOF'
[ -f ~/.bashrc ] && . ~/.bashrc
EOF
ok "Shell config written"

# ── 4. SSH agent socket (for git + nix copy auth) ────────────────────────────
step "Configuring SSH agent forwarding socket..."
# On surnia, the agent is forwarded from otus via socat on port 9000
cat >> "$HOME/.bashrc" << 'EOF'

# SSH agent forwarding from otus (via sprite-tunnel reverse port)
if [ -z "${SSH_AUTH_SOCK:-}" ] || [ ! -S "${SSH_AUTH_SOCK:-}" ]; then
    _agent_sock="$HOME/.ssh-agent.sock"
    if [ ! -S "$_agent_sock" ]; then
        socat UNIX-LISTEN:"$_agent_sock",fork TCP:127.0.0.1:9000 &>/dev/null &
        disown
    fi
    export SSH_AUTH_SOCK="$_agent_sock"
fi
EOF
ok "Agent socket configured"

# ── 5. Clone / update repo ────────────────────────────────────────────────────
step "Setting up sober-nix repo..."
if [ ! -d "$REPO_DIR/.git" ]; then
    # Try forge first, fall back to GitHub
    if ssh -o StrictHostKeyChecking=no -o ConnectTimeout=5 -p 2222 init@sober-bubo.flycast echo ok &>/dev/null 2>&1; then
        git clone "$REPO" "$REPO_DIR" || git clone "https://github.com/michaelssingh/sober-nix.git" "$REPO_DIR"
    else
        git clone "https://github.com/michaelssingh/sober-nix.git" "$REPO_DIR"
    fi
    ok "Repo cloned to $REPO_DIR"
else
    git -C "$REPO_DIR" pull --ff-only || true
    ok "Repo up to date"
fi

# ── 6. Home Manager via chroot ────────────────────────────────────────────────
step "Applying Home Manager config (may take 10-15 min on first run)..."
"$NIX_CHROOT" "$NIX_STORE" bash -lc "
    set -euo pipefail
    # Source nix profile (chroot maps NIX_STORE → /nix)
    [ -f \"\$HOME/.nix-profile/etc/profile.d/nix.sh\" ] && source \"\$HOME/.nix-profile/etc/profile.d/nix.sh\"
    echo \"  Nix: \$(nix --version)\"
    nix run \
        --extra-experimental-features 'nix-command flakes' \
        github:nix-community/home-manager/release-26.05 \
        -- switch \
        --flake '$FLAKE_URI' \
        --extra-experimental-features 'nix-command flakes' \
        -b backup --refresh
"

ok "Home Manager applied"

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
echo -e "${GREEN}✓ surnia is ready for remote development!${RESET}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
echo ""
echo "  Repo:   ~/sober-nix"
echo "  Build:  cd ~/sober-nix && nix build .#nixosConfigurations.otus.config.system.build.toplevel"
echo "  Deploy: On otus, run:  ./bin/deploy.sh"
echo ""
echo "  Re-login or: exec bash -l"
