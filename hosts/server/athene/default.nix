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
in
soberLib.mkContainerImage {
  name = "sober-athene";
  packages = [ pkgs.soju ];
  harden = true; # Production-hardened with minimal binaries
  exposedPorts = {
    "6697/tcp" = {};
    "8080/tcp" = {};
  };
  entrypoint = ''
    # Create the sqlite and uploads directories on the persistent Fly.io volume
    ${pkgs.coreutils}/bin/mkdir -p /var/lib/soju/uploads
    
    # Symlink config to default path so that sojuctl/sojudb default configurations work
    ${pkgs.coreutils}/bin/mkdir -p /etc/soju
    ${pkgs.coreutils}/bin/ln -sf ${sojuConfig} /etc/soju/config

    # Launch Soju pointing to the read-only Nix store configuration
    exec ${pkgs.soju}/bin/soju -config ${sojuConfig}
  '';
}
