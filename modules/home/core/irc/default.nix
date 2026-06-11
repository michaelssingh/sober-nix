{
  config,
  pkgs,
  ...
}:
let
  colors = config.sober.theme.current.colors;
in
{
  sops.secrets.soju_password = { };

  home.packages = [
    pkgs.weechat-matrix-rs
    (pkgs.weechat.override {
      configure = { availablePlugins, ... }: {
        plugins = with availablePlugins; [
          python
          perl
          lua
        ];
      };
    })
  ];

  xdg.configFile."weechat/python/autoload/soju.py".source = pkgs.fetchurl {
    url = "https://raw.githubusercontent.com/weechat/scripts/master/python/soju.py";
    sha256 = "0lbsm6qcks7ispi3drp5qcj2ixkzlaq83fq474mxw7cbylxjg9kl";
  };

  xdg.configFile."weechat/python/autoload/read_marker.py".source = pkgs.fetchurl {
    url = "https://raw.githubusercontent.com/weechat/scripts/master/python/read_marker.py";
    sha256 = "0dc0q61m7kb39nj3igy2fml011x1lwv967z298cqa2ky9zqdcz3c";
  };

  xdg.configFile."weechat/python/autoload/colorize_nicks.py".source = pkgs.fetchurl {
    url = "https://raw.githubusercontent.com/weechat/scripts/master/python/colorize_nicks.py";
    sha256 = "1zkv0bgkaxp36q5iqyniilg0d0xlvl6qbfm6nibk4w583lny7jwd";
  };

  xdg.configFile."weechat/perl/autoload/colorize_lines.pl".source = pkgs.fetchurl {
    url = "https://raw.githubusercontent.com/weechat/scripts/master/perl/colorize_lines.pl";
    sha256 = "062fzrfi8r0d2h3nvxmdkjkkf7mnjpwgsywiyyhl562dh18gn3iv";
  };

  xdg.configFile."weechat/plugins/matrix.so".source =
    "${pkgs.weechat-matrix-rs}/lib/weechat/plugins/matrix.so";

  xdg.configFile."weechat/irc.conf".text = ''
    [server_default]
    autojoin = ""
    sasl_mechanism = plain

    [server]
    soju.address = "sober-athene.flycast"
    soju.port = 6697
    soju.username = "init @weechat"
    soju.password = "pineapple"
    soju.sasl_username = "init @weechat"
    soju.sasl_password = "pineapple"
    soju.autoconnect = on
    soju.ssl = off
  '';

  xdg.configFile."weechat/matrix-rust.conf".text = ''
    [server]
    athene.address = "http://sober-athene.flycast:6167"
    athene.username = "init"
    athene.password = "pineapple"
    athene.autoconnect = on
  '';

  xdg.configFile."weechat/weechat.conf".text = ''
    [look]
    color_nicks_number = 10
    item_buffer_filter = on
    smart_filter = on

    [filter]
    irc_smart.enabled = on
    irc_smart.buffer = "*"
    irc_smart.rule = "irc_smart_filter"

    [color]
    status_number = "${colors.yellow}"
    status_name = "${colors.blue}"
    status_data = "${colors.cyan}"
    status_more = "${colors.magenta}"
    status_bg = "${colors.bg_dark}"
    chat_bg = "${colors.bg}"
    chat_fg = "${colors.fg}"
    chat_time = "${colors.comment}"
    chat_delimiters = "${colors.comment}"
    chat_highlight = "${colors.red}"
    chat_highlight_bg = "${colors.bg_highlight}"
    chat_nick_self = "${colors.magenta}"
    chat_nick_other = "${colors.blue}"
    chat_prefix_action = "${colors.orange}"
    chat_prefix_join = "${colors.green1}"
    chat_prefix_quit = "${colors.red}"
    chat_prefix_suffix = "${colors.comment}"
    chat_buffer = "${colors.cyan}"
    chat_channel = "${colors.cyan}"
    chat_nick_colors = "${colors.red},${colors.orange},${colors.yellow},${colors.green1},${colors.cyan},${colors.blue},${colors.magenta},${colors.magenta2},${colors.teal},${colors.pink}"

    [bar]
    status.color_bg = "${colors.bg_dark}"
    status.color_fg = "${colors.fg}"
    title.color_bg = "${colors.bg_dark}"
    title.color_fg = "${colors.fg}"
  '';

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
