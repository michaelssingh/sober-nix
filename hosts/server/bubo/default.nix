{ pkgs, soberLib, ... }:

soberLib.mkContainerImage {
  name = "sober-bubo";
  packages = [
    pkgs.forgejo
    pkgs.su-exec
    pkgs.git
  ];
  usrBinEnv = true; # Enable /usr/bin/env to support Git push pre-receive hooks
  harden = true;

  users = {
    git = {
      uid = 1000;
      gid = 1000;
      description = "Git User";
      home = "/data/forgejo";
    };
  };

  exposedPorts = {
    "3000/tcp" = { };
    "2222/tcp" = { };
  };

  env = {
    USER = "git";
    HOME = "/data/forgejo";
    GITEA_WORK_DIR = "/data/forgejo";
    GITEA__database__DB_TYPE = "sqlite3";
    GITEA__database__PATH = "/data/forgejo/data/gitea.db";
    GITEA__server__HTTP_PORT = "3000";
    GITEA__server__START_SSH_SERVER = "true";
    GITEA__server__SSH_PORT = "2222";
    GITEA__server__SSH_LISTEN_PORT = "2222";
    GITEA__security__INSTALL_LOCK = "true";
    GITEA__service__DISABLE_REGISTRATION = "true";
  };

  entrypoint = ''
        # Create persistent directories inside the mounted volume
        ${pkgs.coreutils}/bin/mkdir -p /data/forgejo/custom/conf
        ${pkgs.coreutils}/bin/mkdir -p /data/forgejo/data
        ${pkgs.coreutils}/bin/mkdir -p /data/forgejo/repositories

        # Skip setup wizard by writing the configuration if it is missing
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

        # Restrict permissions and drop privileges to git user
        ${pkgs.coreutils}/bin/chown git:git /data/forgejo
        ${pkgs.coreutils}/bin/chown -R git:git /data/forgejo

        echo "Starting Forgejo..."
        exec ${pkgs.su-exec}/bin/su-exec git ${pkgs.forgejo}/bin/forgejo web
  '';
}
