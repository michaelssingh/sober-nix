{ pkgs, ... }:

let
  theme = import ../../core/theme.nix;
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
          bar.bg = theme.colors.background;
          indicator.start = theme.colors.blue;
          indicator.stop = theme.colors.green;
          odd.bg = theme.colors.background;
          odd.fg = theme.colors.foreground;
          even.bg = theme.colors.background;
          even.fg = theme.colors.foreground;
          selected.odd.bg = theme.colors.blue;
          selected.odd.fg = theme.colors.background;
          selected.even.bg = theme.colors.blue;
          selected.even.fg = theme.colors.background;
        };
        statusbar = {
          normal.bg = theme.colors.background;
          normal.fg = theme.colors.foreground;
        };
        completion = {
          fg = theme.colors.foreground;
          odd.bg = theme.colors.background;
          even.bg = theme.colors.background;
          item.selected.bg = theme.colors.blue;
          item.selected.fg = theme.colors.background;
          match.fg = theme.colors.red;
        };
        hints = {
          bg = theme.colors.yellow;
          fg = theme.colors.background;
          match.fg = theme.colors.foreground;
        };
        downloads = {
          bar.bg = theme.colors.background;
          start.bg = theme.colors.blue;
          stop.bg = theme.colors.green;
        };
        keyhint = {
          bg = theme.colors.background;
          fg = theme.colors.foreground;
        };
      };
      fonts.default_family = "monospace";
      qt.args = [
        "disable-features=Vp9VideoDecoder,Av1VideoDecoder"
      ];
      url.searchengines = {
        "DEFAULT" = "https://duckduckgo.com/?q={}";
        "g" = "https://google.com/search?q={}";
      };
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
      "text/html" = "qutebrowser.desktop";
      "x-scheme-handler/http" = "qutebrowser.desktop";
      "x-scheme-handler/https" = "qutebrowser.desktop";
      "x-scheme-handler/about" = "qutebrowser.desktop";
      "x-scheme-handler/unknown" = "qutebrowser.desktop";
    };
  };
}
