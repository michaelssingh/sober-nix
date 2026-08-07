{
  lib,
  config,
  ...
}:

{
  options = {
    sober.core.networking = {
      mac-rotation.enable = lib.mkEnableOption "MAC Address Rotation/Randomization" // {
        default = false;
      };
      firewall.enable = lib.mkOption {
        type = lib.types.bool;
        default = true;
        description = "Whether to enable the firewall.";
      };
      secure-dns.enable = lib.mkEnableOption "DNS-over-TLS and DNSSEC via systemd-resolved" // {
        default = false;
      };
    };
  };

  config = lib.mkMerge [
    (lib.mkIf config.sober.core.networking.secure-dns.enable {
      services.resolved = {
        enable = true;
        settings = {
          Resolve = {
            DNSSEC = "true";
            Domains = [ "~." ];
            DNSOverTLS = "true";
            DNS = "9.9.9.9#dns.quad9.net 149.112.112.112#dns.quad9.net 1.1.1.1#cloudflare-dns.com 1.0.0.1#cloudflare-dns.com";
          };
        };
      };
    })
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
      networking.firewall.enable = true;

      networking.nftables = {
        enable = true;
        ruleset =
          let
            trustedInterfaces = config.networking.firewall.trustedInterfaces;
            trustedRules = lib.concatMapStringsSep "\n                " (
              iface: "iifname \"${iface}\" accept"
            ) trustedInterfaces;
            trustedOutputRules = lib.concatMapStringsSep "\n                " (
              iface: "oifname \"${iface}\" accept"
            ) trustedInterfaces;
          in
          ''
            table inet filter {
              chain input {
                type filter hook input priority filter;
                policy accept;
                
                iifname "lo" accept
                ${trustedRules}
                
                # Stealth mode: drop ICMP echo requests
                ip protocol icmp icmp type echo-request drop
                
                # Allow existing connections
                ct state established,related accept
              }

              chain forward {
                type filter hook forward priority filter;
                policy drop;
              }

              chain output {
                type filter hook output priority filter;
                policy ${
                  if config.sober.services.wg-sober.enable && config.sober.services.wg-sober.killSwitch then
                    "drop"
                  else
                    "accept"
                };

                # Allow loopback
                oifname "lo" accept

                # Allow return traffic for established connections
                ct state established,related accept

                # Allow outbound traffic through trusted VPN interfaces (wg-sober, wg-fly)
                ${trustedOutputRules}

                # Allow local network subnets for captive portals and LAN access
                ip daddr { 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16 } accept

                # Allow WireGuard UDP handshake packets to endpoints
                udp dport 51820 accept

                # Allow local network DHCP & DNS requests
                udp dport { 53, 67 } accept
                tcp dport 53 accept
                udp sport 68 accept
              }
            }
          '';
      };
    })
    {
      networking.hosts = {
        "10.13.13.1" = [
          "glaucidium"
          "glaucidium.sober.vpn"
        ];
        "10.13.13.2" = [
          "otus"
          "otus.sober.vpn"
        ];
        "10.13.13.3" = [
          "ninox"
          "ninox.sober.vpn"
        ];
      };
    }
  ];
}
