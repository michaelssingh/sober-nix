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
    };
  };

  config = lib.mkIf cfg.enable {
    # 1. WireGuard Configuration (Declarative)
    networking.wg-quick.interfaces."${cfg.interface}" = {
      autostart = true;
      address = [ "10.2.0.2/32" "2a07:b944::2:2/128" ];
      dns = [ "10.2.0.1" "2a07:b944::2:1" ];
      privateKey = "***REDACTED***="; # TODO: Encrypt later

      peers = [
        {
          publicKey = "gucaLaM/mgJQbHVvnZNtW+1L4Mi7E2mtTMrhS0K4miU=";
          allowedIPs = [ "0.0.0.0/0" "::/0" ];
          endpoint = "${cfg.endpoint}:51820";
          persistentKeepalive = 25;
        }
      ];
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

    environment.systemPackages = [ pkgs.wireguard-tools ];
  };
}
