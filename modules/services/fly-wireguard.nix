{ config, lib, pkgs, ... }:

let
  cfg = config.sober.services.fly-wireguard;
in
{
  options = {
    sober.services.fly-wireguard = {
      enable = lib.mkEnableOption "Fly.io Private WireGuard Network";
      interface = lib.mkOption {
        type = lib.types.str;
        default = "wg-fly";
        description = "Name of the Fly.io WireGuard interface.";
      };
    };
  };

  config = lib.mkIf cfg.enable {
    # 0. Trust the Fly interface in the firewall
    networking.firewall.trustedInterfaces = [ cfg.interface ];

    # 1. WireGuard Configuration
    networking.wg-quick.interfaces."${cfg.interface}" = {
      autostart = true;
      # Address assigned by Fly.io
      address = [ "fdaa:3:7a15:a7b:8cfe:2709:e7f0:ff02/120" ];
      dns = [ "fdaa:3:7a15::3" ];
      
      # Hardcoded for now per user request
      privateKey = "***REDACTED***=";

      peers = [
        {
          # Fly.io Gateway
          publicKey = "q+cTUCrE9NekeuZEF/gCYxr2wNBjvYgGoqYwV1logEI=";
          allowedIPs = [ "fdaa:3:7a15::/48" ];
          endpoint = "iad2.gateway.6pn.dev:51820";
          persistentKeepalive = 15;
        }
      ];

      # 2. Internal DNS resolution for .internal names
      # We must disable DNSSEC and DoT for this link because Fly.io's 
      # internal DNS server does not support them.
      postUp = ''
        ${pkgs.systemd}/bin/resolvectl dns ${cfg.interface} fdaa:3:7a15::3
        ${pkgs.systemd}/bin/resolvectl domain ${cfg.interface} "~internal"
        ${pkgs.systemd}/bin/resolvectl dnssec ${cfg.interface} no
        ${pkgs.systemd}/bin/resolvectl dnsovertls ${cfg.interface} no
      '';
    };
  };
}
