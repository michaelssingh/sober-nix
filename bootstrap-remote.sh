#!/bin/bash
# --- hashnix.club Bootstrapper ---
set -euo pipefail

NIX_CHROOT="$HOME/nix-user-chroot"
NIX_STORE="$HOME/.nix"
FLAKE_URI="github:michaelssingh/sober-nix#init@hashnix"

echo "⠋ Starting restoration of hashnix environment..."

# 1. Ensure nix-user-chroot exists
if [ ! -f "$NIX_CHROOT" ]; then
    echo "⠋ Downloading nix-user-chroot..."
    curl -L -s https://api.github.com/repos/nix-community/nix-user-chroot/releases/latest \
    | grep "browser_download_url" | grep "x86_64-unknown-linux-musl" \
    | cut -d '\"' -f 4 | xargs curl -L -o "$NIX_CHROOT"
    chmod +x "$NIX_CHROOT"
fi

# 2. Check Nix Store
if [ ! -d "$NIX_STORE" ]; then
    echo "⚠ Nix store missing. Re-installing..."
    curl -L https://nixos.org/nix/install | sh -s -- --no-daemon --yes
fi

# 3. Restore Shell Configurations
echo "⠋ Restoring shell configurations..."
cat << 'BASHRC' > "$HOME/.bashrc"
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
alias chat='tmux -L host -f ~/.tmux-host.conf new-session -A -s host -n dream "dream"'
if [ -f /etc/bash_completion ]; then . /etc/bash_completion; fi
BASHRC

cat << 'BASHPROF' > "$HOME/.bash_profile"
if [ -f ~/.bashrc ]; then . ~/.bashrc; fi
if [[ \$- == *i* ]] && [ -z "\$TMUX" ]; then
  tmux -L host -f ~/.tmux-host.conf has-session -t host 2>/dev/null || tmux -L host -f ~/.tmux-host.conf new-session -d -s host -n dream 'dream'
fi
BASHPROF

# 4. Re-link Environment via Home Manager
echo "⠋ Re-applying Home Manager configuration..."
# Find any functional nix binary in the existing store
NIX_BIN=\$(find "\$NIX_STORE/store" -maxdepth 3 -name nix -type f -executable | head -n 1)

"\$NIX_CHROOT" "\$NIX_STORE" bash -l -c "
    \$NIX_BIN run --extra-experimental-features 'nix-command flakes' github:nix-community/home-manager/master -- switch --flake \$FLAKE_URI --extra-experimental-features 'nix-command flakes' -b backup --refresh
"

echo "✓ Environment successfully restored!"
