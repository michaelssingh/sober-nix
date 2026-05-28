{ pkgs, publicKeys, ... }:

let
  # Entrypoint to run sshd on port 2222
  # This allows the Fly.io proxy to wake the machine when SSH traffic arrives.
  entrypoint = pkgs.writeShellScriptBin "entrypoint" ''
    set -e

    # 1. Setup SSH Config
    mkdir -p /etc/ssh
    echo "Port 2222" > /etc/ssh/sshd_config
    echo "PermitRootLogin prohibit-password" >> /etc/ssh/sshd_config
    echo "PasswordAuthentication no" >> /etc/ssh/sshd_config
    echo "PubkeyAuthentication yes" >> /etc/ssh/sshd_config

    # 2. Setup Authentication
    mkdir -p /root/.ssh
    echo "${publicKeys.otus}" > /root/.ssh/authorized_keys
    chmod 700 /root/.ssh
    chmod 600 /root/.ssh/authorized_keys

    # 3. Generate Host Keys
    ${pkgs.openssh}/bin/ssh-keygen -A

    # 4. Run SSHD
    exec ${pkgs.openssh}/bin/sshd -D -e
  '';
in
pkgs.dockerTools.buildLayeredImage {
  name = "sober-styx";
  tag = "latest";

  contents = [
    pkgs.nix
    pkgs.bash
    pkgs.coreutils
    pkgs.openssh
    entrypoint
    (pkgs.writeTextDir "etc/passwd" ''
      root:x:0:0::/root:${pkgs.bash}/bin/bash
    '')
    (pkgs.writeTextDir "etc/group" ''
      root:x:0:
    '')
  ];

  config = {
    Entrypoint = [ "${entrypoint}/bin/entrypoint" ];
    ExposedPorts = {
      "2222/tcp" = { };
    };
    Env = [ "PATH=/bin" ];
  };
}
