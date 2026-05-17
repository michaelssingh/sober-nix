{ config, pkgs, ... }:
{
  programs.waybar = {
    enable = true;
    settings = [
      {
        layer = "bottom";
        height = 30;
        position = "top";
        modules-left = [ "sway/workspaces" "sway/mode" ];
        modules-center = [ "clock" ];
        modules-right = [
          "temperature" "cpu" "memory" "disk" "backlight" "pulseaudio" "network" "custom/vpn" "battery" "tray"
        ];
        
        "custom/vpn" = {
          format = "󰖂 {}";
          exec = "ip link show wg0 > /dev/null 2>&1 && echo '{\"text\": \"ON\", \"tooltip\": \"VPN: Active - '\"$(curl -s https://ifconfig.me)\"'\"}' || echo '{\"text\": \"OFF\", \"tooltip\": \"VPN: Inactive\"}'";
          interval = 5;
          return-type = "json";
          on-click = "systemctl start wg-quick-wg0";
          on-click-right = "systemctl stop wg-quick-wg0";
        };
        disk = {
          interval = 30;
          format = "󰋊 {percentage_used}%";
          path = "$SOBER_WAYBAR_DISK_PATH";
        };
        temperature = {
          hwmon-path = "$SOBER_WAYBAR_TEMP_PATH";
          critical-threshold = 80;
          format = " {temperatureC}°C";
        };
        backlight = {
          format = "{icon} {percent}%";
          format-icons = [ "" "" "" "" "" "" "" "" "" ];
          on-scroll-up = "brightnessctl set 1%+";
          on-scroll-down = "brightnessctl set 1%-";
        };
        pulseaudio = {
          format = "{icon} {volume}%";
          format-muted = "󰝟";
          format-icons = { default = [ "󰕿" "󰖀" "󰕾" ]; };
          on-click = "pavucontrol";
        };
        network = {
          format-wifi = "  {essid} ({signalStrength}%)";
          format-ethernet = "󰈀 {ifname}";
          format-disconnected = "⚠ Disconnected";
          on-click = "foot -e nmtui";
        };
        cpu = { format = "󰍛 {usage}%"; interval = 10; };
        memory = { format = "󰘚 {percentage}%"; interval = 10; };
        clock = { format = "  {:%H:%M  |  %d %b}"; };
        battery = {
          states = { warning = 30; critical = 15; };
          format = "{icon} {capacity}%";
          format-icons = [ "󰁺" "󰁻" "󰁼" "󰁽" "󰁾" "󰁿" "󰂀" "󰂁" "󰂂" "󰁹" ];
        };
      }
    ];
  };
}
