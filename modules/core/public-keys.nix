{ lib, ... }:

{
  options.sober.core.public-keys = {
    # SSH Public Keys (Labels match Bitwarden)
    github = lib.mkOption { type = lib.types.str; };
    fly = lib.mkOption { type = lib.types.str; };
    nixbuild = lib.mkOption { type = lib.types.str; };

    # WireGuard Public Keys
    wg_fly_otus = lib.mkOption { type = lib.types.str; };
    wg_sober_otus = lib.mkOption { type = lib.types.str; };
    wg_sober_glaucidium = lib.mkOption { type = lib.types.str; };
  };

  config.sober.core.public-keys = {
    # SSH Keys
    github = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHy4pVdTzFoOPdQkXdwG9cJuSeHCXm3UTaDgRl/UXGWI";
    fly = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAPGyqlLfLc3PTAQ00M2fg4kaEnoOkmMfECNGOQo/2FI";
    nixbuild = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEk7iS+9SpdVSG/RVdVjPP13RDyd/xLBNMvcVbpgJrAX";

    # WireGuard Keys
    wg_fly_otus = "23wz66STtDjKTiL9ipLykJy7ElCVkRGR/4js7pm7MzM=";
    wg_sober_otus = "23wz66STtDjKTiL9ipLykJy7ElCVkRGR/4js7pm7MzM=";
    wg_sober_glaucidium = "BgF0yad/27+0o74CldVXUWtkS+h4VsT1nAPEkKD3VHo=";
  };
}
