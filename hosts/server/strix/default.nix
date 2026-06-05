{ pkgs, soberLib, ... }:

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
in
soberLib.mkContainerImage {
  name = "sober-strix";
  packages = [ pkgs.rustypaste pkgs.gnused ];
  harden = true;
  exposedPorts = {
    "8000/tcp" = {};
  };
  entrypoint = ''
    ${pkgs.coreutils}/bin/mkdir -p /data/upload

    APP_NAME="''${FLY_APP_NAME:-sober-strix}"
    PUBLIC_URL="https://$APP_NAME.fly.dev"

    echo "Configuring rustypaste with URL: $PUBLIC_URL"
    ${pkgs.coreutils}/bin/mkdir -p /etc/rustypaste
    ${pkgs.gnused}/bin/sed "s|@URL@|$PUBLIC_URL|g" ${rustypasteConfigTemplate} > /etc/rustypaste/config.toml

    export CONFIG="/etc/rustypaste/config.toml"
    exec ${pkgs.rustypaste}/bin/rustypaste
  '';
}
