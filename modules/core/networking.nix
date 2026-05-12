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
      networking.firewall.trustedInterfaces = [
        "tailscale0"
      ];

      services.tailscale.enable = true;

      networking.nftables = {
        enable = true;
        ruleset =
          let
            trustedInterfaces = config.networking.firewall.trustedInterfaces;
            trustedRules = lib.concatMapStringsSep "\n                " (iface: "iifname \"${iface}\" accept") trustedInterfaces;
            trustedOutputRules = lib.concatMapStringsSep "\n                " (iface: "oifname \"${iface}\" accept") trustedInterfaces;
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
                policy accept;

                oifname "lo" accept
                ${trustedOutputRules}
                udp dport 53 accept
                tcp dport 53 accept

                # Allow existing connections
                ct state established,related accept
              }
            }
          '';
      };
    })
  ];
}
