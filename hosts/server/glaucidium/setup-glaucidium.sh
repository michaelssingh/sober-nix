#!/usr/bin/env bash
# ==============================================================================
# Setup Script for glaucidium (Oracle Cloud Always Free VPN Concentrator)
# Reproducibly provisions WireGuard VPN server and firewall on Ubuntu 24.04
# ==============================================================================
set -euo pipefail

echo "=== [1/4] Installing Required Packages ==="
sudo apt-get update -qq
sudo curl -fsSL https://pkg.cloudflareclient.com/pubkey.gpg | sudo gpg --yes --dearmor --output /usr/share/keyrings/cloudflare-warp-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/cloudflare-warp-archive-keyring.gpg] https://pkg.cloudflareclient.com/ noble main" | sudo tee /etc/apt/sources.list.d/cloudflare-warp.list
sudo apt-get update -qq
sudo apt-get install -y wireguard wireguard-tools iptables iptables-persistent dnsmasq redsocks cloudflare-warp ipset bind9-host

echo "=== [2/4] Configuring WireGuard Server Interface (wg0) & Services ==="
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
cache-size=10000
EOF"

sudo warp-cli --accept-tos registration new 2>/dev/null || true
sudo warp-cli --accept-tos mode proxy 2>/dev/null || true
sudo warp-cli --accept-tos proxy port 4000 2>/dev/null || true
sudo warp-cli --accept-tos connect 2>/dev/null || true

sudo bash -c "cat << EOF > /etc/redsocks.conf
base {
 log_debug = off;
 log_info = on;
 log = \"syslog:daemon\";
 daemon = on;
 user = redsocks;
 group = redsocks;
 redirector = iptables;
}

redsocks {
 local_ip = 0.0.0.0;
 local_port = 12345;
 ip = 127.0.0.1;
 port = 4000;
 type = socks5;
}
EOF"

sudo chmod 600 /etc/wireguard/wg0.conf
sudo systemctl enable --now wg-quick@wg0
sudo systemctl restart dnsmasq redsocks

sudo ipset create vpn_bypass hash:ip 2>/dev/null || true

sudo bash -c "cat << 'EOF' > /usr/local/bin/update-vpn-bypass.sh
#!/bin/bash
DOMAINS="imgur.com i.imgur.com reddit.com www.reddit.com v.redd.it i.redd.it anidb.net api.anidb.net proton.me mail.proton.me account.proton.me api.protonmail.ch protonmail.com"
for domain in \$DOMAINS; do
  ips=\$(host -t A \$domain 1.1.1.1 2>/dev/null | grep \"has address\" | awk '{print \$NF}')
  for ip in \$ips; do
    ipset add vpn_bypass \$ip 2>/dev/null || true
  done
done
EOF"

sudo chmod +x /usr/local/bin/update-vpn-bypass.sh
sudo /usr/local/bin/update-vpn-bypass.sh

sudo bash -c "cat << EOF > /etc/systemd/system/update-vpn-bypass.service
[Unit]
Description=Auto-update IPSet vpn_bypass list for Imgur, Reddit, and AniDB
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/update-vpn-bypass.sh
EOF"

sudo bash -c "cat << EOF > /etc/systemd/system/update-vpn-bypass.timer
[Unit]
Description=Run update-vpn-bypass service every 6 hours

[Timer]
OnBootSec=1min
OnUnitActiveSec=6h

[Install]
WantedBy=timers.target
EOF"

sudo systemctl daemon-reload
sudo systemctl enable --now update-vpn-bypass.timer
sudo systemctl enable --now unattended-upgrades

echo "=== [3/4] Hardening Host Firewall (IPTables) ==="
sudo sysctl -w net.ipv4.ip_forward=1
sudo iptables -I INPUT 5 -p udp --dport 51820 -j ACCEPT 2>/dev/null || true
sudo iptables -I INPUT 5 -i wg0 -p udp --dport 53 -j ACCEPT 2>/dev/null || true
sudo iptables -I INPUT 5 -i wg0 -p tcp --dport 53 -j ACCEPT 2>/dev/null || true
sudo iptables -I INPUT 5 -i wg0 -p tcp --dport 12345 -j ACCEPT 2>/dev/null || true
sudo iptables -t nat -I PREROUTING 1 -i wg0 -p tcp -m set --match-set vpn_bypass dst -j REDIRECT --to-ports 12345 2>/dev/null || true
sudo mkdir -p /etc/iptables
sudo iptables-save | sudo tee /etc/iptables/rules.v4 > /dev/null

echo "=== [4/4] Verification ==="
sudo wg show wg0
echo "=== Setup Complete! ==="
