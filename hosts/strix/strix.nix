{ pkgs, lib, ... }:

let
  rustypasteConfigTemplate = pkgs.writeText "config-template.toml" ''
    [config]
    refresh_rate = "1s"

    [server]
    address = "0.0.0.0:8000"
    url = "@URL@"
    max_content_length = "100MB"
    upload_path = "/data/upload"
    timeout = "30s"
    expose_version = false
    expose_list = false
    handle_spaces = "replace"

    [landing_page]
    text = "sober-strix pastebin. Please use the CLI tool or curl to upload pastes."
    content_type = "text/plain; charset=utf-8"

    [paste]
    random_url = { type = "petname", words = 2, separator = "-" }
    default_extension = "txt"

    [[paste.mime_override]]
    mime = "text/plain; charset=utf-8"
    regex = "^.*\\.(nix|go|sh|conf|toml|yaml|yml|json|txt|md|log|kbd|diff|patch|ini|cfg)$"
  '';

  entrypoint = pkgs.writeShellScriptBin "entrypoint" ''
    set -e
    mkdir -p /data/upload

    # Set public URL dynamically based on Fly.io app name
    APP_NAME="''${FLY_APP_NAME:-sober-strix}"
    PUBLIC_URL="https://$APP_NAME.fly.dev"

    echo "Configuring rustypaste with URL: $PUBLIC_URL"
    mkdir -p /etc/rustypaste
    sed "s|@URL@|$PUBLIC_URL|g" ${rustypasteConfigTemplate} > /etc/rustypaste/config.toml

    export CONFIG="/etc/rustypaste/config.toml"

    # Run rustypaste
    exec ${pkgs.rustypaste}/bin/rustypaste
  '';
in
pkgs.dockerTools.buildLayeredImage {
  name = "sober-strix";
  tag = "latest";

  contents = [
    pkgs.rustypaste
    pkgs.bash
    pkgs.coreutils
    pkgs.gnused
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
      "8000/tcp" = { };
    };
    Env = [
      "PATH=/bin"
      "SSL_CERT_FILE=${pkgs.cacert}/etc/ssl/certs/ca-bundle.crt"
    ];
  };
}
