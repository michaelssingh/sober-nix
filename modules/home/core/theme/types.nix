{ lib }:
{
  themeType = lib.types.submodule {
    options = {
      name = lib.mkOption { type = lib.types.str; };
      variant = lib.mkOption { type = lib.types.str; }; # e.g. "storm", "night"
      isDark = lib.mkOption { type = lib.types.bool; default = true; };
      
      # The Semantic Palette
      colors = {
        bg      = lib.mkOption { type = lib.types.str; };
        fg      = lib.mkOption { type = lib.types.str; };
        accent  = lib.mkOption { type = lib.types.str; };
        border  = lib.mkOption { type = lib.types.str; };
        # ANSI Base
        black   = lib.mkOption { type = lib.types.str; };
        red     = lib.mkOption { type = lib.types.str; };
        green   = lib.mkOption { type = lib.types.str; };
        yellow  = lib.mkOption { type = lib.types.str; };
        blue    = lib.mkOption { type = lib.types.str; };
        magenta = lib.mkOption { type = lib.types.str; };
        cyan    = lib.mkOption { type = lib.types.str; };
        white   = lib.mkOption { type = lib.types.str; };
        comment = lib.mkOption { type = lib.types.str; };
      };
    };
  };
}
