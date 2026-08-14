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

      # Autostart applications
      exec-once = [
        "${pkgs.waybar}/bin/waybar"
        "${pkgs.hyprpaper}/bin/hyprpaper"
        "${pkgs.hypridle}/bin/hypridle"
        "${pkgs.swaynotificationcenter}/bin/swaync"
        "systemctl --user import-environment WAYLAND_DISPLAY XDG_CURRENT_DESKTOP"
      ];

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
          "windows, 1, 4, wind, slide"
          "windowsIn, 1, 4, winIn, slide"
          "windowsOut, 1, 4, winOut, slide"
          "windowsMove, 1, 4, wind, slide"
          "border, 1, 1, liner"
          "fade, 1, 4, default"
          "workspaces, 1, 4, wind, slide"
        ];
      };

      dwindle = {
        preserve_split = true;
      };

      # Keybindings
      bind = [
        # Core Shortcuts
        "$mainMod, RETURN, exec, ghostty"
        "$mainMod, Q, killactive,"
        "$mainMod SHIFT, E, exit,"
        "$mainMod, D, exec, rofi -show drun"
        "$mainMod, F, togglefloating,"
        "$mainMod, SPACE, fullscreen, 0"
        "$mainMod, L, exec, hyprlock"
        "$mainMod, P, pseudo,"
        "$mainMod SHIFT, S, exec, hyprshot -m region"

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

      # Window Rules
      windowrule = [
        "float, class:^(mpv)$"
        "float, class:^(pavucontrol)$"
        "float, title:^(Picture-in-Picture)$"
        "center, class:^(mpv)$"
      ];
    };
  };
}
