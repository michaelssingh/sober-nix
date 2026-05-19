{ config, lib, pkgs, ... }:

{
  xdg.configFile."senpai/senpai.scfg".text = ''
    # Senpai configuration for Libera.Chat via Soju Bouncer
    # Managed declaratively by Home Manager
    
    address ircs://libera.chat:6697
    nickname init
    
    # CertFP authentication
    tls-cert %h/senpai/certs/libera.pem
  '';
}
