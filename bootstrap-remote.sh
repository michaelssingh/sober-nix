#!/bin/bash
# --- surnia Bootstrapper ---
# Run this as `init` on sober-surnia.flycast after SSH in.
set -euo pipefail

NIX_CHROOT="$HOME/nix-user-chroot"
NIX_STORE="$HOME/.nix"
FLAKE_URI="github:michaelssingh/sober-nix#init@surnia"

echo "⠋ Starting restoration of surnia environment..."

# 1. Ensure nix-user-chroot exists
if [ ! -f "$NIX_CHROOT" ]; then
    echo "⠋ Downloading nix-user-chroot..."
    curl -L -s https://api.github.com/repos/nix-community/nix-user-chroot/releases/latest \
    | grep "browser_download_url" | grep "x86_64-unknown-linux-musl" \
    | cut -d '"' -f 4 | xargs curl -L -o "$NIX_CHROOT"
    chmod +x "$NIX_CHROOT"
fi

# 2. Check Nix Store
if [ ! -d "$NIX_STORE" ]; then
    echo "⚠ Nix store missing. Re-installing inside chroot..."
    mkdir -p "$NIX_STORE"
    "$NIX_CHROOT" "$NIX_STORE" bash -c "curl -L https://nixos.org/nix/install | sh -s -- --no-daemon --yes"
fi

# 3. Restore Shell Configurations (outside nix, as a fallback)
echo "⠋ Writing baseline shell config..."
cat << 'BASHRC_EOF' > "$HOME/.bashrc"
export CLICOLOR=1
export LSCOLORS=GxFxCxDxBxegedabagaced
alias ls='ls --color=auto'
alias ll='ls -lh'
alias la='ls -A'
alias grep='grep --color=auto'
HISTSIZE=5000
HISTFILESIZE=10000
HISTCONTROL=ignoreboth:erasedups
shopt -s histappend
PS1='\[\033[01;32m\]\u@\h\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\]\$ '
alias hack='~/nix-user-chroot ~/.nix bash -l -c fish'
alias hms="~/nix-user-chroot ~/.nix bash -l -c \"home-manager switch --flake github:michaelssingh/sober-nix#init@surnia --extra-experimental-features 'nix-command flakes' --refresh\""
alias chat='tmux -L host -f ~/.tmux-host.conf new-session -A -s host -n dream "dream"'
if [ -f /etc/bash_completion ]; then . /etc/bash_completion; fi
BASHRC_EOF

cat << 'BASHPROF_EOF' > "$HOME/.bash_profile"
if [ -f ~/.bashrc ]; then . ~/.bashrc; fi
if [[ $- == *i* ]] && [ -z "$TMUX" ]; then
  tmux -L host -f ~/.tmux-host.conf has-session -t host 2>/dev/null || tmux -L host -f ~/.tmux-host.conf new-session -d -s host -n dream 'dream'
fi
BASHPROF_EOF

# 4. Re-link Environment via Home Manager inside the chroot
# The key: source the nix profile INSIDE the chroot before calling home-manager.
# nix-user-chroot maps NIX_STORE to /nix inside the container, so the standard
# profile paths (~/.nix-profile, /nix/var/nix/profiles/default) work correctly.
echo "⠋ Re-applying Home Manager configuration via chroot..."
"$NIX_CHROOT" "$NIX_STORE" bash -lc "
    set -euo pipefail

    # Source Nix profile (the chroot maps NIX_STORE -> /nix, so standard paths work)
    if [ -f ~/.nix-profile/etc/profile.d/nix.sh ]; then
        source ~/.nix-profile/etc/profile.d/nix.sh
    elif [ -f /nix/var/nix/profiles/default/etc/profile.d/nix-daemon.sh ]; then
        source /nix/var/nix/profiles/default/etc/profile.d/nix-daemon.sh
    fi

    echo \"  Nix version: \$(nix --version)\"

    # Apply home-manager config
    nix run \\
        --extra-experimental-features 'nix-command flakes' \\
        github:nix-community/home-manager/release-26.05 \\
        -- switch \\
        --flake '$FLAKE_URI' \\
        --extra-experimental-features 'nix-command flakes' \\
        -b backup \\
        --refresh
"

echo "✓ Environment successfully restored!"
echo ""
echo "Run 'exec bash -l' or re-login to enter the managed environment."
