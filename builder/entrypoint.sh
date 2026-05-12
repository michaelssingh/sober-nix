#!/usr/bin/env bash
set -e

echo "🚀 Starting Fedora-based Nix Builder..."

# 1. Setup Persistent Storage (OverlayFS for Nix)
# We use the /nix from the image as the base, and store changes in the volume.
mkdir -p /var/lib/nix_persist/nix_upper /var/lib/nix_persist/nix_work
# We must move the existing /nix content to a temporary location if the volume is empty,
# or just overlay directly if we want to keep the image's nix as the lower layer.
echo "📦 Mounting persistent Nix store..."
mount -t overlay overlay -o lowerdir=/nix,upperdir=/var/lib/nix_persist/nix_upper,workdir=/var/lib/nix_persist/nix_work /nix

# 2. Setup Authorized Keys for Root
mkdir -p /root/.ssh
echo "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAPGyqlLfLc3PTAQ00M2fg4kaEnoOkmMfECNGOQo/2FI" > /root/.ssh/authorized_keys
chmod 700 /root/.ssh
chmod 600 /root/.ssh/authorized_keys

# 2. Persistent SSH Host Keys
# Fly microVMs lose files in /etc on restart. We persist keys to our volume.
mkdir -p /var/lib/nix_persist/ssh
if [[ "$(ls /var/lib/nix_persist/ssh/*_key 2>/dev/null)" = "" ]]; then
    echo "🔑 Generating new host keys on persistent volume..."
    ssh-keygen -A
    cp /etc/ssh/ssh_host_*_key* /var/lib/nix_persist/ssh/
else
    echo "🔑 Restoring host keys from persistent volume..."
    cp -f /var/lib/nix_persist/ssh/ssh_host_*_key* /etc/ssh/
    chmod 600 /etc/ssh/ssh_host_*_key
fi

# 3. Ensure Nix environment is available
if [ -f /nix/var/nix/profiles/default/etc/profile.d/nix-daemon.sh ]; then
    . /nix/var/nix/profiles/default/etc/profile.d/nix-daemon.sh
fi

# 4. Start Nix Daemon in background
echo "❄️ Starting Nix Daemon..."
/nix/var/nix/profiles/default/bin/nix-daemon &

# 5. Start SSHD in foreground
echo "📡 Starting SSHD on port 2222 (mapped to 22)..."
exec /usr/sbin/sshd -D -e
