{
  lib,
  ...
}:

let
  publicKeys = import ../../../lib/public-keys.nix;
in
{
  system.stateVersion = "26.05";
  nixpkgs.hostPlatform = lib.mkDefault "x86_64-linux";
  networking.hostName = "glaucidium";

  # Enable WireGuard VPN Server (Sober VPN Concentrator)
  networking.nat = {
    enable = true;
    externalInterface = "eth0";
    internalInterfaces = [ "wg0" ];
  };

  networking.wireguard.interfaces.wg0 = {
    ips = [
      "10.13.13.1/24"
      "fd00::1/64"
    ];
    listenPort = 51820;
    mtu = 1280;
    privateKeyFile = "/etc/wireguard/private.key";

    peers = [
      {
        # otus
        publicKey = "23wz66STtDjKTiL9ipLykJy7ElCVkRGR/4js7pm7MzM=";
        allowedIPs = [
          "10.13.13.2/32"
          "fd00::2/128"
        ];
      }
      {
        # ninox
        publicKey = "vEetjpZ9DSPgsrvVRFox1uDHD0J2XrdG7NJFv9Rau2Y=";
        allowedIPs = [
          "10.13.13.3/32"
          "fd00::3/128"
        ];
      }
    ];
  };

  networking.firewall = {
    enable = true;
    allowedTCPPorts = [ 22 ];
    allowedUDPPorts = [ 51820 ];
  };

  # --- HARDENED SSH CONFIGURATION (SSH Best Practices) ---
  services.openssh = {
    enable = true;
    settings = {
      PasswordAuthentication = false;
      KbdInteractiveAuthentication = false;
      PermitRootLogin = "prohibit-password";
    };
  };

  users.users.root.openssh.authorizedKeys.keys = [
    publicKeys.glaucidium
    publicKeys.forge
    publicKeys.fly
    publicKeys.nixbuild
    publicKeys.agy
  ];

  # Minimal boot and file system configuration for cloud VPS
  boot.loader.grub.enable = true;
  boot.loader.grub.device = "/dev/sda";
  fileSystems."/" = {
    device = "/dev/sda1";
    fsType = "ext4";
  };
}
