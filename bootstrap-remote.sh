#!/bin/bash
# --- surnia Bootstrapper ---
# Run this as `init` on sober-surnia.flycast after SSH in.
set -euo pipefail

NIX_CHROOT="$HOME/nix-user-chroot"
NIX_STORE="$HOME/.nix"
FLAKE_URI="github:michaelssingh/sober-nix#init@surnia"
# Pin a known-good release to avoid flaky GitHub API parsing
NIX_USER_CHROOT_URL="https://github.com/nix-community/nix-user-chroot/releases/download/1.2.2/nix-user-chroot-bin-1.2.2-x86_64-unknown-linux-musl"

echo "⠋ Starting restoration of surnia environment..."

# 1. Ensure nix-user-chroot exists
if [ ! -f "$NIX_CHROOT" ]; then
    echo "⠋ Downloading nix-user-chroot..."
    curl -L -o "$NIX_CHROOT" "$NIX_USER_CHROOT_URL"
    chmod +x "$NIX_CHROOT"
fi
echo "✓ nix-user-chroot: $("$NIX_CHROOT" --version 2>&1 || echo ok)"

# 2. Check Nix Store
if [ ! -d "$NIX_STORE/store" ]; then
    echo "⠋ Nix store missing. Installing inside chroot (this takes a few minutes)..."
    mkdir -p "$NIX_STORE"
    "$NIX_CHROOT" "$NIX_STORE" bash -c "curl -L https://nixos.org/nix/install | sh -s -- --no-daemon --yes"
    echo "✓ Nix installed."
else
    echo "✓ Nix store found."
fi

# 3. Write baseline shell config
echo "⠋ Writing baseline shell config..."
cat << 'BASHRC_EOF' > "$HOME/.bashrc"
export CLICOLOR=1
alias ls='ls --color=auto'
alias ll='ls -lh'
alias la='ls -A'
HISTSIZE=5000
HISTFILESIZE=10000
HISTCONTROL=ignoreboth:erasedups
shopt -s histappend
PS1='\[\033[01;32m\]\u@\h\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\]\$ '
alias hack='~/nix-user-chroot ~/.nix bash -l -c fish'
alias hms="~/nix-user-chroot ~/.nix bash -l -c \"home-manager switch --flake github:michaelssingh/sober-nix#init@surnia --extra-experimental-features 'nix-command flakes' --refresh\""
if [ -f /etc/bash_completion ]; then . /etc/bash_completion; fi
BASHRC_EOF

cat << 'BASHPROF_EOF' > "$HOME/.bash_profile"
if [ -f ~/.bashrc ]; then . ~/.bashrc; fi
BASHPROF_EOF

# 4. Apply Home Manager inside the chroot
# nix-user-chroot maps NIX_STORE -> /nix, so ~/.nix-profile paths work correctly inside.
echo "⠋ Re-applying Home Manager configuration (this may take 10+ minutes on first run)..."
"$NIX_CHROOT" "$NIX_STORE" bash -lc "
    set -euo pipefail

    # Source Nix profile - standard paths work inside the chroot
    if [ -f \"\$HOME/.nix-profile/etc/profile.d/nix.sh\" ]; then
        source \"\$HOME/.nix-profile/etc/profile.d/nix.sh\"
    elif [ -f /nix/var/nix/profiles/default/etc/profile.d/nix-daemon.sh ]; then
        source /nix/var/nix/profiles/default/etc/profile.d/nix-daemon.sh
    fi

    echo \"  Nix: \$(nix --version)\"

    nix run \
        --extra-experimental-features 'nix-command flakes' \
        github:nix-community/home-manager/release-26.05 \
        -- switch \
        --flake '$FLAKE_URI' \
        --extra-experimental-features 'nix-command flakes' \
        -b backup \
        --refresh
"

echo ""
echo "✓ surnia environment bootstrapped!"
echo "  Re-login or run: exec bash -l"
