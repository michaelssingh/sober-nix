{
  config,
  lib,
  pkgs,
  ...
}:

let
  cfg = config.sober.services.wg-fly;
in
{
  options = {
    sober.services.wg-fly = {
      enable = lib.mkEnableOption "Fly.io Private WireGuard Network";
      interface = lib.mkOption {
        type = lib.types.str;
        default = "wg-fly";
        description = "Name of the Fly.io WireGuard interface.";
      };
      secretName = lib.mkOption {
        type = lib.types.str;
        default = if config.networking.hostName == "ninox" then "wg_fly_ninox_private" else "wg_fly_otus_private";
        description = "Sops secret name for Fly.io WireGuard private key.";
      };
      address = lib.mkOption {
        type = lib.types.listOf lib.types.str;
        default =
          if config.networking.hostName == "ninox" then
            [ "fdaa:3:7a15:a7b:159d:15b0:9fe2:703/120" ]
          else
            [ "fdaa:3:7a15:a7b:159d:15b0:9fe2:702/120" ];
        description = "Fly.io WireGuard interface IP address.";
      };
    };
  };

  config = lib.mkIf cfg.enable {
    # 0. Trust the Fly interface in the firewall
    networking.firewall.trustedInterfaces = [ cfg.interface ];

    sops.secrets."${cfg.secretName}" = {
      sopsFile = ../../secrets/secrets.yaml;
    };

    # 1. WireGuard Configuration
    networking.wg-quick.interfaces."${cfg.interface}" = {
      autostart = true;
      address = cfg.address;
      dns = [ "fdaa:3:7a15::3" ];
      mtu = 1280;

      privateKeyFile = config.sops.secrets."${cfg.secretName}".path;

      peers = [
        {
          # Fly.io Gateway
          publicKey = "aOtHoNmjTnvF32CvJnzFJbOyb9picxPnXgeS8keR2gQ=";
          allowedIPs = [ "fdaa:3:7a15::/48" ];
          endpoint = "dfw1.gateway.6pn.dev:51820";
          persistentKeepalive = 15;
        }
      ];

      # 2. Internal DNS resolution for .internal names
      # We must disable DNSSEC and DoT for this link because Fly.io's
      # internal DNS server does not support them.
      postUp = ''
        ${pkgs.systemd}/bin/resolvectl dns ${cfg.interface} fdaa:3:7a15::3
        ${pkgs.systemd}/bin/resolvectl domain ${cfg.interface} "~internal" "~flycast"
        ${pkgs.systemd}/bin/resolvectl dnssec ${cfg.interface} no
        ${pkgs.systemd}/bin/resolvectl dnsovertls ${cfg.interface} no
      '';
    };

    systemd.services."wg-quick-${cfg.interface}" = {
      before = [ "wg-quick-wg-sober.service" ];
      after = [
        "systemd-resolved.service"
        "network-online.target"
      ];
      wants = [
        "systemd-resolved.service"
        "network-online.target"
      ];
      serviceConfig = {
        Restart = "on-failure";
        RestartSec = "5s";
      };
    };
  };
}
