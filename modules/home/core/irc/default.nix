{
  config,
  lib,
  pkgs,
  ...
}:

{
  sops.secrets.soju_password = { };

  xdg.configFile."senpai/senpai.scfg".text = ''
    address irc+insecure://sober-athene.flycast:6697
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
        text 80
    }
    colors {
        prompt "#7aa2f7"
        unread "#7dcfff"
        status disabled 
        nicks base
        nicks self "#bb9af7"
    }
    shortcuts {
        Alt+l set-editor "Lenovo IdeaPad Slim 1-14AST-05"
    }
  '';
}
