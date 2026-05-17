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
        modules-right = [ "group/system" "group/media" "network" "custom/vpn" "battery" "tray" ];

        "group/system" = {
          orientation = "horizontal";
          modules = [ "cpu" "memory" "disk" "temperature" ];
          drawer = { transition-duration = 300; children-class = "not-visible"; };
        };
        "group/media" = {
          orientation = "horizontal";
          modules = [ "pulseaudio" "backlight" ];
          drawer = { transition-duration = 300; };
        };

        "custom/vpn" = {
          format = "󰖂 {}";
          exec = "ip link show wg0 > /dev/null 2>&1 && echo 'ON' || echo 'OFF'";
          interval = 5;
        };
        cpu = { format = "󰍛 {usage}%"; };
        memory = { format = "󰘚 {percentage}%"; };
        disk = { format = "󰋊 {percentage_used}%"; path = "$SOBER_WAYBAR_DISK_PATH"; };
        temperature = { hwmon-path = "$SOBER_WAYBAR_TEMP_PATH"; format = " {temperatureC}°C"; };
        pulseaudio = { format = "{icon} {volume}%"; format-icons = { default = ["󰕿" "󰖀" "󰕾"]; }; };
        backlight = { format = "{icon} {percent}%"; format-icons = ["" "" "" "" "" "" "" "" ""]; };
        network = { format-wifi = " "; format-ethernet = "󰈀"; };
        clock = { format = " {:%H:%M | %d %b}"; };
        battery = { format = "{icon} {capacity}%"; format-icons = ["󰁺" "󰁻" "󰁼" "󰁽" "󰁾" "󰁿" "󰂀" "󰂁" "󰂂" "󰁹"]; };
      }
    ];
    style = ''
      * { border: none; border-radius: 0; font-family: "JetBrainsMono Nerd Font"; font-size: 13px; }
      window#waybar { background: #1a1b26; color: #c0caf5; }
      #workspaces button { color: #565f89; padding: 0 5px; }
      #workspaces button.focused { color: #bb9af7; }
      #cpu, #memory, #disk, #temperature, #pulseaudio, #backlight, #network, #battery, #tray {
        padding: 0 8px; color: #c0caf5;
      }
    '';
  };
}
