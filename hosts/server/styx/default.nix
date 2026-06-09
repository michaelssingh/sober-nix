{
  pkgs,
  soberLib,
  publicKeys,
  ...
}:

soberLib.mkContainerImage {
  name = "sober-styx";
  packages = [
    pkgs.nix
    pkgs.openssh
  ];
  harden = true;

  users = {
    sshd = {
      uid = 74;
      gid = 74;
      description = "Privilege-separated SSH";
      home = "/var/empty";
      shell = "/bin/nologin";
    };
  };

  extraContents = [
    (pkgs.writeTextDir "etc/nix/nix.conf" ''
      build-users-group =
      sandbox = false
      experimental-features = nix-command flakes
    '')
  ];

  exposedPorts = {
    "2222/tcp" = { };
  };

  entrypoint = ''
    ${pkgs.coreutils}/bin/mkdir -p /etc/ssh
    echo "Port 2222" > /etc/ssh/sshd_config
    echo "PermitRootLogin prohibit-password" >> /etc/ssh/sshd_config
    echo "PasswordAuthentication no" >> /etc/ssh/sshd_config
    echo "PubkeyAuthentication yes" >> /etc/ssh/sshd_config

    ${pkgs.coreutils}/bin/mkdir -p /root/.ssh
    echo "${publicKeys.fly}" > /root/.ssh/authorized_keys
    chmod 700 /root/.ssh
    chmod 600 /root/.ssh/authorized_keys

    ${pkgs.coreutils}/bin/mkdir -p /var/empty
    ${pkgs.openssh}/bin/ssh-keygen -A

    # Set up Nix socket path and environment variables
    ${pkgs.coreutils}/bin/mkdir -p /nix/var/nix/daemon-socket
    export NIX_BUILD_HOOK= # Prevent build loops

    echo "Starting Nix Daemon..."
    ${pkgs.nix}/bin/nix-daemon &

    echo "Starting SSHD..."
    exec ${pkgs.openssh}/bin/sshd -D -e
  '';
}
