#!/bin/sh

# Initialize persistent store if empty
if [ -z "$(ls -A /var/lib/nix_persist)" ]; then
    echo "Initializing persistent Nix store..."
    cp -ra /nix/* /var/lib/nix_persist/
fi

# Bind mount the persistent volume to the active Nix store
mount --bind /var/lib/nix_persist /nix

# Start the Nix daemon in the background
/nix/var/nix/profiles/default/bin/nix-daemon &

# Start SSH daemon in the foreground
exec /root/.nix-profile/bin/sshd -D -e
