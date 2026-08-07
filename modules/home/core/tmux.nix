{
  pkgs,
  config,
  lib,
  ...
}:

{
  programs.tmux = {
    enable = true;
    clock24 = true;
    mouse = true;
    prefix = "C-a";
    baseIndex = 1;
    keyMode = "vi";
    terminal = "tmux-256color";
    shell = "${pkgs.fish}/bin/fish";
    plugins = with pkgs.tmuxPlugins; [
      fingers
      extrakto
    ];

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
        set -as terminal-features ",ghostty:hyperlinks,foot:hyperlinks,xterm-256color:hyperlinks"
        set -as terminal-overrides ",ghostty:HLS=\E]8;%p1%s;%p2%s\E\\,foot:HLS=\E]8;%p1%s;%p2%s\E\\"
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

        # URL Picker overlay (Prefix + u)
        bind-key u capture-pane -J -S - -E - \; save-buffer /tmp/tmux-url-buffer \; run-shell "${pkgs.bash}/bin/bash -c 'grep -o -E \"(https?|ftp|file)://[-A-Za-z0-9+&@#/%?=~_|!:,.;]*[-A-Za-z0-9+&@#/%=~_|]\" /tmp/tmux-url-buffer | sort -u | ${pkgs.fzf}/bin/fzf --tmux --multi | xargs -r xdg-open'"

      '';
  };

  # Automatically start tmux sessions on user login
  home.packages = [
    (pkgs.writeShellScriptBin "attach-tmux" ''
      if [ -z "''${TMUX_TMPDIR:-}" ] && [ -n "''${XDG_RUNTIME_DIR:-}" ]; then
          export TMUX_TMPDIR="$XDG_RUNTIME_DIR"
      fi

      SESSION_NAME="$1"
      MAX_RETRIES=30
      RETRY_INTERVAL=0.5

      for ((i=1; i<=MAX_RETRIES; i++)); do
          if ${pkgs.tmux}/bin/tmux has-session -t "$SESSION_NAME" 2>/dev/null; then
              exec ${pkgs.tmux}/bin/tmux attach -t "$SESSION_NAME"
          fi
          sleep "$RETRY_INTERVAL"
      done

      echo "Error: Tmux session '$SESSION_NAME' did not appear after 15 seconds."
      read -p "Press enter to exit..."
      exit 1
    '')
  ];

  systemd.user.services.tmux-autostart = lib.mkIf (!config.sober.isRemote) {
    Unit = {
      Description = "Auto-start default tmux sessions";
      Documentation = "man:tmux(1)";
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
      ExecStart = "${pkgs.bash}/bin/bash -l -c ${pkgs.writeShellScript "tmux-autostart-script" ''
        # 1. comms session
        if ! tmux has-session -t comms 2>/dev/null; then
          tmux new-session -d -s comms -n irc
          tmux send-keys -t comms:1 'senpai' C-m
          tmux new-window -t comms:2 -n matrix
          tmux send-keys -t comms:2 'iamb --profile athene' C-m
          tmux new-window -t comms:3 -n email
          tmux send-keys -t comms:3 'aerc' C-m
        fi

        # 2. sys session
        if ! tmux has-session -t sys 2>/dev/null; then
          tmux new-session -d -s sys -n editor -c /home/michael/git/sober-nix 'nvim .'
        fi

        # 3. hack session
        if ! tmux has-session -t hack 2>/dev/null; then
          tmux new-session -d -s hack -n editor -c /home/michael/git/learn/c 'nvim .'
        fi
      ''}";
    };
  };
}
