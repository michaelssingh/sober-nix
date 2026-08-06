{ pkgs, soberLib, ... }:

let
  wgConfig = pkgs.writeText "wg0.conf-partial" ''
    Address = 10.13.13.1/24, fd00::1/64
    ListenPort = 51820
    MTU = 1280

    [Peer]
    # otus
    PublicKey = 23wz66STtDjKTiL9ipLykJy7ElCVkRGR/4js7pm7MzM=
    AllowedIPs = 10.13.13.2/32, fd00::2/128

    [Peer]
    # ninox
    PublicKey = vEetjpZ9DSPgsrvVRFox1uDHD0J2XrdG7NJFv9Rau2Y=
    AllowedIPs = 10.13.13.3/32, fd00::3/128
  '';
in
soberLib.mkContainerImage {
  name = "sober-glaucidium";
  harden = true;
  observability = {
    lokiUrl = "https://logs-prod-042.grafana.net";
    prometheusUrl = "https://prometheus-prod-66-prod-us-east-3.grafana.net/api/prom/push";
    apiKeyFile = "/run/secrets/grafana_api_key";
  };
  packages = [
    pkgs.wireguard-tools
    pkgs.iproute2
    pkgs.iptables-legacy
    pkgs.procps
  ];
  users = {
    sshd = {
      uid = 74;
      gid = 74;
      description = "Privilege-separated SSH";
      home = "/var/empty";
      shell = "/bin/nologin";
    };
  };
  exposedPorts = {
    "51820/udp" = { };
  };
  entrypoint = ''


    # Set up kernel tun interface
    ${pkgs.coreutils}/bin/mkdir -p /var/empty
    ${pkgs.coreutils}/bin/mkdir -p /dev/net
    if [ ! -c /dev/net/tun ]; then
        ${pkgs.coreutils}/bin/mknod /dev/net/tun c 10 200
    fi

    # Write private keys and configurations
    ${pkgs.coreutils}/bin/mkdir -p /etc/wireguard
    echo "[Interface]" > /etc/wireguard/wg0.conf
    echo "PrivateKey = $WG_PRIVATE_KEY" >> /etc/wireguard/wg0.conf
    cat ${wgConfig} >> /etc/wireguard/wg0.conf
    chmod 600 /etc/wireguard/wg0.conf

    # Forwarding rules for traffic routing
    sysctl -w net.ipv4.ip_forward=1
    sysctl -w net.ipv6.conf.all.forwarding=1

    iptables-legacy -A FORWARD -i wg0 -j ACCEPT
    iptables-legacy -A FORWARD -o wg0 -m state --state RELATED,ESTABLISHED -j ACCEPT
    iptables-legacy -t nat -A POSTROUTING -s 10.13.13.0/24 -o eth0 -j MASQUERADE

    ip6tables-legacy -A FORWARD -i wg0 -j ACCEPT
    ip6tables-legacy -A FORWARD -o wg0 -m state --state RELATED,ESTABLISHED -j ACCEPT
    ip6tables-legacy -t nat -A POSTROUTING -s fd00::/64 -o eth0 -j MASQUERADE

    wg-quick up wg0
    echo "✅ Glaucidium VPN is online."
    exec sleep infinity
  '';
}
