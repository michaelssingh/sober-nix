{ pkgs, ... }:

{
  # --- 1. Sound (Pipewire) ---
  security.rtkit.enable = true;
  services.pipewire = {
    enable = true;
    alsa.enable = true;
    alsa.support32Bit = true;
    pulse.enable = true;
    jack.enable = true;
  };

  # --- 2. Graphics (Hardware Acceleration) ---
  hardware.graphics = {
    enable = true;
    extraPackages = with pkgs; [
      libvdpau-va-gl
      libva-vdpau-driver
    ];
  };

  # --- 3. Fonts ---
  fonts.packages = with pkgs; [
    nerd-fonts.jetbrains-mono
    nerd-fonts.symbols-only
    noto-fonts-color-emoji
    font-awesome
  ];

  # --- 4. System Services ---
  # XDG Portals (Needed for file dialogs)
  # xdg.portal = {
  #   enable = true;
  #   wlr.enable = true;
  #   extraPortals = [ pkgs.xdg-desktop-portal-gtk ];
  # };
  xdg.portal = {
    enable = true;
    extraPortals = [ pkgs.xdg-desktop-portal-gtk ];

    # ADD THIS BLOCK:
    config = {
      common = {
        default = [ "gtk" ];
      };
    };
  };
  # Keyring (Needed for saving WiFi passwords)
  services.gnome.gnome-keyring.enable = true;
  security.pam.services.login.enableGnomeKeyring = true;
  security.pam.services.swaylock = { };
  services.fwupd.enable = false;
}
