{ pkgs, ... }:
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
  # --- Systemd user service for sway-audio-idle-inhibit ---
  systemd.user.services.sway-audio-idle-inhibit = {
    Unit = {
      Description = "Sway Audio Idle Inhibit Daemon";
      Documentation = [ "man:sway-audio-idle-inhibit(1)" ];
      After = [
        "graphical-session.target"
        "pipewire.service"
      ];
      PartOf = [ "graphical-session.target" ];
    };
    Service = {
      ExecStart = "${pkgs.sway-audio-idle-inhibit}/bin/sway-audio-idle-inhibit";
      Restart = "on-failure";
      RestartSec = "5s";
    };
    Install = {
      WantedBy = [ "graphical-session.target" ];
    };
  };
}
