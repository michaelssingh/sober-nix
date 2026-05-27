{ pkgs, ... }:
let
  theme = import ../../core/theme.nix;
in
{
  programs.zathura = {
    enable = true;
    options = {
      default-bg = theme.colors.background;
      default-fg = theme.colors.foreground;
      statusbar-bg = theme.colors.background;
      statusbar-fg = theme.colors.foreground;
      inputbar-bg = theme.colors.background;
      inputbar-fg = theme.colors.foreground;
      completion-bg = theme.colors.background;
      completion-fg = theme.colors.blue;
      completion-highlight-bg = theme.colors.blue;
      completion-highlight-fg = theme.colors.background;
      highlight-color = theme.colors.yellow;
      highlight-active-color = theme.colors.red;
      notification-bg = theme.colors.background;
      notification-fg = theme.colors.foreground;
    };
  };
}
