{
  lib,
  config,
  ...
}:

{
  options = {
    sober.core.networking = {
      mac-rotation.enable = lib.mkEnableOption "MAC Address Rotation/Randomization" // {
        default = true;
      };
      firewall.enable = lib.mkOption {
        type = lib.types.bool;
        default = true;
        description = "Whether to enable the firewall.";
      };
      secure-dns.enable = lib.mkEnableOption "DNS-over-TLS and DNSSEC via systemd-resolved" // {
        default = true;
      };
    };
  };

  config = lib.mkMerge [
    (lib.mkIf config.sober.core.networking.secure-dns.enable {
      services.resolved = {
        enable = true;
        dnssec = "true";
        domains = [ "~." ];
        dnsovertls = "true";
        extraConfig = ''
          DNS=9.9.9.9#dns.quad9.net 149.112.112.112#dns.quad9.net 1.1.1.1#cloudflare-dns.com 1.0.0.1#cloudflare-dns.com
        '';
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
            vpnEnabled = config.sober.services.vpn.enable;
            vpnInterface = config.sober.services.vpn.interface;
            vpnEndpoint = config.sober.services.vpn.endpoint;
          in
          ''
            table inet filter {
              chain input {
                type filter hook input priority filter;
                ${lib.optionalString vpnEnabled "policy drop;"}
                
                iifname "lo" accept
                ${lib.optionalString vpnEnabled "iifname \"${vpnInterface}\" accept"}
                
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
                ${lib.optionalString vpnEnabled "policy drop;"}

                oifname "lo" accept
                ${lib.optionalString vpnEnabled "oifname \"${vpnInterface}\" accept"}

                ${lib.optionalString vpnEnabled ''
                  # Killswitch: Allow traffic to VPN endpoint only
                  ip daddr ${vpnEndpoint} udp dport 51820 accept
                  
                  # Allow DHCP
                  udp dport 67-68 accept
                ''}

                # Allow existing connections
                ct state established,related accept
              }
            }
          '';
      };
    })
  ];
}
