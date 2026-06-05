{
  pkgs,
  lib,
  user,
  inputs,
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
    ../../modules/services/nix-remote-builder.nix
    ../../modules/services/wg-fly.nix
    ../../modules/services/wg-sober.nix
  ];

  home-manager.backupFileExtension = "backup";

  # --- ENABLE FEATURES ---
  # Remote Nix Builders
  sober.services.nix-remote-builder.enable = false;

  # Transmission
  services.transmission.enable = true;
  services.transmission.package = inputs.nixpkgs-pinned.legacyPackages.${pkgs.system}.transmission_4;
  services.transmission.settings.download-dir = "/home/michael/torrents/download";
  services.transmission.settings.watch-dir = "/home/michael/torrents/watch";
  services.transmission.settings.watch-dir-enabled = true;

  # Turn on the keyboard remapper
  sober.services.kanata.enable = true;

  # Low-end hardware optimizations
  sober.core.perf.lowend.enable = true;

  # --- Bootloader ---
  boot.loader.systemd-boot.enable = true;
  boot.loader.efi.canTouchEfiVariables = true;

  # --- Networking ---
  sober.core.networking.mac-rotation.enable = true;
  sober.core.networking.secure-dns.enable = true;
  sober.core.networking.firewall.enable = true;
  sober.services.wg-fly.enable = true;
  sober.services.wg-sober.enable = true;

  programs.nh = {
    enable = true;
    flake = "/home/michael/git/sober-nix";
    clean = {
      enable = true;
      extraArgs = "--keep-since 3d --keep 3";
    };
  };

  networking = {
    hostName = "otus";
    networkmanager = {
      enable = true;
      dns = "systemd-resolved";
    };
  };

  programs.fish.enable = true;

  boot.kernelParams = [
    "amd_iommu=on"
    "ivrs_ioapic[4]=00:14.0"
    "ivrs_ioapic[5]=00:00.2"
  ];

  # --- Waybar Hardware Settings ---
  # Hardware-specific paths for monitoring
  # Change these if the hardware environment changes
  environment.variables = {
    SOBER_WAYBAR_TEMP_PATH = "/sys/class/hwmon/hwmon3/temp1_input";
    SOBER_WAYBAR_DISK_PATH = "/";
  };
  programs.dconf.enable = true;

  programs.ssh.knownHosts = {
    nixbuild = {
      hostNames = [ "eu.nixbuild.net" ];
      publicKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIPIQCZc54poJ8vqawd8TraNryQeJnvH1eLpIDgbiqymM";
    };
  };

  programs.ssh.extraConfig = ''
    Host sober-styx.flycast
      Port 2222
      User root
      IdentityFile /home/michael/.ssh/fly
      StrictHostKeyChecking no
      UserKnownHostsFile /dev/null
  '';

  nix = {
    distributedBuilds = true;
buildMachines = [
    {
      hostName = "eu.nixbuild.net";
      protocol = "ssh-ng";
      system = "x86_64-linux";
      maxJobs = 100;
      speedFactor = 2;
      supportedFeatures = [
        "benchmark"
        "big-parallel"
      ];
    }
  ];
  };

  # Bridge the user's SSH agent to the Nix daemon
  # This allows the daemon to use keys loaded via 'bw-ssh-init'.
  systemd.services.nix-daemon.serviceConfig.Environment = [
    "SSH_AUTH_SOCK=/run/user/1001/ssh-agent"
  ];

  users.users.${user} = {
    isNormalUser = true;
    extraGroups = [
      "networkmanager"
      "wheel"
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

  # --- Sops-Nix Configuration ---
  sops.defaultSopsFile = ../../secrets/secrets.yaml;
  sops.age.keyFile = "/home/michael/.config/sops/age/keys.txt";

  # --- System State ---
  # Force rebuild
  nixpkgs.config.allowUnfree = true;
  system.stateVersion = "25.11";

  # Remember password forever until session ends
  security.sudo.extraConfig = ''
    Defaults timestamp_timeout=-1
  '';
}
