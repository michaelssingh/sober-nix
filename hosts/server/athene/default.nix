{
  pkgs,
  soberLib,
  inputs,
  ...
}:

let
  enableHeisenbridge = false; # Toggle this to enable/disable

  unstable = import inputs.nixpkgs-unstable {
    system = pkgs.stdenv.hostPlatform.system;
    config = {
      allowUnfree = true;
      permittedInsecurePackages = [
        "olm-3.2.16"
      ];
    };
  };

  mautrix-googlechat = unstable.mautrix-googlechat.overrideAttrs (oldAttrs: {
    postPatch = (oldAttrs.postPatch or "") + ''
      substituteInPlace maugclib/pblite.py \
        --replace-fail "field.label" "getattr(field, 'label', getattr(field, '_label', None))" \
        --replace-fail "field_descriptor.label" "getattr(field_descriptor, 'label', getattr(field_descriptor, '_label', None))"
      substituteInPlace mautrix_googlechat/formatter/from_matrix/gc_message.py \
        --replace-fail "URL = auto()" "URL = \"URL\"" \
        --replace-fail "EMAIL = auto()" "EMAIL = \"EMAIL\""
    '';
    propagatedBuildInputs = (oldAttrs.propagatedBuildInputs or [ ]) ++ [
      unstable.python3Packages.legacy-cgi
      unstable.python3Packages.async-timeout
    ];
  });

  vectorLatest = pkgs.stdenv.mkDerivation {
    pname = "vector";
    version = "0.56.0";
    src = pkgs.fetchzip {
      url = "https://github.com/vectordotdev/vector/releases/download/v0.56.0/vector-0.56.0-x86_64-unknown-linux-musl.tar.gz";
      hash = "sha256-XVgWSdMgd/CzNG7oWxTfJUZbGFrDu9gbHgVNTq0onAo=";
    };
    installPhase = ''
      mkdir -p $out/bin
      cp bin/vector $out/bin/vector
    '';
  };

  # The configuration profile for the Soju IRC bouncer
  sojuConfig = pkgs.writeText "soju.conf" ''
    listen irc+insecure://0.0.0.0:6667
    listen unix+admin:///data/admin.sock
    listen http+insecure://0.0.0.0:8081
    http-ingress https://sober-athene.fly.dev
    file-upload fs /data/uploads
    db sqlite3 /data/soju.db
  '';

  # Declarative configuration for the local Conduit homeserver
  conduitConfig = pkgs.writeText "conduit.toml" ''
    [global]
    server_name = "sober.fyi"
    port = 6167
    address = "0.0.0.0"
    database_path = "/data/conduit"
    database_backend = "rocksdb"
    allow_registration = true
    allow_federation = false
    max_request_size = 20971520
    log = "warn"
    appservice_registration = ["/data/mautrix-googlechat/registration.yaml" ${pkgs.lib.optionalString enableHeisenbridge ", \"/data/heisenbridge/registration.yaml\""}]
  '';

