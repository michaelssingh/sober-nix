#!/usr/bin/env bash
set -euo pipefail

echo "=========================================="
echo "   Ninox Automated NixOS Installer"
echo "=========================================="

TARGET_DISK="${1:-/dev/nvme0n1}"
echo "Target Disk: $TARGET_DISK"

if [ ! -b "$TARGET_DISK" ]; then
  echo "Error: Block device $TARGET_DISK does not exist!"
  exit 1
fi

echo "WARNING: All data on $TARGET_DISK will be WIPED!"
read -p "Continue with installation? (y/N) " -n 1 -r < /dev/tty
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
  echo "Installation cancelled."
  exit 1
fi

echo "[0/6] Cleaning up existing mounts and mappings..."
umount -R /mnt 2>/dev/null || true
cryptsetup close crypted 2>/dev/null || true
swapoff -a 2>/dev/null || true

echo "[1/6] Partitioning $TARGET_DISK..."
sgdisk -Z "$TARGET_DISK"
sgdisk -n 1:0:+1G -t 1:ef00 -c 1:ESP "$TARGET_DISK"
sgdisk -n 2:0:0 -t 2:8309 -c 2:luks "$TARGET_DISK"
partprobe "$TARGET_DISK" 2>/dev/null || udevadm settle 2>/dev/null || sleep 2

PART1="${TARGET_DISK}p1"
PART2="${TARGET_DISK}p2"
if [ ! -b "$PART1" ]; then
  PART1="${TARGET_DISK}1"
  PART2="${TARGET_DISK}2"
fi

echo "[2/6] Formatting EFI System Partition..."
mkfs.vfat -F32 -n ESP "$PART1"

echo "[3/6] Setting up LUKS Encryption..."
cryptsetup luksFormat "$PART2"
dmsetup remove -f crypted 2>/dev/null || cryptsetup close crypted 2>/dev/null || true
udevadm settle 2>/dev/null || sleep 1
cryptsetup open "$PART2" crypted

echo "[4/6] Formatting Btrfs & Subvolumes..."
mkfs.btrfs -f -L ninox /dev/mapper/crypted
mount /dev/mapper/crypted /mnt
btrfs subvolume create /mnt/@root
btrfs subvolume create /mnt/@home
btrfs subvolume create /mnt/@nix
umount /mnt

echo "[5/6] Mounting Target Filesystems to /mnt..."
mount -o compress=zstd,noatime,subvol=@root /dev/mapper/crypted /mnt
mkdir -p /mnt/{boot,home,nix}
mount "$PART1" /mnt/boot
mount -o compress=zstd,noatime,subvol=@home /dev/mapper/crypted /mnt/home
mount -o compress=zstd,noatime,subvol=@nix /dev/mapper/crypted /mnt/nix

echo "[6/6] Provisioning SOPS Age Key & Repository..."
mkdir -p /mnt/home/michael/.config/sops/age
cat <<'EOF_AGE' > /mnt/home/michael/.config/sops/age/keys.txt
# created: 2026-05-09T09:27:08-04:00
# public key: ***REDACTED***
***REDACTED***
EOF_AGE
chmod 600 /mnt/home/michael/.config/sops/age/keys.txt

mkdir -p /mnt/home/michael/git
if [ ! -d /mnt/home/michael/git/sober-nix ]; then
  git clone https://github.com/michaelssingh/sober-nix.git /mnt/home/michael/git/sober-nix
fi

echo "=== Running NixOS System Installation ==="
nixos-install --flake github:michaelssingh/sober-nix#ninox --no-channel-copy

echo "Initial default password for user 'michael' is set to 'nixos'."
read -p "Would you like to set a custom password for user 'michael' now? (y/N) " -n 1 -r < /dev/tty
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
  nixos-enter --root /mnt -- passwd michael
fi

echo "=========================================="
echo "   Ninox Installation Complete!"
echo "   You may now unmount and reboot."
echo "=========================================="
