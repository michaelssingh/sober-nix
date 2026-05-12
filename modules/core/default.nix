{ pkgs, ... }:

{
  imports = [
    ./networking.nix
    ./time.nix
  ];

  # --- 1. NIX SETTINGS (The Engine) ---
  nixpkgs.config.allowUnfree = true;
  nix.settings = {
    # Enable Flakes (Essential for this setup)
    experimental-features = [
      "nix-command"
      "flakes"
    ];

    # Security: Only allow root and our main user to use remote builders
    trusted-users = [ "root" "michael" ];

    # Optimizations

    substituters = [
      "https://cache.nixos.org?priority=10"
      "https://nix-community.cachix.org"
    ];
    trusted-public-keys = [
      "nix-community.cachix.org-1:mB9FSh9qf2dCimDSUo8Zy7bkq5CX+/rkCWyvRCYg3Fs="
    ];

    # Performance: Use all available power for builds
    max-jobs = "auto";
    cores = 0; # 0 means use all available cores
    auto-optimise-store = true;
  };

  # --- 2. PERFORMANCE & STABILITY ---
  # Use the faster dbus-broker
  services.dbus.implementation = "broker";

  # Distribute interrupts
  services.irqbalance.enable = true;

  # Prevent system freezes when RAM is full (very likely on 4GB)
  services.earlyoom = {
    enable = true;
    freeMemThreshold = 5;
    freeSwapThreshold = 5;
  };

  # Automatic Garbage Collection (Keep disk clean)
  nix.gc = {
    automatic = true;
    dates = "weekly";
    options = "--delete-older-than 7d";
  };

  # --- 3. UNIVERSAL PACKAGES (Root Tools) ---
  # Tools you need available even if you break your user config
  environment.systemPackages = with pkgs; [
    # vim
    git
    curl
    wget
    htop
  ];

  # --- 4. BOOT OPTIMIZATIONS ---
  boot.loader.timeout = 0;
  boot.initrd.systemd.enable = true;
  boot.kernelParams = [
    "quiet"
    "splash"
    "pci=noaer"
    "udev.log_level=3"
    "rd.udev.log_level=3"
    "mitigations=off"
  ];
  systemd.services.NetworkManager-wait-online.enable = false;
}
