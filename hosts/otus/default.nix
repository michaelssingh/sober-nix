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
    ../../modules/services/fly-wireguard.nix
    ../../modules/services/sober-vpn-client.nix
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
  sober.services.fly-wireguard.enable = true;
  sober.services.sober-vpn-client.enable = true;

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

  # nixbuild.net
  programs.ssh.extraConfig = ''
    Host sober-services.internal
      HostName fdaa:3:7a15:a7b:572:11c:754f:2
      Port 2222
      User root
      StrictHostKeyChecking no
      UserKnownHostsFile /dev/null

    Host eu.nixbuild.net
      PubkeyAcceptedKeyTypes ssh-ed25519
      ServerAliveInterval 60
      RequestTTY no
      Compression yes
      ControlMaster auto
      ControlPath /tmp/ssh-%r@%h:%p
      ControlPersist 10m
  '';

  programs.ssh.knownHosts = {
    nixbuild = {
      hostNames = [ "eu.nixbuild.net" ];
      publicKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIPIQCZc54poJ8vqawd8TraNryQeJnvH1eLpIDgbiqymM";
    };
    sober-services = {
      hostNames = [
        "[fdaa:3:7a15:a7b:572:11c:754f:2]:2222"
        "sober-services.internal"
      ];
      publicKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMxyjPsJjJr7uGC9LRQkU9vixOZML0zLMb0KQH24NGl1";
    };
  };

  nix = {
    distributedBuilds = true;
    buildMachines = [
      {
        hostName = "sober-services.internal";
        system = "x86_64-linux";
        maxJobs = 4;
        speedFactor = 1;
        supportedFeatures = [
          "benchmark"
          "big-parallel"
        ];
      }
      {
        hostName = "eu.nixbuild.net";
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
