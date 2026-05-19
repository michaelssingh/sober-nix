{ pkgs, ... }:

{
  programs.tmux = {
    enable = true;
    clock24 = true;
    mouse = true;
    prefix = "C-a";
    baseIndex = 1;
    keyMode = "vi";
    terminal = "tmux-256color";

    extraConfig = ''
      # --- Sane Defaults ---
      set -g escape-time 10
      set -g focus-events on
      set -g renumber-windows on
      set -g set-clipboard on

      # Status bar - simple, clean, Tokyonight inspired
      set -g status-style "bg=#1a1b26,fg=#c0caf5"
      set -g status-left " #S "
      set -g status-right " %H:%M "
      
      set -g window-status-current-style "bg=#7aa2f7,fg=#1a1b26"
      set -g window-status-current-format " #I:#W "
      set -g window-status-format " #I:#W "

      # Pane borders
      set -g pane-border-style "fg=#565f89"
      set -g pane-active-border-style "fg=#7aa2f7"
    '';
  };
}
