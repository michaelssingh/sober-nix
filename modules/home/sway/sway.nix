{
  pkgs,
  config,
  lib,
  ...
}:
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
          border = "#7aa2f7";
          background = "#7aa2f7";
          text = "#1a1b26";
          indicator = "#bb9af7";
          childBorder = "#7aa2f7";
        };
        unfocused = {
          border = "#1a1b26";
          background = "#1a1b26";
          text = "#565f89";
          indicator = "#1a1b26";
          childBorder = "#1a1b26";
        };
        urgent = {
          border = "#f7768e";
          background = "#f7768e";
          text = "#1a1b26";
          indicator = "#f7768e";
          childBorder = "#f7768e";
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

      bars = [ { command = "${pkgs.waybar}/bin/waybar"; } ];
      startup = [
        { command = "swaymsg workspace number 1"; }
      ];
    };

    # 3. MOVED IT HERE (Raw Sway Syntax)
    extraConfig = ''
      title_align center

      output * bg ${./bg.jpg} fill
      for_window [app_id="foot"] hints none
      default_border  pixel 1
      default_floating_border pixel 1
      gaps inner 0
      gaps outer 0
      for_window [app_id="waybar"] floating_disable
    '';
  };
  # --- Packages needed for Sway ---
  home.packages = with pkgs; [
    swaybg # Wallpaper
    sway-contrib.grimshot # Screenshots
    wl-clipboard # Clipboard
    wf-recorder # Screen recording
    wlr-randr # Monitor settings
    brightnessctl
  ];

  # --- Notification Center (SwayNC) ---
  services.swaync = {
    enable = false;
    settings = {
      positionX = "right";
      positionY = "top";
      control-center-width = 300;
      fit-to-screen = false;
      layer = "overlay";
      control-center-margin-top = 5;
      control-center-margin-bottom = 5;
      control-center-margin-right = 5;
      control-center-margin-left = 5;
      notification-window-width = 300;
      keyboard-shortcuts = true;
      image-visibility = "when-available";
      transition-time = 200;
      hide-on-clear = false;
      hide-on-action = true;
      script-fail-notify = true;
    };
    # You can add style = '' ... css ... ''; here if you want
  };

  # --- App Launcher (Fuzzel) ---
  programs.fuzzel = {
    enable = true;
    settings = {
      main = {
        font = "FiraCode Nerd Font Mono:size=11";
        prompt = "'❯  '";
        icon-theme = "Papirus-Dark";
        width = 40;
      };
      colors = {
        background = "1a1b26ff";
        text = "c0caf5ff";
        match = "bb9af7ff";
        selection = "7aa2f7ff";
        selection-text = "1a1b26ff";
        border = "7aa2f7ff";
      };
      border = {
        width = 2;
        radius = 5;
      };
    };
  };

  programs.foot = {
    enable = true;
    server.enable = false;
    settings = {
      main = {
        dpi-aware = "yes";
        font = "FiraCode Nerd Font Mono:size=11";
      };
      colors = {
        # alpha = 0.9;
        background = "1a1b26";
        foreground = "c0caf5";
        regular0 = "15161e";
        regular1 = "f7768e";
        regular2 = "9ece6a";
        regular3 = "e0af68";
        regular4 = "7aa2f7";
        regular5 = "bb9af7";
        regular6 = "7dcfff";
        regular7 = "a9b1d6";
      };
    };
  };
  programs.waybar = {
    enable = true;
    settings = [
      {
        layer = "top";
        height = 30;
        position = "top";
        modules-left = [
          "sway/workspaces"
          "sway/mode"
        ];
        modules-center = [ "clock" ];
        modules-right = [ "custom/vpn",
          "cpu"
          "temperature"
          "memory"
          "disk"
          "backlight"
          "pulseaudio"
          "network"
          "battery"
          "tray"
        ];
        disk = {
          interval = 30;
          format = "󰋊 {percentage_used}%";
          path = "/";
        };
        temperature = {
          thermal-zone = 2;
          hwmon-path = "/sys/class/hwmon/hwmon2/temp1_input";
          critical-threshold = 80;
          format-critical = "{icon} {temperatureC}°C";
          format = "{icon} {temperatureC}°C";
          format-icons = [
            ""
            ""
            ""
          ];
        };
        backlight = {
          format = "{icon} {percent}%";
          format-icons = [
            ""
            ""
            ""
            ""
            ""
            ""
            ""
            ""
            ""
          ];
          on-scroll-up = "brightnessctl set 1%+";
          on-scroll-down = "brightnessctl set 1%-";
        };
        pulseaudio = {
          format = "{icon} {volume}%";
          format-muted = "󰝟";
          format-icons = {
            default = [
              "󰕿"
              "󰖀"
              "󰕾"
            ];
          };
          on-click = "pavucontrol";
        };
        network = {
          format-wifi = "  {essid}";
          format-ethernet = "󰈀 {ifname}";
          format-disconnected = "⚠ Disconnected";
          tooltip-format = "{ifname} via {gwaddr} ";
          on-click = "foot -e nmtui";
        };
        cpu = {
          format = "󰍛 {usage}%";
          interval = 10;
        };
        memory = {
          format = "󰘚 {percentage}%";
          interval = 10;
        };
        clock = {
          format = "  {:%H:%M  |  %d %b}";
          tooltip-format = "<big>{:%Y %B}</big>\n<tt><small>{calendar}</small></tt>";
        };
        battery = {
          states = {
            warning = 30;
            critical = 15;
          };
          format = "{icon} {capacity}%";
          format-charging = "󰂄 {capacity}%";
          format-plugged = " {capacity}%";
          format-alt = "{icon} {time}";
          format-icons = [
            "󰁺"
            "󰁻"
            "󰁼"
            "󰁽"
            "󰁾"
            "󰁿"
            "󰂀"
            "󰂁"
            "󰂂"
            "󰁹"
          ];
        };
      }
    ];
    style = ''
      * {
        border: none;
        border-radius: 0;
        font-family: "JetBrainsMono Nerd Font";
        font-size: 13px;
        min-height: 0;
      }
      window#waybar {
        background: rgba(26, 27, 38, 0.95);
        color: #c0caf5;
      }
      #workspaces button {
        padding: 0 5px;
        color: #565f89;
      }
      #workspaces button.focused {
        color: #bb9af7;
      }
      #workspaces button.urgent {
        color: #f7768e;
      }
      #cpu, #temperature, #memory, #disk, #backlight, #pulseaudio, #network, #battery, #tray {
        padding: 0 10px;
        margin: 4px 2px;
        background: #24283b;
        border-radius: 4px;
      }
      #cpu { color: #7aa2f7; }
      #temperature { color: #f7768e; }
      #memory { color: #bb9af7; }
      #disk { color: #9ece6a; }
      #backlight { color: #e0af68; }
      #pulseaudio { color: #7aa2f7; }
      #network { color: #9ece6a; }
      #battery { color: #e0af68; }
      #clock { color: #c0caf5; background: transparent; }
      #battery { color: #e0af68; }
      #battery.charging { color: #9ece6a; }
      #battery.warning:not(.charging) { color: #f7768e; }
    '';
  };
}
