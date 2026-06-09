{
  pkgs,
  config,
  lib,
  ...
}:
let
  colors = config.sober.theme.current.colors;
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

      # 2. Dynamic Design System Palette
      colors = {
        focused = {
          border = colors.blue;
          background = colors.blue;
          text = colors.bg;
          indicator = colors.magenta;
          childBorder = colors.blue;
        };
        unfocused = {
          border = colors.bg;
          background = colors.bg;
          text = colors.comment;
          indicator = colors.bg;
          childBorder = colors.bg;
        };
        urgent = {
          border = colors.red;
          background = colors.red;
          text = colors.bg;
          indicator = colors.red;
          childBorder = colors.red;
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
          "XF86AudioRaiseVolume" = "exec wpctl set-volume -l 1.0 @DEFAULT_AUDIO_SINK@ 5%+";
          "XF86AudioLowerVolume" = "exec wpctl set-volume @DEFAULT_AUDIO_SINK@ 5%-";
          "XF86AudioMute" = "exec wpctl set-mute @DEFAULT_AUDIO_SINK@ toggle";
          "XF86AudioMicMute" = "exec wpctl set-mute @DEFAULT_AUDIO_SOURCE@ toggle";

          # IdeaPad System Keys
          "F5" = "exec swaymsg reload";
          "XF86TouchpadToggle" = "input \"type:touchpad\" events toggle";
          "XF86RFKill" = "exec rfkill toggle all";
          "F9" =
            "exec swaylock -f -i ${./../bg.jpg} --indicator-radius 100 --indicator-thickness 7 --ring-color bb9af7 --key-hl-color 9ece6a";
          "XF86Display" = "exec wlr-randr --output eDP-1 --toggle";

          # Screenshots
          "Print" = "exec grimshot --notify savecopy anything";

          "${modifier}+Return" = null;
          "${modifier}+c" = "exec makoctl dismiss --all";
          "${modifier}+z" = "exec foot";
          "${modifier}+d" = "exec fuzzel";
          "${modifier}+Shift+d" = "exec dict-lookup";
          "${modifier}+b" = "exec qutebrowser";
          "${modifier}+Shift+n" = "exec makoctl restore";
          "${modifier}+Shift+e" =
            "exec swaylock -f -i ${./../bg.jpg} --indicator-radius 100 --indicator-thickness 7 --ring-color bb9af7 --key-hl-color 9ece6a";
          "${modifier}+minus" = "scratchpad show";
          "${modifier}+Shift+minus" = "move scratchpad";
          "${modifier}+t" = "floating disable";
        };

      # 3. Simplify Waybar (Reference variables)
      bars = [ ];
      startup = [
        { command = "swaymsg workspace number 1"; }
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

      default_border  pixel 1
      default_floating_border pixel 1
      gaps inner 0
      gaps outer 0
      for_window [app_id="waybar"] floating_disable
    '';
  };
}