in
soberLib.mkContainerImage {
  name = "sober-athene";
  harden = false;
  env = {
    PROTOCOL_BUFFERS_PYTHON_IMPLEMENTATION = "python";
  };
  observability = {
    package = vectorLatest;
    lokiUrl = "https://logs-prod-042.grafana.net";
    prometheusUrl = "https://prometheus-prod-66-prod-us-east-3.grafana.net/api/prom/push";
    apiKeyFile = "/run/secrets/grafana_api_key";
  };
  packages = [
    pkgs.soju
    unstable.matrix-conduit
    mautrix-googlechat
    pkgs.curl
    pkgs.jq
    pkgs.sqlite
    pkgs.binutils
    pkgs.gnugrep
    pkgs.gnutar
  ]
  ++ pkgs.lib.optional enableHeisenbridge pkgs.heisenbridge
  ++ [ inputs.self.packages.${pkgs.stdenv.hostPlatform.system}.appservice-mgr ];
  exposedPorts = {
    "6667/tcp" = { };
    "6167/tcp" = { };
    "8081/tcp" = { };
  };
  entrypoint = ''
    # Configure aggressive TCP keepalive settings to clean up dead connections quickly
    echo 300 > /proc/sys/net/ipv4/tcp_keepalive_time
    echo 15 > /proc/sys/net/ipv4/tcp_keepalive_intvl
    echo 5 > /proc/sys/net/ipv4/tcp_keepalive_probes

    # Create persistent directories inside the mounted volume
    ${pkgs.coreutils}/bin/mkdir -p /data/uploads
    ${pkgs.coreutils}/bin/mkdir -p /data/conduit
    ${pkgs.coreutils}/bin/mkdir -p /data/mautrix-googlechat
    ${pkgs.lib.optionalString enableHeisenbridge "${pkgs.coreutils}/bin/mkdir -p /data/heisenbridge"}

    # Symlink config to default path so that sojuctl/sojudb default configurations work
    ${pkgs.coreutils}/bin/mkdir -p /etc/soju
    ${pkgs.coreutils}/bin/ln -sf ${sojuConfig} /etc/soju/config

    # Explicitly overwrite conduit configuration
    ${pkgs.coreutils}/bin/cp ${conduitConfig} /data/conduit/conduit.toml

    # 1. Setup mautrix-googlechat configuration
    echo "Initializing/Patching mautrix-googlechat configuration..."
    if [ ! -f /data/mautrix-googlechat/config.yaml ]; then
        cp ${mautrix-googlechat}/share/mautrix-googlechat/example-config.yaml /data/mautrix-googlechat/config.yaml
    fi
    chmod +w /data/mautrix-googlechat/config.yaml

    echo "Ensuring config.yaml keys are correct..."
    ${pkgs.yq-go}/bin/yq -i '
      .homeserver.address = "http://localhost:6167" |
      .homeserver.domain = "sober.fyi" |
      .appservice.address = "http://localhost:29318" |
      .appservice.hostname = "127.0.0.1" |
      .appservice.port = 29318 |
      .appservice.database = "sqlite:/data/mautrix-googlechat/mautrix-googlechat.db" |
      .bridge.encryption.allow = false |
      .bridge.encryption.default = false |
      .bridge.state.enabled = false |
      .bridge.m_bridge = false |
      .bridge.displayname_template = "{full_name}" |
      del(.bridge.permissions) |
      .bridge.permissions["sober.fyi"] = "user" |
      .bridge.permissions["@init:sober.fyi"] = "admin" |
      .bridge.permissions["@googlechatbot:sober.fyi"] = "admin"
    ' /data/mautrix-googlechat/config.yaml

    # 2. Pre-generate registration files so Conduit can load them on boot
    if [ ! -f /data/mautrix-googlechat/registration.yaml ]; then
        echo "Generating mautrix-googlechat appservice registration file..."
        ${mautrix-googlechat}/bin/mautrix-googlechat -g \
           -c /data/mautrix-googlechat/config.yaml \
           -r /data/mautrix-googlechat/registration.yaml
    fi

    ${pkgs.lib.optionalString enableHeisenbridge ''
      if [ ! -f /data/heisenbridge/registration.yaml ]; then
          echo "Generating heisenbridge appservice registration file..."
          ${pkgs.heisenbridge}/bin/heisenbridge \
              -c /data/heisenbridge/registration.yaml \
              --generate-compat \
              -l 127.0.0.1 -p 6668 \
              http://localhost:6167
      fi
    ''}

    # 3. Start Conduit Matrix homeserver in the background
    echo "Starting Conduit homeserver..."
    CONDUIT_CONFIG=${conduitConfig} ${unstable.matrix-conduit}/bin/conduit 2>&1 | tee -a /var/log/conduit.log &

    # Wait a few seconds for Conduit to initialize its API
    sleep 5

    # 4. Perform automatic appservice registration via the Go tool
    echo "Running appservice-mgr to register bridges with Conduit..."
    appservice-mgr /data 2>&1 | tee -a /var/log/appservice-mgr.log &

    # 5. Start mautrix-googlechat bridge in the background
    echo "Starting mautrix-googlechat bridge..."
    ${mautrix-googlechat}/bin/mautrix-googlechat -c /data/mautrix-googlechat/config.yaml 2>&1 | tee -a /var/log/mautrix.log &

    # 6. Start heisenbridge gateway in the background
    ${pkgs.lib.optionalString enableHeisenbridge ''
      echo "Starting heisenbridge..."
      ${pkgs.heisenbridge}/bin/heisenbridge \
          -v \
          -c /data/heisenbridge/registration.yaml \
          -l 127.0.0.1 -p 6668 \
          http://localhost:6167 2>&1 | tee -a /var/log/heisenbridge.log &
    ''}

    # Wait a few seconds for background services to initialize
    sleep 3

    # 7. Execute Soju bouncer in the foreground
    echo "Starting Soju IRC bouncer..."
    rm -f /data/admin.sock
    exec ${pkgs.soju}/bin/soju -config ${sojuConfig} 2>&1 | tee -a /var/log/soju.log
  '';
}
