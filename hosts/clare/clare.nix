{ pkgs, lib, ... }:

let
  # The Go SSH portal application
  clareShell = pkgs.buildGoModule {
    pname = "clare-shell";
    version = "0.1.0";
    src = ./shell;
    vendorHash = "sha256-srIM6kwFuOF9yo5Opi7PDFad0ohToyg2EwqLq40yRUM=";
  };

  # The Go Raki API layer for Soju
  rakiApi = pkgs.buildGoModule {
    pname = "raki-api";
    version = "0.1.0";
    src = ../../packages/raki-api;
    vendorHash = null;
  };

  # The config for soju in the container
  sojuConfig = pkgs.writeTextDir "etc/soju/config" ''
    listen irc+insecure://0.0.0.0:6697
    listen unix+admin:///var/lib/soju/admin.sock
    listen http+insecure://0.0.0.0:8080
    http-ingress https://sober-clare.fly.dev
    file-upload fs /var/lib/soju/uploads
    db sqlite3 /var/lib/soju/soju.db
  '';

  # Minimal Ergo IRCd config for testing
  ergoConfig = pkgs.writeTextDir "etc/ergo/ergo.yaml" ''
    server:
        name: irc.local
        listen:
            - "127.0.0.1:6667"
    network:
        name: local-test-net
    datastore:
        path: /var/lib/ergo/ergo.db
    accounts:
        registration:
            enabled: false
    logging:
        - method: stdout
          level: debug
  '';

  # A wrapper script to start soju, the api, and the SSH portal
  entrypoint = pkgs.writeShellScriptBin "entrypoint" ''
    set -e
    mkdir -p /var/lib/soju/uploads
    mkdir -p /var/lib/ergo
    
    # Start Ergo IRCd in the background
    ${pkgs.ergochat}/bin/ergo run --conf /etc/ergo/ergo.yaml &
    
    # Start soju in the background
    ${pkgs.soju}/bin/soju &
    
    # Wait for the socket to exist
    sleep 2
    
    # Start the API in the background
    ${rakiApi}/bin/raki-api -socket /var/lib/soju/admin.sock -listen :8081 -api-keys "''${RAKI_API_KEY:-default-insecure-key}" &
    
    # Start the custom SSH shell portal in the foreground
    exec ${clareShell}/bin/clare-shell
  '';
in
pkgs.dockerTools.buildLayeredImage {
  name = "sober-clare";
  tag = "latest";

  contents = [
    pkgs.soju
    pkgs.ergochat
    clareShell
    rakiApi
    pkgs.bashInteractive
    pkgs.coreutils
    pkgs.cacert
    pkgs.curl
    pkgs.netcat-openbsd
    pkgs.procps
    pkgs.findutils
    pkgs.sqlite
    sojuConfig
    ergoConfig
    entrypoint
    (pkgs.writeTextDir "etc/passwd" ''
      root:x:0:0::/root:/bin/bash
    '')
    (pkgs.writeTextDir "etc/group" ''
      root:x:0:
    '')
  ];

  config = {
    Entrypoint = [ "${entrypoint}/bin/entrypoint" ];
    ExposedPorts = {
      "6697/tcp" = { };
      "8081/tcp" = { };
      "2222/tcp" = { };
    };
    Env = [
      "PATH=/bin"
      "SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt"
    ];
  };
}
