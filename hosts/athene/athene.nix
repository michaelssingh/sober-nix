{ pkgs, lib, ... }:

let
  # The config for soju in the container
  sojuConfig = pkgs.writeText "soju.conf" ''
    listen irc+insecure://0.0.0.0:6697
    listen unix+admin:///var/lib/soju/admin.sock
    listen http+insecure://0.0.0.0:8080
    http-ingress https://sober-athene.fly.dev
    file-upload fs /var/lib/soju/uploads
    db sqlite3 /var/lib/soju/soju.db
  '';

  # A wrapper script to ensure directories exist before starting soju
  entrypoint = pkgs.writeShellScriptBin "entrypoint" ''
    set -e
    mkdir -p /var/lib/soju/uploads
    # Ensure the DB file exists in the volume, copying from root if missing (for bootstrap)
    if [ ! -f /var/lib/soju/soju.db ] && [ -f /soju.db ]; then
        cp /soju.db /var/lib/soju/soju.db
    fi
    exec ${pkgs.soju}/bin/soju -config ${sojuConfig}
  '';
in
pkgs.dockerTools.buildLayeredImage {
  name = "sober-athene";
  tag = "latest";

  contents = [
    pkgs.soju
    pkgs.bash
    pkgs.coreutils
    pkgs.cacert
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
      "8080/tcp" = { };
    };
    Env = [ "PATH=/bin" "SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt" ];
  };
}
