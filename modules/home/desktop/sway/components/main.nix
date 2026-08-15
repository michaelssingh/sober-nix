{
  pkgs,
  config,
  lib,
  osConfig ? { },
  ...
}:
let
  colors = config.sober.theme.current.colors;
  termCmd = if ((osConfig.networking.hostName or "") == "otus") then "foot" else "ghostty";
in
{
  wayland.windowManager.sway = {
    enable = true;
    xwayland = true;

    config = {
      input = {
        "type:keyboard" = {
          repeat_delay = "300";
          repeat_rate = "35";
        };
        "type:touchpad" = {
          tap = "enabled";
        };
      };

      seat.seat0.xcursor_theme = "Simp1e-Tokyo-Night 24";
      output = {
        "*" = {
          bg = "${./../bg.jpg} fill";
        };
      }
      // lib.optionalAttrs ((osConfig.networking.hostName or "") == "ninox") {
        "eDP-1" = {
          scale = "1.25";
        };
      };

      modifier = "Mod4";
      terminal = termCmd;
      focus.followMouse = false;
      window.titlebar = false;

      gaps = {
        inner = 0;
        outer = 0;
        smartGaps = false;
      };

      # 1. Fonts
      fonts = {
        names = [ "JetBrainsMono Nerd Font" ];
        style = "Regular";
        size = 9.0;
      };

      # 2. Dynamic Design System Palette
      colors = {
        focused = {
          border = colors.blue;
          background = colors.blue;
          text = colors.bg;
          indicator = colors.blue;
          childBorder = colors.blue;
        };
        focusedInactive = {
          border = colors.black;
          background = colors.bg;
          text = colors.comment;
          indicator = colors.black;
          childBorder = colors.black;
        };
        unfocused = {
          border = colors.black;
          background = colors.bg;
          text = colors.comment;
          indicator = colors.black;
          childBorder = colors.black;
        };
        urgent = {
          border = colors.red;
          background = colors.red;
          text = colors.fg;
          indicator = colors.red;
          childBorder = colors.red;
        };
      };

      keybindings =
        let
          modifier = config.wayland.windowManager.sway.config.modifier;
        in
        lib.mkOptionDefault {
          # Focus movements
          "${modifier}+h" = "focus left";
          "${modifier}+j" = "focus down";
          "${modifier}+k" = "focus up";
          "${modifier}+l" = "focus right";

          # Move containers
          "${modifier}+Shift+h" = "move left";
          "${modifier}+Shift+j" = "move down";
          "${modifier}+Shift+k" = "move up";
          "${modifier}+Shift+l" = "move right";

          # Layout toggles
          "${modifier}+s" = "layout stacking";
          "${modifier}+w" = "layout tabbed";

          # Media Controls
          "XF86AudioRaiseVolume" = "exec wpctl set-volume @DEFAULT_AUDIO_SINK@ 5%+";
          "XF86AudioLowerVolume" = "exec wpctl set-volume @DEFAULT_AUDIO_SINK@ 5%-";
          "XF86AudioMute" = "exec wpctl set-mute @DEFAULT_AUDIO_SINK@ toggle";
          "XF86AudioMicMute" = "exec wpctl set-mute @DEFAULT_AUDIO_SOURCE@ toggle";
          "XF86MonBrightnessUp" = "exec brightnessctl set +5%";
          "XF86MonBrightnessDown" = "exec brightnessctl set 5%-";

          # Screenshots
          "Print" = "exec grimshot --notify savecopy anything";

          "${modifier}+Return" = null;
          "${modifier}+c" = "exec makoctl dismiss --all";
          "${modifier}+z" = "exec ${termCmd}";
          "${modifier}+d" = "exec fuzzel";
          "${modifier}+Shift+d" = "exec dict-lookup";
          "${modifier}+b" = "exec qutebrowser";
          "${modifier}+Shift+n" = "exec makoctl restore";
          "${modifier}+Shift+e" =
            "exec swaylock -f -i ${./../bg.jpg} --indicator-radius 100 --indicator-thickness 7 --ring-color bb9af7 --key-hl-color 9ece6a";
          "${modifier}+minus" = "scratchpad show";
          "${modifier}+Shift+minus" = "move scratchpad";
          "${modifier}+t" = "floating disable";

          # Named workspaces
          "${modifier}+2" = "workspace number 2: comms";
          "${modifier}+3" = "workspace number 3: www";
          "${modifier}+9" = "workspace number 9: sys";
          "${modifier}+0" = "workspace number 10: hack";
          "${modifier}+Shift+2" = "move container to workspace number 2: comms";
          "${modifier}+Shift+3" = "move container to workspace number 3: www";
          "${modifier}+Shift+9" = "move container to workspace number 9: sys";
          "${modifier}+Shift+0" = "move container to workspace number 10: hack";
        };

      # 3. Simplify Waybar (Reference variables)
      bars = [ ];
      startup = [
        {
          command = "systemctl --user import-environment DISPLAY WAYLAND_DISPLAY SWAYSOCK && systemctl --user start tmux-autostart";
        }
        {
          command = "${pkgs.swayidle}/bin/swayidle -w timeout 300 'swaylock -f -i ${./../bg.jpg} --indicator-radius 100 --indicator-thickness 7 --ring-color bb9af7 --key-hl-color 9ece6a' before-sleep 'swaylock -f -i ${./../bg.jpg} --indicator-radius 100 --indicator-thickness 7 --ring-color bb9af7 --key-hl-color 9ece6a' lock 'swaylock -f -i ${./../bg.jpg} --indicator-radius 100 --indicator-thickness 7 --ring-color bb9af7 --key-hl-color 9ece6a'";
        }
        { command = "qutebrowser"; }
        { command = "swaymsg 'workspace number 2: comms; exec ${termCmd} -e attach-tmux comms'"; }
        { command = "swaymsg 'workspace number 9: sys; exec ${termCmd} -e attach-tmux sys'"; }
        {
          command = "swaymsg 'workspace number 10: hack; exec ${termCmd} -e attach-tmux hack; exec zathura \"/home/michael/git/books/K&R2\"'";
        }
        { command = "swaymsg workspace number 1"; }
      ];
    };

    extraConfig = ''
      title_align center

      output * bg ${./../bg.jpg} fill
      for_window [app_id="ghostty"] hints none
      for_window [app_id="foot"] hints none

      default_border  pixel 1
      default_floating_border pixel 1
      gaps inner 0
      gaps outer 0
      for_window [app_id="waybar"] floating_disable

      # Target window workspaces & idle inhibition
      for_window [class="mpv"] move to workspace number 1
      for_window [app_id="mpv"] move to workspace number 1; idle_inhibit focus
      for_window [app_id="qutebrowser"] move to workspace number 3: www; idle_inhibit focus
      for_window [class="firefox"] move to workspace number 3: www; idle_inhibit focus
      for_window [app_id="firefox"] move to workspace number 3: www; idle_inhibit focus
    '';
  };
}
