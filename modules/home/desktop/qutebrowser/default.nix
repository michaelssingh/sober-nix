{ config, ... }:

let
  colors = config.sober.theme.current.colors;
  play-smart = ./play-smart.py;
in
{
  home.file.".local/share/qutebrowser/userscripts/play-smart".source = play-smart;
  home.file.".local/share/qutebrowser/userscripts/play-smart".executable = true;

  programs.qutebrowser = {
    enable = true;
    settings = {
      colors = {
        tabs = {
          bar.bg = colors.bg;
          indicator.start = colors.blue;
          indicator.stop = colors.green;
          odd.bg = colors.bg;
          odd.fg = colors.fg;
          even.bg = colors.bg;
          even.fg = colors.fg;
          selected.odd.bg = colors.blue;
          selected.odd.fg = colors.bg;
          selected.even.bg = colors.blue;
          selected.even.fg = colors.bg;
        };
        statusbar = {
          normal.bg = colors.bg;
          normal.fg = colors.fg;
        };
        completion = {
          inherit (colors) fg;
          odd.bg = colors.bg;
          even.bg = colors.bg;
          item.selected.bg = colors.blue;
          item.selected.fg = colors.bg;
          match.fg = colors.red;
        };
        hints = {
          bg = colors.yellow;
          fg = colors.bg;
          match.fg = colors.fg;
        };
        downloads = {
          bar.bg = colors.bg;
          start.bg = colors.blue;
          stop.bg = colors.green;
        };
        keyhint = {
          inherit (colors) bg;
          inherit (colors) fg;
        };
        webpage = {
          inherit (colors) bg;
          darkmode = {
            enabled = true;
            policy.page = "smart";
            # Note: 'background_color' might not exist in all versions,
            # if this causes an error, remove the following line.
          };
          preferred_color_scheme = "dark";
        };
      };
      fonts.default_family = "FiraCode Nerd Font Mono";
      qt.args = [
        "disable-features=Vp9VideoDecoder,Av1VideoDecoder"
      ];
    };
    searchEngines = {
      "DEFAULT" = "https://duckduckgo.com/?q={}";
      "g" = "https://google.com/search?q={}";
      "ai" = "https://www.google.com/search?q={}&v=ai&udm=50&ntc=1";
    };
    keyBindings = {
      normal = {
        ",v" = "spawn --userscript play-smart";
      };
    };
  };

  xdg.mimeApps = {
    enable = true;
    defaultApplications = {
      "text/html" = "org.qutebrowser.qutebrowser.desktop";
      "x-scheme-handler/http" = "org.qutebrowser.qutebrowser.desktop";
      "x-scheme-handler/https" = "org.qutebrowser.qutebrowser.desktop";
    };
  };
}
