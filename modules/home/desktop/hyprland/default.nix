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
    };
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
        "XDG_CURRENT_DESKTOP,Hyprland"
        "XDG_SESSION_TYPE,wayland"
        "XDG_SESSION_DESKTOP,Hyprland"
        "QT_QPA_PLATFORM,wayland"
        "GDK_BACKEND,wayland,x11"
        "CLUTTER_BACKEND,wayland"
        "MOZ_ENABLE_WAYLAND,1"
      ];

      # Autostart applications & tmux sessions
      exec-once = [
        "systemctl --user import-environment WAYLAND_DISPLAY XDG_CURRENT_DESKTOP"
        "systemctl --user start tmux-autostart"
        "hyprctl dispatch exec [workspace name:2: comms] 'ghostty -e attach-tmux comms'"
        "hyprctl dispatch exec [workspace name:3: www] 'qutebrowser'"
        "hyprctl dispatch exec [workspace name:9: sys] 'ghostty -e attach-tmux sys'"
        "hyprctl dispatch exec [workspace name:10: hack] 'ghostty -e attach-tmux hack'"
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
          range = 12;
          render_power = 3;
          color = "rgba(1a1b26ee)";
        };

        blur = {
          enabled = true;
          size = 5;
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

      workspace = [
        "1, default:true"
        "2, name:2: comms"
        "3, name:3: www"
        "9, name:9: sys"
        "10, name:10: hack"
      ];

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
        "$mainMod, 2, workspace, name:2: comms"
        "$mainMod, 3, workspace, name:3: www"
        "$mainMod, 4, workspace, 4"
        "$mainMod, 5, workspace, 5"
        "$mainMod, 6, workspace, 6"
        "$mainMod, 7, workspace, 7"
        "$mainMod, 8, workspace, 8"
        "$mainMod, 9, workspace, name:9: sys"
        "$mainMod, 0, workspace, name:10: hack"

        # Move Active Window to Workspace (1..10)
        "$mainMod SHIFT, 1, movetoworkspace, 1"
        "$mainMod SHIFT, 2, movetoworkspace, name:2: comms"
        "$mainMod SHIFT, 3, movetoworkspace, name:3: www"
        "$mainMod SHIFT, 4, movetoworkspace, 4"
        "$mainMod SHIFT, 5, movetoworkspace, 5"
        "$mainMod SHIFT, 6, movetoworkspace, 6"
        "$mainMod SHIFT, 7, movetoworkspace, 7"
        "$mainMod SHIFT, 8, movetoworkspace, 8"
        "$mainMod SHIFT, 9, movetoworkspace, name:9: sys"
        "$mainMod SHIFT, 0, movetoworkspace, name:10: hack"
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
    };

    extraConfig = ''
      windowrule = match:class ^(mpv)$, float 1
      windowrule = match:class ^(pavucontrol)$, float 1
      windowrule = match:title ^(Picture-in-Picture)$, float 1
      windowrule = match:class ^(mpv)$, center 1
    '';
  };
}
