{
  lib,
  config,
  ...
}:

{
  options = {
    sober.core.networking = {
      mac-rotation.enable = lib.mkEnableOption "MAC Address Rotation/Randomization" // {
        default = true;
      };
      firewall.enable = lib.mkOption {
        type = lib.types.bool;
        default = true;
        description = "Whether to enable the firewall.";
      };
    };
  };

  config = lib.mkMerge [
    (lib.mkIf config.sober.core.networking.mac-rotation.enable {
      networking.networkmanager.settings = {
        "device" = {
          "wifi.scan-rand-mac-address" = "yes";
        };
        "connection" = {
          "wifi.cloned-mac-address" = "random";
          "ethernet.cloned-mac-address" = "random";
        };
      };
    })
    (lib.mkIf config.sober.core.networking.firewall.enable {
      networking.firewall = {
        enable = true;
      };
      # Use nftables for modern filtering
      networking.nftables = {
        enable = true;
        ruleset = ''
          table inet filter {
            chain input {
              type filter hook input priority filter;
              # Stealth mode: drop ICMP echo requests
              ip protocol icmp icmp type echo-request drop
            }
          }
        '';
      };
    })
  ];
}
