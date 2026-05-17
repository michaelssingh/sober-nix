{
  pkgs,
  config,
  lib,
  ...
}:
let
  theme = import ../../theme.nix;
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
        size = 11.0;
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
          "${modifier}+z" = "exec foot";
          "${modifier}+d" = "exec fuzzel";
          "${modifier}+b" = "exec firefox";
          "${modifier}+Shift+s" = "exec grimshot copy area";
          "Print" = "exec grimshot copy active";
          "${modifier}+minus" = "scratchpad show";
          "${modifier}+Shift+minus" = "move scratchpad";
          "${modifier}+t" = "floating disable";
        };

      # 3. Simplify Waybar (Reference variables)
      bars = [ { command = "${pkgs.waybar}/bin/waybar"; } ];
      startup = [
        { command = "swaymsg workspace number 1"; }
      ];
    };

    extraConfig = ''
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
