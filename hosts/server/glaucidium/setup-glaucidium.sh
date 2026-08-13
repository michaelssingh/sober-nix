#!/usr/bin/env bash
# ==============================================================================
# Setup Script for glaucidium (Oracle Cloud Always Free VPN Concentrator)
# Reproducibly provisions WireGuard VPN server and firewall on Ubuntu 24.04
# ==============================================================================
set -euo pipefail

echo "=== [1/4] Installing Required Packages ==="
sudo apt-get update -qq
sudo apt-get install -y wireguard wireguard-tools iptables iptables-persistent dnsmasq

echo "=== [2/4] Configuring WireGuard Server Interface (wg0) & DNSmasq ==="
sudo mkdir -p /etc/wireguard
if [ ! -f /etc/wireguard/private.key ]; then
  (umask 077 && sudo wg genkey | sudo tee /etc/wireguard/private.key > /dev/null)
fi
IFACE=$(ip route show default | awk '{print $5}')
PRIVKEY=$(sudo cat /etc/wireguard/private.key)

sudo bash -c "cat << EOF > /etc/wireguard/wg0.conf
[Interface]
Address = 10.13.13.1/24, fd00::1/64
ListenPort = 51820
PrivateKey = \${PRIVKEY}
PostUp = sysctl -w net.ipv4.ip_forward=1; iptables -I FORWARD 1 -i wg0 -j ACCEPT; iptables -I FORWARD 2 -o wg0 -m state --state RELATED,ESTABLISHED -j ACCEPT; iptables -t nat -A POSTROUTING -o \${IFACE} -j MASQUERADE
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT; iptables -D FORWARD -o wg0 -m state --state RELATED,ESTABLISHED -j ACCEPT; iptables -t nat -D POSTROUTING -o \${IFACE} -j MASQUERADE

# otus
[Peer]
PublicKey = 23wz66STtDjKTiL9ipLykJy7ElCVkRGR/4js7pm7MzM=
AllowedIPs = 10.13.13.2/32, fd00::2/128

# ninox
[Peer]
PublicKey = vEetjpZ9DSPgsrvVRFox1uDHD0J2XrdG7NJFv9Rau2Y=
AllowedIPs = 10.13.13.3/32, fd00::3/128
EOF"

sudo bash -c "cat << EOF > /etc/dnsmasq.d/wg-dns.conf
interface=wg0
listen-address=10.13.13.1
bind-interfaces
server=1.1.1.1
server=1.0.0.1
EOF"

sudo chmod 600 /etc/wireguard/wg0.conf
sudo systemctl enable --now wg-quick@wg0
sudo systemctl restart dnsmasq

echo "=== [3/4] Hardening Host Firewall (IPTables) ==="
sudo sysctl -w net.ipv4.ip_forward=1
sudo iptables -I INPUT 5 -p udp --dport 51820 -j ACCEPT 2>/dev/null || true
sudo iptables -I INPUT 5 -i wg0 -p udp --dport 53 -j ACCEPT 2>/dev/null || true
sudo iptables -I INPUT 5 -i wg0 -p tcp --dport 53 -j ACCEPT 2>/dev/null || true
sudo mkdir -p /etc/iptables
sudo iptables-save | sudo tee /etc/iptables/rules.v4 > /dev/null

echo "=== [4/4] Verification ==="
sudo wg show wg0
echo "=== Setup Complete! ==="
