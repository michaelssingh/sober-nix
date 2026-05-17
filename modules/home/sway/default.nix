{ config, ... }:
{
  imports = [
    ./components/main.nix
  ];

  # --- Packages needed for Sway ---
  home.packages = with pkgs; [
    swaybg # Wallpaper
    sway-contrib.grimshot # Screenshots
    wl-clipboard # Clipboard
    wf-recorder # Screen recording
    wlr-randr # Monitor settings
    brightnessctl
  ];
}
