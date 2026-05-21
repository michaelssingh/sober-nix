{ pkgs, ... }:

{
  # WireGuard configuration for Sober VPN Server
  programs.wireguard = {
    enable = true;
  };

  # Using systemd-managed network-online and networking for the interface
  # We use systemd.network for a declarative approach
  systemd.network.netdevs."10-wg-sober" = {
    netdevConfig = {
      Name = "wg-sober";
      Kind = "wireguard";
    };
    wireguardConfig = {
      PrivateKey = "0DGVDVBobtwR3yoX290O0naF1Wswmi2pbwJmnSV9L2w=";
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

  # Handle NAT/Masquerading via nftables (modern iptables)
  networking.nftables = {
    enable = true;
    ruleset = ''
      table ip nat {
        chain postrouting {
          type nat hook postrouting priority srcnat;
          policy accept;
          oifname "eth0" masquerade
        }
      }
    '';
  };
}
