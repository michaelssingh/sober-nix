{
  pkgs,
  config,
  lib,
  ...
}:
let
  theme = import ../../../core/theme.nix;
in
{
  wayland.windowManager.sway = {
    enable = true;
    xwayland = false;

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
        "eDP-1" = {
          res = "1366x768@60Hz";
        };
      };

      modifier = "Mod4";
      terminal = "foot";
      focus.followMouse = false;
      window.titlebar = false;

      gaps = {
        inner = 0;
        outer = 0;
        smartGaps = false;
      };

      # 1. Fonts
      fonts = {
        names = [ "FiraCode Nerd Font" ];
        style = "Regular";
        size = 9.0;
      };

      # 2. The Tokyo Night Palette
      colors = {
        focused = {
          border = theme.colors.blue;
          background = theme.colors.blue;
          text = theme.colors.background;
          indicator = theme.colors.magenta;
          childBorder = theme.colors.blue;
        };
        unfocused = {
          border = theme.colors.background;
          background = theme.colors.background;
          text = theme.colors.comment;
          indicator = theme.colors.background;
          childBorder = theme.colors.background;
        };
        urgent = {
          border = theme.colors.red;
          background = theme.colors.red;
          text = theme.colors.background;
          indicator = theme.colors.red;
          childBorder = theme.colors.red;
        };
      };

      keybindings =
        let
          modifier = config.wayland.windowManager.sway.config.modifier;
        in
        lib.mkOptionDefault {
          # Brightness (Ideapad Keys)
          "XF86MonBrightnessUp" = "exec brightnessctl set 5%+";
          "XF86MonBrightnessDown" = "exec brightnessctl set 5%-";

          # Volume (Pipewire/Wireplumber)
          "XF86AudioRaiseVolume" = "exec wpctl set-volume @DEFAULT_AUDIO_SINK@ 5%+";
          "XF86AudioLowerVolume" = "exec wpctl set-volume @DEFAULT_AUDIO_SINK@ 5%-";
          "XF86AudioMute" = "exec wpctl set-mute @DEFAULT_AUDIO_SINK@ toggle";

          "${modifier}+Return" = null;
          "${modifier}+c" = "exec makoctl dismiss --all";
          "${modifier}+z" = "exec foot";
          "${modifier}+d" = "exec fuzzel";
          "${modifier}+Shift+d" = "exec dict-lookup";
          "${modifier}+b" = "exec firefox";
          "${modifier}+Shift+p" = "exec grimshot --notify savecopy anything";
          "${modifier}+Shift+n" = "exec makoctl restore";
          "${modifier}+Shift+e" =
            "exec swaylock -f -i ${./../bg.jpg} --indicator-radius 100 --indicator-thickness 7 --ring-color bb9af7 --key-hl-color 9ece6a";
          "${modifier}+minus" = "scratchpad show";
          "${modifier}+Shift+minus" = "move scratchpad";
          "${modifier}+i" = "[app_id=\"senpai\"] scratchpad show";
          "${modifier}+p" = "[app_id=\"castero\"] scratchpad show";
          "${modifier}+m" = "[app_id=\"mpv\"] scratchpad show";
          "${modifier}+t" = "floating disable";
        };

      # 3. Simplify Waybar (Reference variables)
      bars = [ ];
      startup = [
        { command = "swaymsg workspace number 1"; }
        { command = "foot --app-id senpai senpai-dev"; }
        { command = "foot --app-id castero castero"; }
        {
          command = "mpv --idle=yes --force-window=yes --input-ipc-server=/tmp/mpv-socket --really-quiet && live dw > /dev/null ";
        }
        {
          command = "${pkgs.swayidle}/bin/swayidle -w timeout 300 'swaylock -f -i ${./../bg.jpg} --indicator-radius 100 --indicator-thickness 7 --ring-color bb9af7 --key-hl-color 9ece6a' before-sleep 'swaylock -f -i ${./../bg.jpg} --indicator-radius 100 --indicator-thickness 7 --ring-color bb9af7 --key-hl-color 9ece6a' lock 'swaylock -f -i ${./../bg.jpg} --indicator-radius 100 --indicator-thickness 7 --ring-color bb9af7 --key-hl-color 9ece6a'";
        }
      ];
    };

    extraConfig = ''
      exec sway-audio-idle-inhibit
      title_align center

      output * bg ${./../bg.jpg} fill
      for_window [app_id="foot"] hints none
      for_window [app_id="senpai"] {
          floating enable
          move position center
          resize set 960 540
          move scratchpad
      }
      for_window [app_id="castero"] {
          floating enable
          move position center
          resize set 960 540 
          move scratchpad
      }
      for_window [app_id="mpv"] {
          floating enable
          move position center
          resize set 960 540
          move scratchpad
      }
      default_border  pixel 1
      default_floating_border pixel 1
      gaps inner 0
      gaps outer 0
      for_window [app_id="waybar"] floating_disable
    '';
  };
}
