{ config, pkgs, ... }:

let
  edgerunnersDir = ../../core/theme/themes/wallpapers/edgerunners;
  wallpapers = [
    "${edgerunnersDir}/lucy.jpg"
    "${edgerunnersDir}/oxocarbon.jpg"
    "${edgerunnersDir}/eyes.jpg"
    "${edgerunnersDir}/moon_kiss.jpg"
    "${edgerunnersDir}/edgerunners_orig.png"
  ];
  currentWallpaperSymlink = "${config.home.homeDirectory}/.cache/current_wallpaper";

  rotateScript = pkgs.writeShellScriptBin "rotate-wallpaper" ''
    WALLPAPERS=(
      ${lib.concatStringsSep "\n      " (map (w: "\"${w}\"") wallpapers)}
    )
    CURRENT=$(readlink -f "${currentWallpaperSymlink}" 2>/dev/null || echo "")

    NEXT=""
    for i in "''${!WALLPAPERS[@]}"; do
      if [[ "''${WALLPAPERS[$i]}" == "$CURRENT" ]]; then
        NEXT_INDEX=$(( (i + 1) % ''${#WALLPAPERS[@]} ))
        NEXT="''${WALLPAPERS[$NEXT_INDEX]}"
        break
      fi
    done

    if [[ -z "$NEXT" ]]; then
      NEXT="''${WALLPAPERS[0]}"
    fi

    mkdir -p "${config.home.homeDirectory}/.cache"
    ln -sf "$NEXT" "${currentWallpaperSymlink}"
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
        ",${currentWallpaperSymlink}"
        "eDP-1,${currentWallpaperSymlink}"
      ];
    };
  };

  # Systemd User Timer for 5-minute wallpaper rotation
  systemd.user.services.hyprpaper-rotate = {
    Unit = {
      Description = "Rotate Cyberpunk Edgerunners Wallpaper";
      After = [ "graphical-session.target" ];
    };
    Service = {
      ExecStart = "${rotateScript}/bin/rotate-wallpaper";
      Type = "oneshot";
    };
  };

  systemd.user.timers.hyprpaper-rotate = {
    Unit = {
      Description = "Timer to rotate Cyberpunk Edgerunners Wallpaper every 5 minutes";
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
