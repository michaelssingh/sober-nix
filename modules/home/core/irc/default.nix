{ config, lib, pkgs, ... }:

{
  sops.secrets.soju_password = {};

  xdg.configFile."senpai/senpai.scfg".text = ''
    address irc+insecure://sober-services.internal:6697
    nickname init
    realname michael
    username init
    password-cmd "cat ${config.sops.secrets.soju_password.path}"

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
