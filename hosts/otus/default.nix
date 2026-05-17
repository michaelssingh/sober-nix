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
    ../../modules/services/nix-remote-builder.nix
    ../../modules/services/fly-wireguard.nix
    ../../modules/services/vpn.nix
  ];

  home-manager.backupFileExtension = "backup";

  # --- ENABLE FEATURES ---
  # Remote Nix Builders
  sober.services.nix-remote-builder.enable = true;
  # Fly.io Private Network
  sober.services.fly-wireguard.enable = true;

  # Turn on the keyboard remapper
  sober.services.kanata.enable = true;
  # Secure networking with a VPN
  sober.services.vpn.enable = true;
  sober.services.vpn.killswitch = false;
  # Low-end hardware optimizations
  sober.core.perf.lowend.enable = true;

  # --- Bootloader ---
  boot.loader.systemd-boot.enable = true;
  boot.loader.efi.canTouchEfiVariables = true;

  # --- Networking ---
  sober.core.networking.mac-rotation.enable = true;
  sober.core.networking.secure-dns.enable = true;
  sober.core.networking.firewall.enable = true;

  programs.nh = {
    enable = true;
    flake = "/home/michael/git/sober-nix";
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

  # System-wide SSH config for the Nix daemon (KISS)
  # This fixes the port mapping and host key issues for the background builder.
  programs.ssh.extraConfig = ''
    Host sober-builder.internal
      Port 2222
      User root
      StrictHostKeyChecking no
      UserKnownHostsFile /dev/null
  '';

  # Bridge the user's SSH agent to the Nix daemon
  # This allows the daemon to use keys loaded via 'bw-ssh-init'.
  systemd.services.nix-daemon.serviceConfig.Environment = [ "SSH_AUTH_SOCK=/run/user/1000/ssh-agent" ];

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
