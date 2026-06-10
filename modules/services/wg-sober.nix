{
  config,
  lib,
  pkgs,
  ...
}:

let
  cfg = config.sober.services.wg-sober;
in
{
  options = {
    sober.services.wg-sober = {
      enable = lib.mkEnableOption "Sober VPN Client";
      interface = lib.mkOption {
        type = lib.types.str;
        default = "wg-sober";
        description = "Name of the Sober VPN interface.";
      };
      # Rationale: Provides a safe way to limit VPN traffic to internal subnets
      # during troubleshooting, preventing a total network lock-out if
      # the VPN routing is misconfigured or if we need to isolate
      # VPN traffic from general internet traffic.
      debugMode = lib.mkEnableOption "Debug mode (limits allowedIPs to internal subnets)";
    };
  };

  config = lib.mkIf cfg.enable {
    networking.firewall.trustedInterfaces = [ cfg.interface ];
    networking.firewall.checkReversePath = "loose";

    sops.secrets.wg_sober_otus_private = {
      sopsFile = ../../secrets/secrets.yaml;
    };

    networking.wg-quick.interfaces."${cfg.interface}" = {
      autostart = true;
      address = [
        "10.13.13.2/32"
        "fd00::2/128"
      ];
      privateKeyFile = config.sops.secrets.wg_sober_otus_private.path;
      mtu = 1280;

      peers = [
        {
          publicKey = "BgF0yad/27+0o74CldVXUWtkS+h4VsT1nAPEkKD3VHo=";
          # Rationale: If debugMode is enabled, only route internal traffic through
          # the VPN to allow for easier debugging of reachability without
          # impacting general internet connectivity. If disabled, route all traffic
          # through the VPN as intended for normal operation.
          allowedIPs =
            if cfg.debugMode then
              [
                "10.13.13.0/24"
                "fd00::/64"
              ]
            else
              [
                "0.0.0.0/0"
                "::/0"
              ];
          endpoint = "168.220.91.179:51820";
          persistentKeepalive = 25;
        }
      ];
      postUp = ''
        ${pkgs.systemd}/bin/resolvectl dns ${cfg.interface} 1.1.1.1#cloudflare-dns.com 1.0.0.1#cloudflare-dns.com
        ${pkgs.systemd}/bin/resolvectl default-route ${cfg.interface} false
        ${pkgs.systemd}/bin/resolvectl domain ${cfg.interface} "~."
        ${pkgs.systemd}/bin/resolvectl dnssec ${cfg.interface} yes
        ${pkgs.systemd}/bin/resolvectl dnsovertls ${cfg.interface} yes 
      '';
    };

    systemd.services."wg-quick-${cfg.interface}" = {
      after = [
        "wg-quick-wg-fly.service"
        "systemd-resolved.service"
      ];
      wants = [
        "wg-quick-wg-fly.service"
        "systemd-resolved.service"
      ];
    };
  };
}
