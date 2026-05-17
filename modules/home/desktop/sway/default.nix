{ config, pkgs, ... }:
{
  imports = [
    ./components/main.nix
    ../waybar.nix
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
