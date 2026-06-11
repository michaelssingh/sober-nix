{
  pkgs,
  soberLib,
  inputs,
  ...
}:

let
  unstable = import inputs.nixpkgs-unstable {
    system = pkgs.stdenv.hostPlatform.system;
    config = {
      allowUnfree = true;
      permittedInsecurePackages = [
        "olm-3.2.16"
      ];
    };
  };

  mautrix-googlechat =
    (unstable.mautrix-googlechat.override {
      python3 = unstable.python312;
    }).overrideAttrs
      (oldAttrs: {
        propagatedBuildInputs =
          (builtins.filter (p: p.pname or "" != "protobuf") oldAttrs.propagatedBuildInputs)
          ++ [
            unstable.python312Packages.async-timeout
            unstable.python312Packages.protobuf4
          ];
      });

  # The configuration profile for the Soju IRC bouncer
  sojuConfig = pkgs.writeText "soju.conf" ''
    listen irc+insecure://0.0.0.0:6697
    listen unix+admin:///var/lib/soju/admin.sock
    listen http+insecure://0.0.0.0:8080
    http-ingress http://sober-athene.fly.dev
    file-upload fs /var/lib/soju/uploads
    db sqlite3 /var/lib/soju/soju.db
  '';

  # Declarative configuration for the local Conduit homeserver
  conduitConfig = pkgs.writeText "conduit.toml" ''
    [global]
    server_name = "sober.fyi"
    port = 6167
    address = "0.0.0.0"
    database_path = "/var/lib/soju/conduit"
    database_backend = "rocksdb"
    allow_registration = true
    allow_federation = false
    max_request_size = 20971520
    log = "warn"
    appservice_registration = "/var/lib/soju/mautrix-googlechat/registration.yaml"
    appservice_registration = "/var/lib/soju/heisenbridge/registration.yaml"
  '';
in
soberLib.mkContainerImage {
  name = "sober-athene";
  harden = false; # Needed to run multiple background services and dynamic commands
  observability = {
    lokiUrl = "https://logs-prod-042.grafana.net/loki/api/v1/push";
    prometheusUrl = "https://prometheus-prod-66-prod-us-east-3.grafana.net/api/prom/push";
    apiKeyFile = "/run/secrets/grafana_api_key";
  };
  packages = [
    pkgs.soju
    unstable.matrix-conduit
    mautrix-googlechat
    pkgs.heisenbridge
    pkgs.yq-go
    pkgs.sqlite
    pkgs.curl
    pkgs.coreutils
  ];
  exposedPorts = {
    "6697/tcp" = { };
    "8080/tcp" = { };
    "6167/tcp" = { };
  };
  entrypoint = ''
    # Configure aggressive TCP keepalive settings to clean up dead connections quickly
    echo 300 > /proc/sys/net/ipv4/tcp_keepalive_time
    echo 15 > /proc/sys/net/ipv4/tcp_keepalive_intvl
    echo 5 > /proc/sys/net/ipv4/tcp_keepalive_probes

    # Create persistent directories inside the mounted volume
    ${pkgs.coreutils}/bin/mkdir -p /var/lib/soju/uploads
    ${pkgs.coreutils}/bin/mkdir -p /var/lib/soju/conduit
    ${pkgs.coreutils}/bin/mkdir -p /var/lib/soju/mautrix-googlechat

    # Symlink config to default path so that sojuctl/sojudb default configurations work
    ${pkgs.coreutils}/bin/mkdir -p /etc/soju
    ${pkgs.coreutils}/bin/ln -sf ${sojuConfig} /etc/soju/config

    # 1. Start Conduit Matrix homeserver in the background
    echo "Starting Conduit homeserver..."
    CONDUIT_CONFIG=${conduitConfig} ${unstable.matrix-conduit}/bin/conduit &

    # 2. Setup and start mautrix-googlechat bridge in the background
    if [ ! -f /var/lib/soju/mautrix-googlechat/config.yaml ]; then
        echo "Initializing mautrix-googlechat configuration..."
        cp ${mautrix-googlechat}/share/mautrix-googlechat/example-config.yaml /var/lib/soju/mautrix-googlechat/config.yaml
        chmod +w /var/lib/soju/mautrix-googlechat/config.yaml

        echo "Patching config.yaml keys..."
        ${pkgs.yq-go}/bin/yq -i '
          .homeserver.address = "http://localhost:6167" |
          .homeserver.domain = "sober.fyi" |
          .appservice.address = "http://localhost:29318" |
          .appservice.hostname = "127.0.0.1" |
          .appservice.port = 29318 |
          .appservice.database = "sqlite:///var/lib/soju/mautrix-googlechat/mautrix-googlechat.db" |
          .appservice.namespaces.users[0].regex = "@googlechat_.*:sober\\.fyi" |
          .appservice.namespaces.users[1].regex = "@googlechatbot:sober\\.fyi" |
          .encryption.allow = false |
          .encryption.default = false |
          del(.bridge.permissions) |
          .bridge.permissions["sober.fyi"] = "user" |
          .bridge.permissions["@init:sober.fyi"] = "admin" |
          .bridge.permissions["@googlechatbot:sober.fyi"] = "admin"
        ' /var/lib/soju/mautrix-googlechat/config.yaml

        echo "Generating appservice registration file..."
        ${mautrix-googlechat}/bin/mautrix-googlechat -g \
          -c /var/lib/soju/mautrix-googlechat/config.yaml \
          -r /var/lib/soju/mautrix-googlechat/registration.yaml
    fi

    echo "Starting mautrix-googlechat bridge..."
    ${mautrix-googlechat}/bin/mautrix-googlechat -c /var/lib/soju/mautrix-googlechat/config.yaml &

    # 3. Start heisenbridge gateway in the background
    echo "Starting heisenbridge..."
    ${pkgs.coreutils}/bin/mkdir -p /var/lib/soju/heisenbridge

    # Regenerate config to ensure fresh registration/token compatibility
    ${pkgs.heisenbridge}/bin/heisenbridge \
        -c /var/lib/soju/heisenbridge/config.yaml \
        --generate-compat \
        -l 127.0.0.1 -p 6668 \
        http://localhost:6167
        
    ${pkgs.heisenbridge}/bin/heisenbridge \
        -c /var/lib/soju/heisenbridge/config.yaml \
        -l 127.0.0.1 -p 6668 \
        http://localhost:6167 &

    # Wait a few seconds for background services to initialize
    sleep 3

    # 4. Execute Soju bouncer in the foreground
    echo "Starting Soju IRC bouncer..."
    exec ${pkgs.soju}/bin/soju -config ${sojuConfig}
  '';
}
