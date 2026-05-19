{ config, lib, pkgs, ... }:

{
  xdg.configFile."senpai/senpai.scfg".text = ''
    # Senpai configuration for Libera.Chat via Soju Bouncer
    # Managed declaratively by Home Manager
    
    address irc+insecure://sober-services.internal:6697
    nickname init_
    username init
    password "dT4d8y3Tz*kavNrmue4YzDsX3^VdU%9UA%8U"
  '';
}
