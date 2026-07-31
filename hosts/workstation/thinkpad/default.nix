{
  pkgs,
  user,
  inputs,
  ...
}:

{
  imports = [
    ./hardware-configuration.nix
    ./disko.nix
    ../../../modules/core
    ../../../modules/roles/workstation
    ../../../modules/services/kanata.nix
    ../../../modules/services/greetd.nix
    ../../../modules/services/wg-fly.nix
    ../../../modules/services/wg-sober.nix
  ];

  home-manager.backupFileExtension = "backup";

  # --- HOSTNAME & NETWORKING ---
  networking = {
    hostName = "thinkpad";
    networkmanager = {
      enable = true;
      dns = "systemd-resolved";
    };
  };

  # --- POWER MANAGEMENT & THINKPAD TUNING ---
  # TLP power management for ThinkPad battery longevity
  services.tlp = {
    enable = true;
    settings = {
      CPU_SCALING_GOVERNOR_ON_AC = "performance";
      CPU_SCALING_GOVERNOR_ON_BAT = "powersave";
      CPU_ENERGY_PERF_POLICY_ON_AC = "performance";
      CPU_ENERGY_PERF_POLICY_ON_BAT = "power";
      START_CHARGE_THRESH_BAT0 = 75;
      STOP_CHARGE_THRESH_BAT0 = 80;
    };
  };

  # --- KEYBOARD & HARDWARE FEATURES ---
  sober.services.kanata.enable = true;

  # --- BOOTLOADER & KERNEL ---
  boot.loader.systemd-boot.enable = true;
  boot.loader.efi.canTouchEfiVariables = true;
  boot.loader.systemd-boot.configurationLimit = 10;

  boot.kernelParams = [
    "amdgpu.dcdebugmask=0x40000"
  ];

  # Enable LUKS prompt in initrd
  boot.initrd.luks.devices."crypted" = {
    device = "/dev/nvme0n1p2";
    preLVM = true;
    allowDiscards = true;
  };

  # --- NETWORKING & SECURITY SERVICES ---
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

  programs.fish.enable = true;

  # Waybar Environment Paths
  environment.variables = {
    SOBER_WAYBAR_TEMP_PATH = "/sys/class/hwmon/hwmon0/temp1_input";
    SOBER_WAYBAR_DISK_PATH = "/";
  };
  programs.dconf.enable = true;

  services.openssh = {
    enable = true;
    settings = {
      PasswordAuthentication = false;
      PermitRootLogin = "no";
    };
  };

  users.users.${user} = {
    isNormalUser = true;
    extraGroups = [
      "networkmanager"
      "wheel"
      "video"
      "input"
    ];
    shell = pkgs.fish;
    openssh.authorizedKeys.keys =
      let
        keys = import ../../../lib/public-keys.nix;
      in
      [
        keys.fly
        keys.nixbuild
        keys.forge
        keys.agy
      ];
  };

  fonts = {
    packages = with pkgs; [
      nerd-fonts.fira-code
      inter
      nerd-fonts.symbols-only
      noto-fonts-color-emoji
      font-awesome
      nerd-fonts.jetbrains-mono
    ];

    fontconfig = {
      enable = true;
      defaultFonts = {
        monospace = [
          "FiraCode Nerd Font Mono"
          "Symbols Nerd Font"
          "Noto Color Emoji"
        ];
        sansSerif = [
          "Inter"
          "Symbols Nerd Font"
          "Noto Color Emoji"
        ];
        serif = [
          "Noto Serif"
          "Symbols Nerd Font"
          "Noto Color Emoji"
        ];
        emoji = [ "Noto Color Emoji" ];
      };
    };
  };

  # --- SOPS-NIX SECRETS ---
  sops.defaultSopsFile = ../../secrets/secrets.yaml;
  sops.age.keyFile = "/home/michael/.config/sops/age/keys.txt";

  nixpkgs.config.allowUnfree = true;
  nixpkgs.config.permittedInsecurePackages = [
    "olm-3.2.16"
  ];
  system.stateVersion = "25.11";

  security.sudo.wheelNeedsPassword = false;
}
