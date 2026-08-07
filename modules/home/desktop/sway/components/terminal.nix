{
  config,
  lib,
  ...
}:
let
  colors = config.sober.theme.current.colors;
  # Foot requires hex values WITHOUT the '#' prefix
  strip = color: lib.removePrefix "#" color;
in
{
  programs.foot = {
    enable = true;
    server.enable = false;
    settings = {
      main = {
        dpi-aware = "yes";
        font = "FiraCode Nerd Font Mono:size=10";
        pad = "2x0";
      };
      colors-dark = {
        background = strip colors.bg;
        foreground = strip colors.fg;
        regular0 = strip colors.terminal.black;
        regular1 = strip colors.terminal.red;
        regular2 = strip colors.terminal.green;
        regular3 = strip colors.terminal.yellow;
        regular4 = strip colors.terminal.blue;
        regular5 = strip colors.terminal.magenta;
        regular6 = strip colors.terminal.cyan;
        regular7 = strip colors.terminal.white;
        bright0 = strip colors.terminal.bright_black;
        bright1 = strip colors.terminal.bright_red;
        bright2 = strip colors.terminal.bright_green;
        bright3 = strip colors.terminal.bright_yellow;
        bright4 = strip colors.terminal.bright_blue;
        bright5 = strip colors.terminal.bright_magenta;
        bright6 = strip colors.terminal.bright_cyan;
        bright7 = strip colors.terminal.bright_white;
        "16" = strip colors.orange;
        "17" = strip colors.magenta2;
      };
    };
  };

  # Ghostty Terminal Configuration linked to sober.theme
  programs.ghostty = {
    enable = true;
    settings = {
      font-family = "FiraCode Nerd Font";
      font-size = 10;
      font-feature = [
        "+liga"
        "+calt"
      ];
      background = colors.bg;
      foreground = colors.fg;
      cursor-color = colors.fg;
      palette = [
        "0=${colors.terminal.black}"
        "1=${colors.terminal.red}"
        "2=${colors.terminal.green}"
        "3=${colors.terminal.yellow}"
        "4=${colors.terminal.blue}"
        "5=${colors.terminal.magenta}"
        "6=${colors.terminal.cyan}"
        "7=${colors.terminal.white}"
        "8=${colors.terminal.bright_black}"
        "9=${colors.terminal.bright_red}"
        "10=${colors.terminal.bright_green}"
        "11=${colors.terminal.bright_yellow}"
        "12=${colors.terminal.bright_blue}"
        "13=${colors.terminal.bright_magenta}"
        "14=${colors.terminal.bright_cyan}"
        "15=${colors.terminal.bright_white}"
        "16=${colors.orange}"
        "17=${colors.magenta2}"
      ];
      gtk-single-instance = true;
      window-decoration = false;
    };
  };

  # Alacritty Terminal Configuration linked to sober.theme
  programs.alacritty = {
    enable = true;
    settings = {
      font.normal.family = "JetBrainsMono Nerd Font";
      colors = {
        primary = {
          background = colors.bg;
          foreground = colors.fg;
        };
        normal = {
          black = colors.terminal.black;
          red = colors.terminal.red;
          green = colors.terminal.green;
          yellow = colors.terminal.yellow;
          blue = colors.terminal.blue;
          magenta = colors.terminal.magenta;
          cyan = colors.terminal.cyan;
          white = colors.terminal.white;
        };
        bright = {
          black = colors.terminal.bright_black;
          red = colors.terminal.bright_red;
          green = colors.terminal.bright_green;
          yellow = colors.terminal.bright_yellow;
          blue = colors.terminal.bright_blue;
          magenta = colors.terminal.bright_magenta;
          cyan = colors.terminal.bright_cyan;
          white = colors.terminal.bright_white;
        };
      };
    };
  };
}
