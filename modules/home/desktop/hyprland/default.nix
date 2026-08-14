{
  pkgs,
  ...
}:

{
  imports = [
    ./hyprlock.nix
    ./hypridle.nix
    ./hyprpaper.nix
    ./swaync.nix
    ../waybar.nix
  ];

  home.packages = with pkgs; [
    hyprland
    hyprpaper
    hyprlock
    hypridle
    hyprpicker
    hyprshot
    swaynotificationcenter
    rofi
    grim
    slurp
    wl-clipboard
    brightnessctl
    playerctl
    (pkgs.writeShellScriptBin "launch-workspaces" ''
      CLIENTS=$(${pkgs.hyprland}/bin/hyprctl clients -j 2>/dev/null || echo "[]")

      has_ws() {
          echo "$CLIENTS" | ${pkgs.jq}/bin/jq -e ".[] | select(.workspace.id == $1)" >/dev/null 2>&1
      }

      has_ws_class() {
          echo "$CLIENTS" | ${pkgs.jq}/bin/jq -e ".[] | select(.workspace.id == $1 and (.class | ascii_downcase | contains(\"$2\")))" >/dev/null 2>&1
      }

      if ! has_ws 2; then
          hyprctl dispatch exec [workspace 2] 'ghostty -e attach-tmux comms'
          sleep 0.2
      fi

      if ! has_ws 3; then
          hyprctl dispatch exec [workspace 3] 'qutebrowser'
          sleep 0.2
      fi

      if ! has_ws 9; then
          hyprctl dispatch exec [workspace 9] 'ghostty -e attach-tmux sys'
          sleep 0.2
      fi

      if ! has_ws_class 10 "zathura"; then
          hyprctl dispatch exec [workspace 10] 'zathura "/home/michael/git/books/programming-languages/K&R.epub"'
          sleep 0.2
      fi

      if ! has_ws_class 10 "ghostty"; then
          hyprctl dispatch exec [workspace 10] 'ghostty -e attach-tmux hack'
          sleep 0.2
      fi

      hyprctl dispatch workspace 1
    '')
  ];

  programs.ghostty = {
    enable = true;
    enableFishIntegration = true;
    settings = {
      theme = "TokyoNight Storm";
      font-family = "FiraCode Nerd Font Mono";
      font-size = 12;
      background-opacity = 0.92;
      cursor-style = "block";
      window-padding-x = 8;
      window-padding-y = 8;
      mouse-scroll-multiplier = 0.4;
      window-decoration = false;
    };
  };

  home.pointerCursor = {
    gtk.enable = true;
    x11.enable = true;
    package = pkgs.rose-pine-hyprcursor;
    name = "rose-pine-hyprcursor";
    size = 24;
  };

  programs.rofi = {
    enable = true;
    terminal = "${pkgs.ghostty}/bin/ghostty";
  };

  wayland.windowManager.hyprland = {
    enable = true;
    configType = "hyprlang";
    systemd.enable = true;
    xwayland.enable = true;

    settings = {
      "$mainMod" = "SUPER";

      # Monitor configuration (Auto detection)
      monitor = [
        ",preferred,auto,1"
      ];

      # Environment Variables
      env = [
        "TERMINAL,ghostty"
        "XDG_CURRENT_DESKTOP,Hyprland"
        "XDG_SESSION_TYPE,wayland"
        "XDG_SESSION_DESKTOP,Hyprland"
        "QT_QPA_PLATFORM,wayland"
        "GDK_BACKEND,wayland,x11"
        "CLUTTER_BACKEND,wayland"
        "MOZ_ENABLE_WAYLAND,1"
        "XCURSOR_THEME,rose-pine-hyprcursor"
        "XCURSOR_SIZE,24"
        "HYPRCURSOR_THEME,rose-pine-hyprcursor"
        "HYPRCURSOR_SIZE,24"
      ];

      # Autostart applications & tmux sessions
      exec-once = [
        "hyprctl setcursor rose-pine-hyprcursor 24"
        "systemctl --user import-environment WAYLAND_DISPLAY XDG_CURRENT_DESKTOP"
        "systemctl --user start tmux-autostart"
        "hyprctl dispatch exec [workspace 2] 'ghostty -e attach-tmux comms'"
        "hyprctl dispatch exec [workspace 3] 'qutebrowser'"
        "hyprctl dispatch exec [workspace 9] 'ghostty -e attach-tmux sys'"
        "hyprctl dispatch exec [workspace 10] 'zathura \"/home/michael/git/books/programming-languages/K&R.epub\"'"
        "hyprctl dispatch exec [workspace 10] 'ghostty -e attach-tmux hack'"
        "${pkgs.hyprpaper}/bin/hyprpaper"
        "${pkgs.hypridle}/bin/hypridle"
        "${pkgs.swaynotificationcenter}/bin/swaync"
      ];

      # Input Configuration (Fast keyboard repeat & touchpad matching Sway)
      input = {
        repeat_delay = 250;
        repeat_rate = 45;
        follow_mouse = 0;
        touchpad = {
          natural_scroll = false;
          tap-to-click = true;
          scroll_factor = 0.4;
        };
      };

      # Layout & Aesthetics (Tokyonight Storm Theme)
      general = {
        gaps_in = 4;
        gaps_out = 8;
        border_size = 2;
        "col.active_border" = "rgba(7aa2f7ee) rgba(bb9af7ee) 45deg";
        "col.inactive_border" = "rgba(1a1b26aa)";
        layout = "dwindle";
      };

      decoration = {
        rounding = 8;
        active_opacity = 1.0;
        inactive_opacity = 0.92;
        fullscreen_opacity = 1.0;

        shadow = {
          enabled = true;
          range = 15;
          render_power = 3;
          color = "rgba(1a1b26ee)";
        };

        blur = {
          enabled = true;
          size = 6;
          passes = 2;
          new_optimizations = true;
        };
      };

      animations = {
        enabled = true;
        bezier = [
          "wind, 0.05, 0.9, 0.1, 1.05"
          "winIn, 0.1, 1.1, 0.1, 1.1"
          "winOut, 0.3, -0.3, 0, 1"
          "liner, 1, 1, 1, 1"
        ];
        animation = [
          "windows, 1, 3, wind, slide"
          "windowsIn, 1, 3, winIn, slide"
          "windowsOut, 1, 3, winOut, slide"
          "windowsMove, 1, 1, default"
          "border, 1, 1, liner"
          "fade, 1, 3, default"
          "workspaces, 1, 3, wind, slide"
        ];
      };

      dwindle = {
        preserve_split = true;
      };

      # Keybindings
      bind = [
        # Core Shortcuts
        "$mainMod, RETURN, exec, ghostty"
        "$mainMod, Z, exec, ghostty"
        "$mainMod, B, exec, qutebrowser"
        "$mainMod, Q, killactive,"
        "$mainMod SHIFT, E, exit,"
        "$mainMod, D, exec, rofi -show drun"
        "$mainMod, C, exec, swaync-client -t -sw"
        "$mainMod SHIFT, N, exec, swaync-client -rs"
        "$mainMod, F, togglefloating,"
        "$mainMod, SPACE, fullscreen, 0"
        "$mainMod, Escape, exec, hyprlock"
        "$mainMod, P, pseudo,"

        # Screenshots
        ", Print, exec, hyprshot -m output -o ~/Pictures/Screenshots"
        "$mainMod, Print, exec, hyprshot -m window -o ~/Pictures/Screenshots"
        "$mainMod SHIFT, S, exec, hyprshot -m region -o ~/Pictures/Screenshots"
        "$mainMod SHIFT, Print, exec, hyprshot -m region -o ~/Pictures/Screenshots"

        # Focus Navigation (Vim style + Arrows)
        "$mainMod, h, movefocus, l"
        "$mainMod, l, movefocus, r"
        "$mainMod, k, movefocus, u"
        "$mainMod, j, movefocus, d"
        "$mainMod, left, movefocus, l"
        "$mainMod, right, movefocus, r"
        "$mainMod, up, movefocus, u"
        "$mainMod, down, movefocus, d"

        # Window Movement
        "$mainMod SHIFT, h, movewindow, l"
        "$mainMod SHIFT, l, movewindow, r"
        "$mainMod SHIFT, k, movewindow, u"
        "$mainMod SHIFT, j, movewindow, d"

        # Workspace Switching (1..10)
        "$mainMod, 1, workspace, 1"
        "$mainMod, 2, workspace, 2"
        "$mainMod, 3, workspace, 3"
        "$mainMod, 4, workspace, 4"
        "$mainMod, 5, workspace, 5"
        "$mainMod, 6, workspace, 6"
        "$mainMod, 7, workspace, 7"
        "$mainMod, 8, workspace, 8"
        "$mainMod, 9, workspace, 9"
        "$mainMod, 0, workspace, 10"

        # Move Active Window to Workspace (1..10)
        "$mainMod SHIFT, 1, movetoworkspace, 1"
        "$mainMod SHIFT, 2, movetoworkspace, 2"
        "$mainMod SHIFT, 3, movetoworkspace, 3"
        "$mainMod SHIFT, 4, movetoworkspace, 4"
        "$mainMod SHIFT, 5, movetoworkspace, 5"
        "$mainMod SHIFT, 6, movetoworkspace, 6"
        "$mainMod SHIFT, 7, movetoworkspace, 7"
        "$mainMod SHIFT, 8, movetoworkspace, 8"
        "$mainMod SHIFT, 9, movetoworkspace, 9"
        "$mainMod SHIFT, 0, movetoworkspace, 10"
      ];

      binde = [
        ", XF86AudioRaiseVolume, exec, wpctl set-volume -l 1.5 @DEFAULT_AUDIO_SINK@ 5%+"
        ", XF86AudioLowerVolume, exec, wpctl set-volume @DEFAULT_AUDIO_SINK@ 5%-"
        ", XF86MonBrightnessUp, exec, ${pkgs.brightnessctl}/bin/brightnessctl -d amdgpu_bl1 set +5%"
        ", XF86MonBrightnessDown, exec, ${pkgs.brightnessctl}/bin/brightnessctl -d amdgpu_bl1 set 5%-"
      ];

      bindl = [
        ", XF86AudioMute, exec, wpctl set-mute @DEFAULT_AUDIO_SINK@ toggle"
        ", XF86AudioPlay, exec, ${pkgs.playerctl}/bin/playerctl play-pause"
        ", XF86AudioNext, exec, ${pkgs.playerctl}/bin/playerctl next"
        ", XF86AudioPrev, exec, ${pkgs.playerctl}/bin/playerctl previous"
      ];

      # Mouse Dragging (Hold Super + Left/Right Click)
      bindm = [
        "$mainMod, mouse:272, movewindow"
        "$mainMod, mouse:273, resizewindow"
      ];
    };

    extraConfig = ''
      windowrule = workspace 3, class:org.qutebrowser.qutebrowser
      windowrule = workspace 10, class:org.pwmt.zathura
      windowrule = float, class:mpv
      windowrule = float, class:pavucontrol
      windowrule = float, title:Picture-in-Picture
      windowrule = center, class:mpv
    '';
  };
}
