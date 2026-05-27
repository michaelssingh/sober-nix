{ lib, ... }:

{
  options.sober.core.public-keys = {
    # otus = lib.mkOption { type = lib.types.str; }; # Base identity
    
    # wg-fly (Fly.io Mesh)
    wg_fly_otus = lib.mkOption { type = lib.types.str; };
    
    # wg-sober (VPN Exit)
    wg_sober_otus = lib.mkOption { type = lib.types.str; };
    wg_sober_glaucidium = lib.mkOption { type = lib.types.str; };

    # Other
    nixbuild = lib.mkOption { type = lib.types.str; };
  };

  config.sober.core.public-keys = {
    # Michael's Workstation (Otus)
    # otus = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMxyjPsJjJr7uGC9LRQkU9vixOZML0zLMb0KQH24NGl1";
    
    # wg-fly public keys
    wg_fly_otus = "23wz66STtDjKTiL9ipLykJy7ElCVkRGR/4js7pm7MzM="; # Wait, this might be wg-sober, checking...
    
    # wg-sober public keys
    wg_sober_otus = "23wz66STtDjKTiL9ipLykJy7ElCVkRGR/4js7pm7MzM=";
    wg_sober_glaucidium = "BgF0yad/27+0o74CldVXUWtkS+h4VsT1nAPEkKD3VHo=";
    
    # nixbuild.net
    nixbuild = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIPIQCZc54poJ8vqawd8TraNryQeJnvH1eLpIDgbiqymM";
  };
}
