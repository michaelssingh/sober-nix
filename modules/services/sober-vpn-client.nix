{
  config,
  lib,
  pkgs,
  ...
}:

{
  options = {
    sober.services.sober-vpn-client = {
      enable = lib.mkEnableOption "Sober VPN Client";
    };
  };

  config = lib.mkIf config.sober.services.sober-vpn-client.enable {
    networking.firewall.trustedInterfaces = [ "wg-sober" ];
    networking.firewall.checkReversePath = "loose";

    networking.wg-quick.interfaces.wg-sober = {
      autostart = true;
      configFile = "${pkgs.writeText "wg-sober-client.conf" ''
        [Interface]
        Address = 10.13.13.2/32
        PrivateKey = wKeuhB1Y3grUyAir8aWUp+16VWk1/X4QTPW3s3kFN28=
        DNS = 1.1.1.1

        [Peer]
        PublicKey = hTv/+XOD816hekBrLAFwHpwGIpO/QloV3rMOEXKobwc=
        Endpoint = 137.66.4.172:51820
        AllowedIPs = 0.0.0.0/0
        PersistentKeepalive = 25
      ''}";
      postUp = ''
        ${pkgs.systemd}/bin/resolvectl dns wg-sober 1.1.1.1
        ${pkgs.systemd}/bin/resolvectl domain wg-sober "~."
        ${pkgs.systemd}/bin/resolvectl dnsovertls wg-sober no
      '';
    };
  };
}
