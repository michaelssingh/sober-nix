{ pkgs, publicKeys, ... }:

let
  # Entrypoint to run sshd and nix-daemon
  # This allows the machine to act as a Nix remote builder over Fly.io.
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
    echo "${publicKeys.fly}" > /root/.ssh/authorized_keys
    chmod 700 /root/.ssh
    chmod 600 /root/.ssh/authorized_keys

    # 3. Generate Host Keys
    mkdir -p /var/empty
    ${pkgs.openssh}/bin/ssh-keygen -A

    # 4. Setup Nix environment
    mkdir -p /nix/var/nix/daemon-socket
    export NIX_BUILD_HOOK= # Prevent build loops

    # 5. Start Nix Daemon
    echo "Starting Nix Daemon..."
    ${pkgs.nix}/bin/nix-daemon &

    # 6. Run SSHD
    echo "Starting SSHD..."
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
      sshd:x:74:74:Privilege-separated SSH:/var/empty:/bin/nologin
    '')
    (pkgs.writeTextDir "etc/group" ''
      root:x:0:
      sshd:x:74:
    '')
    (pkgs.writeTextDir "etc/nix/nix.conf" ''
      build-users-group =
      sandbox = false
      experimental-features = nix-command flakes
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
