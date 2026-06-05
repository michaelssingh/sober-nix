{ pkgs, soberLib, ... }:

let
  # The SSH shell portal controls authentication and user shells
  clareShell = pkgs.buildGoModule {
    pname = "clare-shell";
    version = "0.1.0";
    src = ./shell;
    vendorHash = "sha256-srIM6kwFuOF9yo5Opi7PDFad0ohToyg2EwqLq40yRUM=";
  };

  # Raki API manages Soju administration dynamically
  rakiApi = pkgs.buildGoModule {
    pname = "raki-api";
    version = "0.1.0";
    src = ../../../packages/raki-api;
    vendorHash = null;
  };

  # Declarative Soju bouncer configuration
  sojuConfig = pkgs.writeTextDir "etc/soju/config" ''
    listen irc+insecure://0.0.0.0:6697
    listen unix+admin:///var/lib/soju/admin.sock
    listen http+insecure://0.0.0.0:8080
    http-ingress https://sober-clare.fly.dev
    file-upload fs /var/lib/soju/uploads
    db sqlite3 /var/lib/soju/soju.db
  '';

  # Minimal Ergo IRC server configuration for local message routing
  ergoConfig = pkgs.writeTextDir "etc/ergo/ergo.yaml" ''
    network:
        name: local-test-net
    server:
        name: irc.local
        listeners:
            "127.0.0.1:6667": {}
        max-sendq: 1M
    datastore:
        path: /var/lib/ergo/ergo.db
    accounts:
        registration:
            enabled: false
    limits:
        nicklen: 32
        identlen: 20
        realnamelen: 150
        channellen: 64
        awaylen: 390
        kicklen: 390
        topiclen: 390
  '';
in
soberLib.mkContainerImage {
  name = "sober-clare";
  harden = false; # Needs developmental tools/compilers inside the container for dynamic shells
  packages = [
    pkgs.soju
    pkgs.ergochat
    clareShell
    rakiApi
    pkgs.curl
    pkgs.netcat-openbsd
    pkgs.procps
    pkgs.findutils
    pkgs.sqlite
    pkgs.go_1_26
    pkgs.git
  ];
  extraContents = [ sojuConfig ergoConfig ];
  exposedPorts = {
    "6697/tcp" = {};
    "8081/tcp" = {};
    "2222/tcp" = {};
  };
  entrypoint = ''
    ulimit -n 65536 || true
    
    # Setup folders on persistent volume mount
    mkdir -p /var/lib/soju/uploads
    mkdir -p /var/lib/ergo
    
    # Initialize the local Ergo database if it's missing
    if [ ! -f /var/lib/ergo/ergo.db ]; then
        ergo initdb --conf /etc/ergo/ergo.yaml
    fi
    
    # Run services in the background (Ergo daemon, Soju bouncer)
    ergo run --conf /etc/ergo/ergo.yaml &
    soju &
    sleep 2
    
    # Resolve the API binary path, allowing path override for hotfixes or active testing
    API_BIN=${rakiApi}/bin/raki-api
    if [ -f /var/lib/soju/raki-api ]; then
        chmod +x /var/lib/soju/raki-api
        API_BIN=/var/lib/soju/raki-api
    fi
    $API_BIN -socket /var/lib/soju/admin.sock -listen :8081 -api-keys "''${RAKI_API_KEY:-default-insecure-key}" &
    
    # Execute the primary shell portal in the foreground
    exec ${clareShell}/bin/clare-shell
  '';
}
