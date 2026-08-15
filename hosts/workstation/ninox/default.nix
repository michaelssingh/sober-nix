{
  pkgs,
  user,
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
    ../../../modules/services/wg-sober-oci.nix
  ];

  programs.hyprland = {
    enable = true;
    xwayland.enable = true;
  };
  virtualisation.waydroid.enable = true;
  security.pam.services.hyprlock = { };
  sober.services.greetd.enable = false;

  services.displayManager.gdm.enable = true;
  services.displayManager.defaultSession = "hyprland";

  # --- HOSTNAME & NETWORKING ---
  networking = {
    hostName = "ninox";
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
  services.fprintd.enable = true;

  # --- SWAP & MEMORY TUNING ---
  zramSwap = {
    enable = true;
    algorithm = "zstd";
    memoryPercent = 50;
  };
  boot.kernel.sysctl = {
    "vm.swappiness" = 180;
    "vm.watermark_boost_factor" = 0;
    "vm.watermark_scale_factor" = 125;
    "vm.page-cluster" = 0;
  };

  # --- HARDWARE GRAPHICS & VA-API ---
  hardware.graphics = {
    enable = true;
    enable32Bit = true;
    extraPackages = with pkgs; [
      libva
      libvdpau-va-gl
      libva-vdpau-driver
    ];
  };

  # --- THINKPAD FAN CONTROL & ACPI ---
  boot.kernelModules = [ "thinkpad_acpi" ];
  boot.extraModprobeConfig = ''
    options thinkpad_acpi fan_control=1 experimental=1
  '';

  # --- BOOTLOADER, PLYMOUTH & KERNEL ---
  boot.kernelPackages = pkgs.linuxPackages_latest;
  boot.loader.systemd-boot.enable = true;
  boot.loader.efi.canTouchEfiVariables = true;
  boot.loader.systemd-boot.configurationLimit = 10;

  boot.plymouth = {
    enable = true;
  };
  boot.initrd.verbose = false;
  boot.consoleLogLevel = 0;

  boot.kernelParams = [
    "quiet"
    "splash"
    "loglevel=3"
    "rd.systemd.show_status=false"
    "rd.udev.log_level=3"
    "vt.global_cursor_default=0"
    "amdgpu.dcdebugmask=0x40000"
    "amd_pstate=active"
  ];

  # --- NETWORKING & SECURITY SERVICES ---
  sober.core.networking.mac-rotation.enable = true;
  sober.core.networking.secure-dns.enable = true;
  sober.core.networking.firewall.enable = true;
  sober.services.wg-fly.enable = true;
  sober.services.wg-sober.enable = false;
  sober.services.wg-sober-oci.enable = true;
  sober.services.wg-sober-oci.debugMode = false;
  sober.services.wg-sober-oci.killSwitch = true;

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
    initialPassword = "nixos";
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
  # On Ninox, the system uses LUKS full-disk encryption with Btrfs subvolumes.
  # NixOS runs sops-install-secrets inside the activation script (before systemd
  # has mounted any subvolumes), so /home/michael/.config/sops/age/keys.txt is
  # not yet accessible. We therefore use the host SSH key at /etc/ssh/, which
  # lives on the root subvolume and is available immediately after LUKS unlock.
  #
  # NOTE: If this machine is reinstalled, a new SSH host key will be generated.
  # You must then run:
  #   ssh-to-age -i /etc/ssh/ssh_host_ed25519_key.pub
  # and update the &ninox recipient in .sops.yaml + re-encrypt secrets.yaml.
  sops.defaultSopsFile = ../../../secrets/secrets.yaml;
  sops.age.sshKeyPaths = [ "/etc/ssh/ssh_host_ed25519_key" ];

  nixpkgs.config.allowUnfree = true;
  nixpkgs.config.permittedInsecurePackages = [
    "olm-3.2.16"
  ];
  system.stateVersion = "25.11";

  environment.systemPackages = with pkgs; [
    zoom-us
  ];

  security.sudo.wheelNeedsPassword = false;
}
