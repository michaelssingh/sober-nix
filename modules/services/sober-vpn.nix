{ config, lib, pkgs, ... }:

let
  cfg = config.sober.services.sober-vpn;
in
{
  options = {
    sober.services.sober-vpn = {
      enable = lib.mkEnableOption "Sober VPN Client";
    };
  };

  config = lib.mkIf cfg.enable {
    networking.wg-quick.interfaces.wg0 = {
      autostart = true;
      configFile = "${pkgs.writeText "wg0.conf" ''
        [Interface]
        Address = 10.13.13.2/24
        PrivateKey = +K9J24qYg9j5QGraps2tqy0bf5PXT874w9/+G4qr6HM=
        # DNS to use via the VPN
        DNS = 1.1.1.1

        [Peer]
        PublicKey = hTv/+XOD816hekBrLAFwHpwGIpO/QloV3rMOEXKobwc=
        Endpoint = 137.66.4.172:51820
        AllowedIPs = 0.0.0.0/0
        PersistentKeepalive = 25
      ''}";
    };
  };
}
