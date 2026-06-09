{
  pkgs,
  config,
  ...
}:

{
  config.programs.tmux = {
    enable = true;
    clock24 = true;
    mouse = true;
    prefix = "C-a";
    baseIndex = 1;
    keyMode = "vi";
    terminal = "tmux-256color";
    shell = "${pkgs.fish}/bin/fish";

    extraConfig =
      let
        colors = config.sober.theme.current.colors;
      in
      ''
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

        # --- Official Tokyo Night Tmux Styling (Ref: extras/tmux/) ---
        set -g mode-style "fg=${colors.blue},bg=${colors.bg_highlight}"
        set -g message-style "fg=${colors.blue},bg=${colors.bg_highlight}"
        set -g message-command-style "fg=${colors.blue},bg=${colors.bg_highlight}"

        set -g pane-border-style "fg=${colors.bg_highlight}"
        set -g pane-active-border-style "fg=${colors.blue}"

        set -g status "on"
        set -g status-justify "left"
        set -g status-style "fg=${colors.blue},bg=${colors.bg_dark}"

        set -g status-left-length "100"
        set -g status-right-length "100"

        set -g status-left-style NONE
        set -g status-right-style NONE

        set -g status-left "#[fg=${colors.black},bg=${colors.blue},bold] #S #[fg=${colors.blue},bg=${colors.bg_dark},nobold,nounderscore,noitalics]"

        set -g status-right "#[fg=${colors.bg_dark},bg=${colors.bg_dark},nobold,nounderscore,noitalics]#[fg=${colors.blue},bg=${colors.bg_dark}] #{prefix_highlight} #[fg=${colors.bg_highlight},bg=${colors.bg_dark},nobold,nounderscore,noitalics]#[fg=${colors.blue},bg=${colors.bg_highlight}] %Y-%m-%d  %H:%M #[fg=${colors.blue},bg=${colors.bg_highlight},nobold,nounderscore,noitalics]#[fg=${colors.black},bg=${colors.blue},bold] #h "

        setw -g window-status-activity-style "underscore,fg=${colors.fg_dark},bg=${colors.bg_dark}"
        setw -g window-status-separator ""
        setw -g window-status-style "NONE,fg=${colors.fg_dark},bg=${colors.bg_dark}"
        setw -g window-status-format "#[fg=${colors.bg_dark},bg=${colors.bg_dark},nobold,nounderscore,noitalics]#[default] #I  #W #F #[fg=${colors.bg_dark},bg=${colors.bg_dark},nobold,nounderscore,noitalics]"
        setw -g window-status-current-format "#[fg=${colors.bg_dark},bg=${colors.bg_highlight},nobold,nounderscore,noitalics]#[fg=${colors.blue},bg=${colors.bg_highlight},bold] #I  #W #F #[fg=${colors.bg_highlight},bg=${colors.bg_dark},nobold,nounderscore,noitalics]"

        # --- Keybindings ---
        bind-key C-a send-prefix
        bind -r h select-pane -L
        bind -r j select-pane -D
        bind -r k select-pane -U
        bind -r l select-pane -R
      '';
  };
}
