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
      };
      colors = {
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
}
