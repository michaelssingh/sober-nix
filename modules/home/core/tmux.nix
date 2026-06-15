{
  pkgs,
  config,
  lib,
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
        set -g status-position ${if config.sober.isRemote then "bottom" else "top"}

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
        set -g status-style "fg=${colors.blue},bg=${colors.bg}"

        set -g status-left-length "100"
        set -g status-right-length "100"

        set -g status-left-style NONE
        set -g status-right-style NONE

        set -g status-left "#[fg=${colors.black},bg=${colors.blue},bold] #S #[fg=${colors.blue},bg=${colors.bg},nobold,nounderscore,noitalics]"

        set -g status-right "#[fg=${colors.bg},bg=${colors.bg},nobold,nounderscore,noitalics]#[fg=${colors.blue},bg=${colors.bg}] #{prefix_highlight} #[fg=${colors.bg_highlight},bg=${colors.bg},nobold,nounderscore,noitalics]#[fg=${colors.blue},bg=${colors.bg_highlight}] %Y-%m-%d  %H:%M #[fg=${colors.blue},bg=${colors.bg_highlight},nobold,nounderscore,noitalics]#[fg=${colors.black},bg=${colors.blue},bold] #h "

        setw -g window-status-activity-style "underscore,fg=${colors.fg_dark},bg=${colors.bg}"
        setw -g window-status-separator ""
        setw -g window-status-style "NONE,fg=${colors.fg_dark},bg=${colors.bg}"
        setw -g window-status-format "#[fg=${colors.bg},bg=${colors.bg},nobold,nounderscore,noitalics]#[default] #I  #W #F #[fg=${colors.bg},bg=${colors.bg},nobold,nounderscore,noitalics]"
        setw -g window-status-current-format "#[fg=${colors.bg},bg=${colors.bg_highlight},nobold,nounderscore,noitalics]#[fg=${colors.blue},bg=${colors.bg_highlight},bold] #I  #W #F #[fg=${colors.bg_highlight},bg=${colors.bg},nobold,nounderscore,noitalics]"

        # --- Keybindings ---
        bind-key C-a send-prefix

        # Pane navigation (repeatable)
        bind -r h select-pane -L
        bind -r j select-pane -D
        bind -r k select-pane -U
        bind -r l select-pane -R

        # Pane resizing (repeatable)
        bind -r H resize-pane -L 5
        bind -r J resize-pane -D 5
        bind -r K resize-pane -U 5
        bind -r L resize-pane -R 5

        # Window navigation (repeatable)
        bind -r n next-window
        bind -r p previous-window

        # Vi-mode copy/paste keybindings
        bind-key -T copy-mode-vi v send-keys -X begin-selection
        bind-key -T copy-mode-vi y send-keys -X copy-selection-and-cancel
        bind-key -T copy-mode-vi C-v send-keys -X rectangle-toggle
        bind-key P paste-buffer

      '';
  };

  # Automatically start tmux sessions on user login
  config.systemd.user.services.tmux-autostart = {
    Unit = {
      Description = "Auto-start default tmux sessions";
      Documentation = "man:tmux(1)";
    };
    Install = {
      WantedBy = [ "default.target" ];
    };
    Service = {
      Type = "oneshot";
      RemainAfterExit = true;
      Environment = [
        "PATH=${
          lib.makeBinPath (
            with pkgs;
            [
              tmux
              senpai
              iamb
              aerc
              neovim
            ]
          )
        }"
        "TMUX_TMPDIR=%t"
      ];
      ExecStart = "${pkgs.bash}/bin/bash -l -c ${
        pkgs.writeShellScript "tmux-autostart-script" ''
          # 1. comms session
          if ! tmux has-session -t comms 2>/dev/null; then
            tmux new-session -d -s comms -n senpai 'irc'
            tmux new-window -t comms:2 -n iamb 'matrix'
            tmux new-window -t comms:3 -n aerc 'email'
          fi

          # 2. sys session
          if ! tmux has-session -t sys 2>/dev/null; then
            tmux new-session -d -s sys -n editor -c /home/michael/git/sober-nix 'nvim .'
          fi

          # 3. hack session
          if ! tmux has-session -t hack 2>/dev/null; then
            tmux new-session -d -s hack -n editor -c /home/michael/git/learn/c 'nvim .'
          fi
        ''
      }";
    };
  };
}
