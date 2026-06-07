{ pkgs, config, lib, ... }:

{
  options.sober.isRemote = lib.mkOption {
    type = lib.types.bool;
    default = false;
    description = "Whether the host is a remote server (e.g., hashnix, bubo).";
  };

  config.programs.tmux = {
    enable = true;
    clock24 = true;
    mouse = true;
    prefix = "C-a";
    baseIndex = 1;
    keyMode = "vi";
    terminal = "tmux-256color";
    shell = "${pkgs.fish}/bin/fish";

    extraConfig = ''
      # --- Sane Defaults ---
      set -g escape-time 10
      set -g focus-events on
      set -g renumber-windows on
      set -g set-clipboard on

      # Dynamic status position
      set -g status-position ${if config.sober.isRemote then "top" else "bottom"}

      # Ensure mouse events are passed to applications
      set -g mouse on

      # OSC 8 Passthrough & Hyperlink Support
      set -as terminal-features ",foot:hyperlinks"
      set -as terminal-overrides ",foot:HLS=\E]8;%p1%s;%p2%s\E\\"
      set -g default-terminal "tmux-256color"

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

      # --- Keybindings ---
      # Send prefix to underlying application (e.g., jump to beginning of line)
      bind-key C-a send-prefix

      # Vim-style pane navigation
      bind h select-pane -L
      bind j select-pane -D
      bind k select-pane -U
      bind l select-pane -R

      # Repeatable (-r) pane resizing
      bind -r H resize-pane -L 5
      bind -r J resize-pane -D 5
      bind -r K resize-pane -U 5
      bind -r L resize-pane -R 5

      # Repeatable window navigation
      bind -r n next-window
      bind -r p previous-window

      # Vim-style copy mode
      bind-key -T copy-mode-vi v send-keys -X begin-selection
      bind-key -T copy-mode-vi C-v send-keys -X rectangle-toggle
      bind-key -T copy-mode-vi y send-keys -X copy-selection-and-cancel
    '';
  };
}
