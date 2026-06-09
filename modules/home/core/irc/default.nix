{
  config,
  ...
}:
let
  colors = config.sober.theme.current.colors;
in
{
  sops.secrets.soju_password = { };

  # --- Official Tokyo Night Styleset (Ref: extras/aerc/tokyonight_*.ini) ---
  xdg.configFile."aerc/stylesets/tokyonight.ini".text = ''
    *.default=true
    *.normal=true

    border.fg=${colors.fg_gutter}
    border.bg=${colors.bg}

    title.fg=${colors.black}
    title.bg=${colors.blue}
    title.bold=true

    header.fg=${colors.red}
    header.bold=true

    tab.fg=${colors.fg_gutter}
    tab.bg=${colors.bg_dark}
    tab.selected.fg=${colors.black}
    tab.selected.bg=${colors.blue}

    statusline_default.fg=${colors.fg}
    statusline_default.bg=${colors.bg_dark}
    statusline_error.fg=${colors.red}
    statusline_success.fg=${colors.green1}

    *error.bold=true
    *error.fg=${colors.red}
    *warning.fg=${colors.yellow}
    *success.fg=${colors.green1}

    dirlist_*.bg=${colors.bg}
    dirlist_*.fg=${colors.fg}
    dirlist_*.selected.bg=${colors.bg_visual}
    dirlist_*.selected.fg=${colors.fg}

    msglist_*.bg=${colors.bg}
    msglist_*.fg=${colors.fg}
    msglist_*.selected.bg=${colors.bg_visual}
    msglist_unread.bold=true
    msglist_unread.fg=${colors.blue}
    msglist_marked.fg=${colors.orange}

    [viewer]
    url.underline=true
    url.fg=${colors.fg_dark}
    header.fg=${colors.magenta}
    signature.fg=${colors.magenta}
    diff_add.fg=${colors.green1}
    diff_del.fg=${colors.red}
    quote_1.fg=${colors.yellow}
    quote_2.fg=${colors.green}
    quote_3.fg=${colors.green1}
    quote_4.fg=${colors.blue}
    quote_x.fg=${colors.comment}
  '';

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
        prompt "${colors.blue}"
        unread "${colors.cyan}"
        status disabled 
        nicks base
        nicks self "${colors.magenta}"
    }
    shortcuts {
        Alt+l set-editor "Lenovo IdeaPad Slim 1-14AST-05"
    }
  '';
}
