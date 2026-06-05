{ lib, ... }:

{
  options.sober.core.public-keys = {
    # SSH Public Keys (Labels match Bitwarden)
    forge = lib.mkOption { type = lib.types.str; };
    fly = lib.mkOption { type = lib.types.str; };
    nixbuild = lib.mkOption { type = lib.types.str; };

    # WireGuard Public Keys
    wg_fly_otus = lib.mkOption { type = lib.types.str; };
    wg_sober_otus = lib.mkOption { type = lib.types.str; };
    wg_sober_glaucidium = lib.mkOption { type = lib.types.str; };
  };

  config.sober.core.public-keys = import ../../lib/public-keys.nix;
}
