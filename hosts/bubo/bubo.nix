{ pkgs, ... }:

let
  entrypoint = pkgs.writeShellScriptBin "entrypoint" ''
    set -e
    
    # Create necessary directories
    mkdir -p /data/forgejo/custom/conf
    mkdir -p /data/forgejo/data
    mkdir -p /data/forgejo/repositories

    # Ensure the git user owns the data directory
    chown git:git /data/forgejo
    chown -R git:git /data/forgejo

    echo "Starting Forgejo as git user..."
    exec ${pkgs.su-exec}/bin/su-exec git ${pkgs.forgejo}/bin/forgejo web
  '';
in
pkgs.dockerTools.buildLayeredImage {
  name = "sober-bubo";
  tag = "latest";

  contents = [
    pkgs.forgejo
    pkgs.su-exec
    pkgs.bashInteractive
    pkgs.coreutils
    pkgs.git
    pkgs.cacert
    entrypoint
    (pkgs.writeTextDir "etc/passwd" ''
      root:x:0:0::/root:/bin/bash
      git:x:1000:1000:Git User:/data/forgejo:/bin/bash
    '')
    (pkgs.writeTextDir "etc/group" ''
      root:x:0:
      git:x:1000:
    '')
  ];

  config = {
    Entrypoint = [ "${entrypoint}/bin/entrypoint" ];
    ExposedPorts = {
      "3000/tcp" = { };
      "2222/tcp" = { };
    };
    Env = [
      "PATH=/bin"
      "USER=git"
      "HOME=/data/forgejo"
      "GITEA_WORK_DIR=/data/forgejo"
      "GITEA__database__DB_TYPE=sqlite3"
      "GITEA__database__PATH=/data/forgejo/data/gitea.db"
      "GITEA__server__HTTP_PORT=3000"
      "GITEA__server__START_SSH_SERVER=true"
      "GITEA__server__SSH_PORT=2222"
      "GITEA__server__SSH_LISTEN_PORT=2222"
      "GITEA__security__INSTALL_LOCK=true"
      "GITEA__service__DISABLE_REGISTRATION=true"
      "SSL_CERT_FILE=${pkgs.cacert}/etc/ssl/certs/ca-bundle.crt"
    ];
  };
}
