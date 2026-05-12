{ config, lib, pkgs, ... }:

let
  cfg = config.sober.services.vpn;
in
{
  options = {
    sober.services.vpn = {
      enable = lib.mkEnableOption "WireGuard VPN with Killswitch and Stealth features";
      interface = lib.mkOption {
        type = lib.types.str;
        default = "wg0";
        description = "Name of the WireGuard interface.";
      };
      endpoint = lib.mkOption {
        type = lib.types.str;
        default = "146.70.230.146";
        description = "The VPN server IP address for the killswitch whitelist.";
      };
      killswitch = lib.mkEnableOption "Enable VPN killswitch";
    };
  };

  config = lib.mkIf cfg.enable {
    # 0. Trust the VPN interface in the firewall
    networking.firewall.trustedInterfaces = [ cfg.interface ];

    # 1. WireGuard Configuration (Declarative)
    networking.wg-quick.interfaces."${cfg.interface}" = {
      autostart = true;
      configFile = "${pkgs.writeText "wg0.conf" ''
        [Interface]
        Address = 10.2.0.2/32, 2a07:b944::2:2/128
        DNS = 10.2.0.1, 2a07:b944::2:1
        PrivateKey = ***REDACTED***=

        [Peer]
        PublicKey = gucaLaM/mgJQbHVvnZNtW+1L4Mi7E2mtTMrhS0K4miU=
        AllowedIPs = 0.0.0.0/0, ::/0
        Endpoint = ${cfg.endpoint}:51820
        PersistentKeepalive = 25
      ''}";
    };

    # 2. Stealth Measures
    # Disable LLMNR and mDNS to prevent local discovery
    services.resolved = {
      llmnr = "false";
      extraConfig = ''
        MulticastDNS=no
      '';
    };

    # Mask Hostname in DHCP requests to landlord's router
    networking.networkmanager.settings = {
      connection = {
        "ipv4.dhcp-send-hostname" = false;
        "ipv6.dhcp-send-hostname" = false;
      };
    };

    environment.systemPackages = [ 
      pkgs.wireguard-tools
      (pkgs.writeShellScriptBin "vpn-killswitch" ''
        if [ "$1" == "off" ]; then
          echo "Disabling killswitch..."
          sudo nft delete chain inet filter output || echo "Killswitch already disabled."
          # We need to recreate the allow-everything ruleset or just delete the restrictive policy
        elif [ "$1" == "on" ]; then
          echo "Enabling killswitch..."
          # Re-apply the restrictive ruleset via systemctl restart
          sudo systemctl restart nftables
        else
          echo "Usage: vpn-killswitch [on|off]"
        fi
      '')
    ];
  };
}
