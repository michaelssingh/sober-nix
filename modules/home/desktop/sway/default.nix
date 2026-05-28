{ config, pkgs, ... }:
{
  imports = [
    ./components/main.nix
    ./components/terminal.nix
    ./components/launcher.nix
    ./components/mako.nix
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
    sway-audio-idle-inhibit
    libnotify
  ];
}
