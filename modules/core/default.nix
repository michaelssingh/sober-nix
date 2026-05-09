{ pkgs, ... }:

{
  imports = [
    ./networking.nix
  ];

  # --- 1. NIX SETTINGS (The Engine) ---
  nixpkgs.config.allowUnfree = true;
  nix.settings = {
    # Enable Flakes (Essential for this setup)
    experimental-features = [
      "nix-command"
      "flakes"
    ];

    # Speed up downloads
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

  # Process prioritization
  services.ananicy = {
    enable = true;
    package = pkgs.ananicy-cpp;
    rulesProvider = pkgs.ananicy-rules-cachyos;
  };

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

  # --- 3. SECURE DNS ---
  services.resolved = {
    enable = true;
    dnssec = "true";
    domains = [ "~." ];
    dnsovertls = "opportunistic";
    extraConfig = ''
      DNS=9.9.9.9#dns.quad9.net 149.112.112.112#dns.quad9.net 1.1.1.1#cloudflare-dns.com 1.0.0.1#cloudflare-dns.com
    '';
  };
}
