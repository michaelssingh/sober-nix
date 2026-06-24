{
  pkgs,
  soberLib,
  publicKeys,
  ...
}:

soberLib.mkContainerImage {
  name = "sober-surnia";
  packages = [
    pkgs.nix
    pkgs.openssh
    pkgs.git
    pkgs.rsync
    pkgs.tmux
    pkgs.fish
    pkgs.cacert
    pkgs.curl
    pkgs.sudo
    pkgs.coreutils
    pkgs.bash
    pkgs.gnugrep
    pkgs.findutils
  ];
  harden = false;

  users = {
    init = {
      uid = 1000;
      gid = 1000;
      description = "Init User";
      home = "/home/init";
      shell = "${pkgs.fish}/bin/fish";
    };
  };

  exposedPorts = {
    "2222/tcp" = { };
  };

  entrypoint = ''
    # Create necessary dirs
    ${pkgs.coreutils}/bin/mkdir -p /etc/ssh
    ${pkgs.coreutils}/bin/mkdir -p /home/init/.ssh

    # Configure SSHD
    echo "Port 2222" > /etc/ssh/sshd_config
    echo "PermitRootLogin no" >> /etc/ssh/sshd_config
    echo "PasswordAuthentication no" >> /etc/ssh/sshd_config
    echo "PubkeyAuthentication yes" >> /etc/ssh/sshd_config
    echo "UsePAM yes" >> /etc/ssh/sshd_config
    echo "Subsystem sftp internal-sftp" >> /etc/ssh/sshd_config

    # Setup authorized keys for init
    echo "${publicKeys.forge}" > /home/init/.ssh/authorized_keys
    echo "${publicKeys.fly}" >> /home/init/.ssh/authorized_keys
    echo "${publicKeys.nixbuild}" >> /home/init/.ssh/authorized_keys
    echo "${publicKeys.agy}" >> /home/init/.ssh/authorized_keys

    ${pkgs.coreutils}/bin/chmod 700 /home/init/.ssh
    ${pkgs.coreutils}/bin/chmod 600 /home/init/.ssh/authorized_keys
    ${pkgs.coreutils}/bin/chown -R init:init /home/init/.ssh

    # Generate host keys if missing
    ${pkgs.openssh}/bin/ssh-keygen -A

    # Allow passwordless sudo for init
    ${pkgs.coreutils}/bin/mkdir -p /etc/sudoers.d
    echo "init ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/init

    echo "Starting SSHD..."
    exec ${pkgs.openssh}/bin/sshd -D -p 2222
  '';
}
