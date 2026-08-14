{ config, lib, ... }:
let
  colors = config.sober.theme.current.colors;
  isHyprland = config.wayland.windowManager.hyprland.enable;
  workspaceModule = if isHyprland then "hyprland/workspaces" else "sway/workspaces";
in
{
  programs.waybar = {
    enable = true;
    systemd.enable = true;
    settings = [
      {
        layer = "top";
        height = 22;
        position = "bottom";
        modules-left = [ workspaceModule ] ++ lib.optionals (!isHyprland) [ "sway/mode" ];
        modules-center = [ "clock" ];
        modules-right = [
          "custom/hosts"
          "group/system"
          "group/media"
          "network"
          "battery"
          "tray"
        ];

        "hyprland/workspaces" = {
          disable-scroll = true;
          all-outputs = true;
          format = "{name}";
          sort-by-number = true;
          on-click = "activate";
          persistent-workspaces = {
            "*" = [
              1
              2
              3
              9
              10
            ];
          };
        };

        "sway/workspaces" = {
          disable-scroll = true;
          all-outputs = true;
          format = "{name}";
          sort-by-number = true;
        };

        "custom/hosts" = {
          format = "{}";
          exec = "/etc/profiles/per-user/michael/bin/python3 ${config.home.homeDirectory}/git/sober-nix/bin/waybar-hosts";
          interval = 30;
          return-type = "json";
        };

        "group/system" = {
          orientation = "horizontal";
          modules = [
            "cpu"
            "memory"
            "disk"
            "temperature"
          ];
          drawer = {
            transition-duration = 300;
            children-class = "not-visible";
          };
        };
        "group/media" = {
          orientation = "horizontal";
          modules = [
            "pulseaudio"
            "backlight"
          ];
          drawer = {
            transition-duration = 300;
          };
        };

        cpu = {
          format = "󰍛 {usage}%";
        };
        memory = {
          format = "󰘚 {percentage}%";
        };
        disk = {
          format = "󰋊 {percentage_used}%";
          path = "/";
        };
        temperature = {
          hwmon-path = "/sys/class/hwmon/hwmon0/temp1_input";
          format = " {temperatureC}°C";
        };
        pulseaudio = {
          format = "{icon} {volume}%";
          format-muted = "󰖁 {volume}%";
          format-icons = {
            default = [
              "󰕿"
              "󰖀"
              "󰕾"
            ];
          };
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
        };
        network = {
          format-wifi = " ";
          format-ethernet = "󰈀";
          tooltip-format = "{essid} ({signalStrength}%) | IP: {ipaddr}";
        };
        clock = {
          format = " {:%H:%M | %d %b}";
          tooltip-format = "<big>{:%Y %B}</big>\n<tt><small>{calendar}</small></tt>";
        };
        battery = {
          format = "{icon} {capacity}%";
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
      * { border: none; border-radius: 0; font-family: "JetBrainsMono Nerd Font"; font-size: 13px; }
      window#waybar { background: ${colors.bg_dark}; color: ${colors.fg}; }
      #workspaces button { color: ${colors.comment}; padding: 0 5px; }
      #workspaces button.focused, #workspaces button.active { color: ${colors.magenta}; }
      #custom-hosts, #cpu, #memory, #disk, #temperature, #pulseaudio, #backlight, #network, #battery, #tray {
        padding: 0 8px; color: ${colors.fg};
      }
    '';
  };
}
