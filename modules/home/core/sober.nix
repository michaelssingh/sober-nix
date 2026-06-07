{ config, lib, ... }:
let
  t = import ./theme/types.nix { inherit lib; };
  themes = {
    tokyonight-storm = import ./theme/themes/tokyonight-storm.nix;
    tokyonight-night = import ./theme/themes/tokyonight-night.nix;
  };
in
{
  options.sober = {
    isRemote = lib.mkOption {
      type = lib.types.bool;
      default = false;
      description = "Whether the host is a remote server (e.g., hashnix, bubo).";
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
}
