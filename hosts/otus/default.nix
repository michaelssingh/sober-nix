{
  pkgs,
  user,
  ...
}:

{
  imports = [
    ./hardware-configuration.nix
    ../../modules/core
    ../../modules/roles/workstation
    ../../modules/services/kanata.nix
    ../../modules/services/greetd.nix
    ../../modules/core/perf-lowend.nix
    ../../modules/services/vpn.nix
  ];

  home-manager.backupFileExtension = "backup";

  # --- ENABLE FEATURES ---
  # Turn on the keyboard remapper
  sober.services.kanata.enable = true;
  # Secure networking with a VPN
  sober.services.vpn.enable = true;
  # Low-end hardware optimizations
  sober.core.perf.lowend.enable = true;

  # --- Bootloader ---
  boot.loader.systemd-boot.enable = true;
  boot.loader.efi.canTouchEfiVariables = true;

  # --- Networking ---
  sober.core.networking.mac-rotation.enable = true;
  sober.core.networking.secure-dns.enable = true;
  sober.core.networking.firewall.enable = true;

  networking = {
    hostName = "otus";
    networkmanager = {
      enable = true;
      dns = "systemd-resolved";
    };
  };

  boot.kernelParams = [
    "amd_iommu=on"
    "ivrs_ioapic[4]=00:14.0"
    "ivrs_ioapic[5]=00:00.2"
  ];

  # --- User ---
  programs.fish.enable = true;
  programs.dconf.enable = true;

  users.users.${user} = {
    isNormalUser = true;
    extraGroups = [
      "networkmanager"
      "wheel"
      "docker"
    ];
    shell = pkgs.fish;
    # initialPassword = "password";
  };

  fonts = {
    # 1. The Warehouse (Install specific packages)
    packages = with pkgs; [
      nerd-fonts.fira-code
      inter

      nerd-fonts.symbols-only
      noto-fonts-color-emoji
      font-awesome

      nerd-fonts.jetbrains-mono
    ];

    # 2. The Manager (Tell the system what to use when)
    fontconfig = {
      enable = true;

      defaultFonts = {
        # Terminal & Code
        monospace = [
          "FiraCode Nerd Font Mono"
          "Symbols Nerd Font" # Fixed correct family name
          "Noto Color Emoji"
        ];

        # UI & Menus (Firefox, File Manager, etc.)
        sansSerif = [
          "Inter"
          "Symbols Nerd Font"
          "Noto Color Emoji"
        ];

        # Fallback for Serif
        serif = [
          "Noto Serif"
          "Symbols Nerd Font"
          "Noto Color Emoji"
        ];

        # Dedicated Emoji
        emoji = [ "Noto Color Emoji" ];
      };
    };
  };

  # --- System State ---
  system.stateVersion = "25.11";
}
