{ pkgs, soberLib, ... }:

let
  # The configuration profile for the Soju IRC bouncer
  sojuConfig = pkgs.writeText "soju.conf" ''
    listen irc+insecure://0.0.0.0:6697
    listen unix+admin:///var/lib/soju/admin.sock
    listen http+insecure://0.0.0.0:8080
    http-ingress https://sober-athene.fly.dev
    file-upload fs /var/lib/soju/uploads
    db sqlite3 /var/lib/soju/soju.db
  '';

  # Declarative configuration for the local Conduit homeserver
  conduitConfig = pkgs.writeText "conduit.toml" ''
    [global]
    server_name = "athene.local"
    port = 6167
    address = "127.0.0.1"
    database_path = "/var/lib/soju/conduit"
    database_backend = "rocksdb"
    allow_registration = true
    allow_federation = false
    max_request_size = 20971520
    log = "warn"
  '';
in
soberLib.mkContainerImage {
  name = "sober-athene";
  harden = false; # Needed to run multiple background services and dynamic commands
  packages = [
    pkgs.soju
    pkgs.unstable.matrix-conduit
    pkgs.unstable.mautrix-googlechat
    pkgs.matrirc
    pkgs.yq-go
    pkgs.sqlite
    pkgs.curl
    pkgs.coreutils
  ];
  exposedPorts = {
    "6697/tcp" = {};
    "8080/tcp" = {};
  };
  entrypoint = ''
    # Create persistent directories inside the mounted volume
    ${pkgs.coreutils}/bin/mkdir -p /var/lib/soju/uploads
    ${pkgs.coreutils}/bin/mkdir -p /var/lib/soju/conduit
    ${pkgs.coreutils}/bin/mkdir -p /var/lib/soju/mautrix-googlechat
    ${pkgs.coreutils}/bin/mkdir -p /var/lib/soju/matrirc

    # Symlink config to default path so that sojuctl/sojudb default configurations work
    ${pkgs.coreutils}/bin/mkdir -p /etc/soju
    ${pkgs.coreutils}/bin/ln -sf ${sojuConfig} /etc/soju/config

    # 1. Start Conduit Matrix homeserver in the background
    echo "Starting Conduit homeserver..."
    CONDUIT_CONFIG=${conduitConfig} ${pkgs.unstable.matrix-conduit}/bin/matrix-conduit &

    # 2. Setup and start mautrix-googlechat bridge in the background
    if [ ! -f /var/lib/soju/mautrix-googlechat/config.yaml ]; then
        echo "Initializing mautrix-googlechat configuration..."
        ${pkgs.unstable.mautrix-googlechat}/bin/mautrix-googlechat -e -c /var/lib/soju/mautrix-googlechat/config.yaml

        echo "Patching config.yaml keys..."
        ${pkgs.yq-go}/bin/yq -i '
          .homeserver.address = "http://localhost:6167" |
          .homeserver.domain = "athene.local" |
          .appservice.address = "http://localhost:29318" |
          .appservice.hostname = "127.0.0.1" |
          .appservice.port = 29318 |
          .appservice.database = "sqlite:///var/lib/soju/mautrix-googlechat/mautrix-googlechat.db" |
          .encryption.allow = false |
          .encryption.default = false
        ' /var/lib/soju/mautrix-googlechat/config.yaml

        echo "Generating appservice registration file..."
        ${pkgs.unstable.mautrix-googlechat}/bin/mautrix-googlechat -g \
          -c /var/lib/soju/mautrix-googlechat/config.yaml \
          -r /var/lib/soju/mautrix-googlechat/registration.yaml
    fi

    echo "Starting mautrix-googlechat bridge..."
    ${pkgs.unstable.mautrix-googlechat}/bin/mautrix-googlechat -c /var/lib/soju/mautrix-googlechat/config.yaml &

    # 3. Start matrirc gateway in the background
    echo "Starting matrirc..."
    ${pkgs.matrirc}/bin/matrirc \
        -l 127.0.0.1:6668 \
        --allow-register \
        --state-dir /var/lib/soju/matrirc &

    # Wait a few seconds for background services to initialize
    sleep 3

    # 4. Execute Soju bouncer in the foreground
    echo "Starting Soju IRC bouncer..."
    exec ${pkgs.soju}/bin/soju -config ${sojuConfig}
  '';
}
