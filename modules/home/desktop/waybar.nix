{ config, ... }:
let
  colors = config.sober.theme.current.colors;
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
        modules-left = [
          "sway/workspaces"
          "sway/mode"
        ];
        modules-center = [ "clock" ];
        modules-right = [
          "group/system"
          "group/media"
          "network"
          "battery"
          "tray"
        ];

        "sway/workspaces" = {
          disable-scroll = true;
          all-outputs = true;
          format = "{name}";
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
      #workspaces button.focused { color: ${colors.magenta}; }
      #cpu, #memory, #disk, #temperature, #pulseaudio, #backlight, #network, #battery, #tray {
        padding: 0 8px; color: ${colors.fg};
      }
    '';
  };
}
