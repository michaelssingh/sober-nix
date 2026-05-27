{
  config,
  lib,
  pkgs,
  ...
}:

{
  sops.secrets.soju_password = { };

  xdg.configFile."senpai/senpai.scfg".text = ''
    address irc+insecure://sober-athene.internal:6697
    nickname init
    realname michael
    username init
    password pineapple
    spell-check true
    on-highlight-beep true

    pane-widths {
        nicknames 10
        channels 0
        members 0
    }
    colors {
        prompt "#7aa2f7"
        unread "#7dcfff"
        status disabled 
        nicks base
        nicks self "#bb9af7"
    }
    shortcuts {
        Alt+u set-editor "/upload\n"
    }
  '';
}
