{ config, ... }:

let
  wallpaper = config.sober.theme.current.wallpaper;
in
{
  services.hyprpaper = {
    enable = true;
    settings = {
      ipc = "on";
      splash = false;
      preload = [ "${wallpaper}" ];
      wallpaper = [
        ",${wallpaper}"
        "eDP-1,${wallpaper}"
      ];
    };
  };
}
