{ config, ... }:
let
  colors = config.sober.theme.current.colors;
in
{
  programs.zathura = {
    enable = true;
    options = {
      default-bg = colors.bg;
      default-fg = colors.fg;
      statusbar-bg = colors.bg;
      statusbar-fg = colors.fg;
      inputbar-bg = colors.bg;
      inputbar-fg = colors.fg;
      completion-bg = colors.bg;
      completion-fg = colors.blue;
      completion-highlight-bg = colors.blue;
      completion-highlight-fg = colors.bg;
      highlight-color = colors.yellow;
      highlight-active-color = colors.red;
      notification-bg = colors.bg;
      notification-fg = colors.fg;
    };
  };
}
