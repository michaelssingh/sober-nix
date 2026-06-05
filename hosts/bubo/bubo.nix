{ pkgs, ... }:

let
  entrypoint = pkgs.writeShellScriptBin "entrypoint" ''
    set -e
    
    # Create necessary directories
    mkdir -p /data/forgejo/custom/conf
    mkdir -p /data/forgejo/data
    mkdir -p /data/forgejo/repositories

    # Dynamically write default app.ini if missing to skip the installation wizard
    APP_NAME="''${FLY_APP_NAME:-sober-bubo}"
    if [ ! -f /data/forgejo/custom/conf/app.ini ]; then
      echo "Initializing minimal app.ini configuration..."
      cat <<EOF > /data/forgejo/custom/conf/app.ini
[security]
INSTALL_LOCK = true

[database]
DB_TYPE = sqlite3
PATH = /data/forgejo/data/gitea.db

[server]
ROOT_URL = https://$APP_NAME.fly.dev/
HTTP_PORT = 3000
START_SSH_SERVER = true
SSH_PORT = 2222
SSH_LISTEN_PORT = 2222

[service]
DISABLE_REGISTRATION = true
EOF
    fi

    # Ensure the git user owns the data directory
    chown git:git /data/forgejo
    chown -R git:git /data/forgejo

    echo "Starting Forgejo as git user..."
    exec ${pkgs.su-exec}/bin/su-exec git ${pkgs.forgejo}/bin/forgejo web
  '';

  usrBinEnv = pkgs.runCommand "usr-bin-env" {} ''
    mkdir -p $out/usr/bin
    ln -s ${pkgs.coreutils}/bin/env $out/usr/bin/env
  '';

  tmpDir = pkgs.runCommand "tmp-dir" {} ''
    mkdir -p $out/tmp
    chmod 1777 $out/tmp
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
    usrBinEnv
    tmpDir
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
