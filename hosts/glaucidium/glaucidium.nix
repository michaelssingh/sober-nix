{ pkgs, ... }:

let
  # The wireguard config
  # Note: private key should be provided via ENV or a file in the volume
  wgConfig = pkgs.writeText "wg0.conf-partial" ''
    Address = 10.13.13.1/24, fd00::1/64
    ListenPort = 51820
    MTU = 1280

    [Peer]
    # Otus Client
    PublicKey = 23wz66STtDjKTiL9ipLykJy7ElCVkRGR/4js7pm7MzM=
    AllowedIPs = 10.13.13.2/32, fd00::2/128
  '';

  entrypoint = pkgs.writeShellScriptBin "entrypoint" ''
    set -e

    # 0. Set Hostname
    echo "glaucidium" > /etc/hostname
    ${pkgs.hostname}/bin/hostname -F /etc/hostname || echo "⚠️ Could not set hostname"

    # 1. Setup Tun device
    mkdir -p /dev/net
    if [ ! -c /dev/net/tun ]; then
        mknod /dev/net/tun c 10 200
    fi

    # 2. Write private key and config
    mkdir -p /etc/wireguard
    echo "[Interface]" > /etc/wireguard/wg0.conf
    echo "PrivateKey = $WG_PRIVATE_KEY" >> /etc/wireguard/wg0.conf
    cat ${wgConfig} >> /etc/wireguard/wg0.conf
    chmod 600 /etc/wireguard/wg0.conf

    # 3. Enable IP Forwarding (matching Fedora)
    echo "🌐 Enabling IP forwarding..."
    ${pkgs.procps}/bin/sysctl -w net.ipv4.ip_forward=1
    ${pkgs.procps}/bin/sysctl -w net.ipv6.conf.all.forwarding=1

    # 4. Setup NAT & Forwarding (Verified Legacy Rules)
    echo "🛡️ Setting up NAT and Forwarding..."
    ${pkgs.iptables-legacy}/bin/iptables-legacy -A FORWARD -i wg0 -j ACCEPT
    ${pkgs.iptables-legacy}/bin/iptables-legacy -A FORWARD -o wg0 -m state --state RELATED,ESTABLISHED -j ACCEPT
    ${pkgs.iptables-legacy}/bin/iptables-legacy -t nat -A POSTROUTING -o eth0 -j MASQUERADE

    # IPv6 rules
    ${pkgs.iptables-legacy}/bin/ip6tables-legacy -A FORWARD -i wg0 -j ACCEPT
    ${pkgs.iptables-legacy}/bin/ip6tables-legacy -A FORWARD -o wg0 -m state --state RELATED,ESTABLISHED -j ACCEPT
    ${pkgs.iptables-legacy}/bin/ip6tables-legacy -t nat -A POSTROUTING -o eth0 -j MASQUERADE

    # 5. Start WireGuard
    echo "⚡ Starting WireGuard..."
    ${pkgs.wireguard-tools}/bin/wg-quick up wg0

    echo "✅ Glaucidium VPN is online."
    # 6. Keep container alive
    exec sleep infinity
  '';
in
pkgs.dockerTools.buildLayeredImage {
  name = "sober-glaucidium";
  tag = "latest";

  contents = [
    pkgs.wireguard-tools
    pkgs.iproute2
    pkgs.iptables-legacy
    pkgs.bash
    pkgs.coreutils
    pkgs.procps
    pkgs.hostname
    entrypoint
  ];

  config = {
    Entrypoint = [ "${entrypoint}/bin/entrypoint" ];
    ExposedPorts = {
      "51820/udp" = { };
    };
    Env = [ "PATH=/bin" ];
  };
}
