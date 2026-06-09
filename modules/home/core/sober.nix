{
  config,
  lib,
  pkgs,
  ...
}:
let
  t = import ./theme/types.nix { inherit lib; };
  themes = {
    tokyonight-storm = import ./theme/themes/tokyonight-storm.nix;
    tokyonight-night = import ./theme/themes/tokyonight-night.nix;
    tokyonight-day = import ./theme/themes/tokyonight-day.nix;
  };
in
{
  options.sober = {
    isRemote = lib.mkOption {
      type = lib.types.bool;
      default = false;
      description = "Whether the host is a remote server (e.g., bubo).";
    };
    timezone = lib.mkOption {
      type = lib.types.str;
      default = "America/Barbados";
      description = "Default timezone for the system.";
    };
    theme = {
      active = lib.mkOption {
        type = lib.types.enum (builtins.attrNames themes);
        default = if config.sober.isRemote then "tokyonight-night" else "tokyonight-storm";
      };
      current = lib.mkOption {
        type = t.themeType;
        readOnly = true;
        default = themes.${config.sober.theme.active};
      };
    };
  };

  config.services.darkman = {
    enable = !config.sober.isRemote;
    settings = {
      lat = 13.19;
      lng = -59.54;
    };
    darkModeScripts = {
      sway = "${pkgs.sway}/bin/swaymsg reload";
    };
    lightModeScripts = {
      sway = "${pkgs.sway}/bin/swaymsg reload";
    };
  };
}
