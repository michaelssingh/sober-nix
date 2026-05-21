{ config, lib, pkgs, ... }:

{
  xdg.configFile."senpai/senpai.scfg".text = ''
    address irc+insecure://sober-services.internal:6697
    nickname init
    realname michael
    username init
    password "pineapple"

    pane-widths {
        nicknames 10
        channels 0
        members 0
    }
    colors {
        status disabled 
    }
  '';
}
