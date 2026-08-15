{ config, pkgs, ... }:

let
  animeDir = ../../core/theme/themes/wallpapers/anime;
  wallpapers = [
    "${animeDir}/user_lucy.jpg"
    "${animeDir}/clare.png"
    "${animeDir}/clare_action.png"
    "${animeDir}/akudama_drive.png"
    "${animeDir}/akudama_courier.png"
    "${animeDir}/fmab.png"
    "${animeDir}/fmab_circle.png"
    "${animeDir}/edgerunners_lucy.jpg"
    "${animeDir}/edgerunners_moon.jpg"
  ];
  initialWallpaper = "${animeDir}/user_lucy.jpg";

  rotateScript = pkgs.writeShellScriptBin "rotate-wallpaper" ''
    WALLPAPERS=(
      ${lib.concatStringsSep "\n      " (map (w: "\"${w}\"") wallpapers)}
    )
    STATE_FILE="${config.home.homeDirectory}/.cache/wallpaper_index"
    mkdir -p "${config.home.homeDirectory}/.cache"

    INDEX=0
    if [[ -f "$STATE_FILE" ]]; then
      INDEX=$(cat "$STATE_FILE" 2>/dev/null || echo 0)
    fi

    NEXT_INDEX=$(( (INDEX + 1) % ''${#WALLPAPERS[@]} ))
    echo "$NEXT_INDEX" > "$STATE_FILE"

    NEXT="''${WALLPAPERS[$NEXT_INDEX]}"

    ${pkgs.hyprland}/bin/hyprctl hyprpaper wallpaper "eDP-1,$NEXT" 2>/dev/null || \
    ${pkgs.hyprland}/bin/hyprctl hyprpaper wallpaper ",$NEXT" 2>/dev/null || \
    ${pkgs.systemd}/bin/systemctl --user restart hyprpaper
  '';
  inherit (pkgs) lib;
in
{
  home.packages = [ rotateScript ];

  services.hyprpaper = {
    enable = true;
    settings = {
      ipc = "on";
      splash = false;
      preload = wallpapers;
      wallpaper = [
        ",${initialWallpaper}"
        "eDP-1,${initialWallpaper}"
      ];
    };
  };

  # Systemd User Timer for 5-minute wallpaper rotation
  systemd.user.services.hyprpaper-rotate = {
    Unit = {
      Description = "Rotate Anime Wallpaper Suite (Cyberpunk Edgerunners, Clare, Akudama Drive, FMAB)";
      After = [ "graphical-session.target" ];
    };
    Service = {
      ExecStart = "${rotateScript}/bin/rotate-wallpaper";
      Type = "oneshot";
    };
  };

  systemd.user.timers.hyprpaper-rotate = {
    Unit = {
      Description = "Timer to rotate Anime Wallpapers every 5 minutes";
    };
    Timer = {
      OnBootSec = "10s";
      OnUnitActiveSec = "5m";
      Unit = "hyprpaper-rotate.service";
    };
    Install = {
      WantedBy = [ "timers.target" ];
    };
  };
}
