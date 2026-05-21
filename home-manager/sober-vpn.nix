{ pkgs, ... }:

{
  # Using systemd-managed network-online and networking for the interface
  # We use systemd.network for a declarative approach
  systemd.network.netdevs."10-wg-sober" = {
    netdevConfig = {
      Name = "wg-sober";
      Kind = "wireguard";
    };
    wireguardConfig = {
      PrivateKey = "***REDACTED***";
      ListenPort = 51820;
    };
    wireguardPeers = [
      {
        PublicKey = "mLpRrytjze69fuDpFkxwmYmB5ZBHyJKizfos9jyKWAM=";
        AllowedIPs = [ "10.13.13.2/32" ];
      }
    ];
  };

  systemd.network.networks."10-wg-sober" = {
    matchConfig.Name = "wg-sober";
    address = [ "10.13.13.1/24" ];
    networkConfig.IPForward = "yes";
  };
}
