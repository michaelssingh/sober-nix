{ config, lib, pkgs, ... }:

{
  xdg.configFile."senpai/senpai.scfg".text = ''
    # Senpai configuration for Libera.Chat via Soju Bouncer
    # Managed declaratively by Home Manager
    
    # Update these values with your actual Soju bouncer credentials
    address ircs://your-bouncer-address:6697
    nickname your-nick
    password "your-password-or-bouncer-token"
  '';
}
