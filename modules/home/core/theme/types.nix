{ lib }:
{
  themeType = lib.types.submodule {
    options = {
      name = lib.mkOption { type = lib.types.str; };
      variant = lib.mkOption { type = lib.types.str; }; # e.g. "storm", "night", "moon", "day"
      isDark = lib.mkOption {
        type = lib.types.bool;
        default = true;
      };
      wallpaper = lib.mkOption { type = lib.types.path; };

      # The Extended Folke Palette (Reference: extras/lua/)
      colors = {
        bg = lib.mkOption { type = lib.types.str; };
        bg_dark = lib.mkOption { type = lib.types.str; };
        bg_highlight = lib.mkOption { type = lib.types.str; };
        bg_visual = lib.mkOption { type = lib.types.str; };
        bg_search = lib.mkOption { type = lib.types.str; };
        bg_sidebar = lib.mkOption { type = lib.types.str; };
        bg_float = lib.mkOption { type = lib.types.str; };

        fg = lib.mkOption { type = lib.types.str; };
        fg_dark = lib.mkOption { type = lib.types.str; };
        fg_gutter = lib.mkOption { type = lib.types.str; };
        fg_sidebar = lib.mkOption { type = lib.types.str; };

        accent = lib.mkOption { type = lib.types.str; };
        border = lib.mkOption { type = lib.types.str; };
        border_highlight = lib.mkOption { type = lib.types.str; };

        # ANSI / Semantic Accents
        black = lib.mkOption { type = lib.types.str; };
        red = lib.mkOption { type = lib.types.str; };
        green = lib.mkOption { type = lib.types.str; };
        yellow = lib.mkOption { type = lib.types.str; };
        blue = lib.mkOption { type = lib.types.str; };
        magenta = lib.mkOption { type = lib.types.str; };
        cyan = lib.mkOption { type = lib.types.str; };
        white = lib.mkOption { type = lib.types.str; };
        orange = lib.mkOption { type = lib.types.str; };
        pink = lib.mkOption { type = lib.types.str; };
        comment = lib.mkOption { type = lib.types.str; };

        # Specialized logic codes (e.g. blue1, green1)
        blue0 = lib.mkOption { type = lib.types.str; };
        blue1 = lib.mkOption { type = lib.types.str; };
        blue2 = lib.mkOption { type = lib.types.str; };
        blue5 = lib.mkOption { type = lib.types.str; };
        blue6 = lib.mkOption { type = lib.types.str; };
        blue7 = lib.mkOption { type = lib.types.str; };
        magenta2 = lib.mkOption { type = lib.types.str; };
        green1 = lib.mkOption { type = lib.types.str; };
        green2 = lib.mkOption { type = lib.types.str; };
        teal = lib.mkOption { type = lib.types.str; };

        # Terminal specific (matching foot/alacritty extras)
        terminal = {
          black = lib.mkOption { type = lib.types.str; };
          red = lib.mkOption { type = lib.types.str; };
          green = lib.mkOption { type = lib.types.str; };
          yellow = lib.mkOption { type = lib.types.str; };
          blue = lib.mkOption { type = lib.types.str; };
          magenta = lib.mkOption { type = lib.types.str; };
          cyan = lib.mkOption { type = lib.types.str; };
          white = lib.mkOption { type = lib.types.str; };
          bright_black = lib.mkOption { type = lib.types.str; };
          bright_red = lib.mkOption { type = lib.types.str; };
          bright_green = lib.mkOption { type = lib.types.str; };
          bright_yellow = lib.mkOption { type = lib.types.str; };
          bright_blue = lib.mkOption { type = lib.types.str; };
          bright_magenta = lib.mkOption { type = lib.types.str; };
          bright_cyan = lib.mkOption { type = lib.types.str; };
          bright_white = lib.mkOption { type = lib.types.str; };
        };
      };
    };
  };
}
