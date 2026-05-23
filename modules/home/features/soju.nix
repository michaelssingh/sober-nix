{ config, lib, pkgs, ... }:

let
  isFly = config.home.username == "root";
  dataPath = if isFly then "/var/lib/nix_persist/soju" else "${config.home.homeDirectory}/.local/share/soju";
in
{
  home.packages = [ pkgs.soju ];

  xdg.configFile."soju/config".text = ''
    listen irc+insecure://0.0.0.0:6697
    listen unix+admin://${dataPath}/admin.sock
    db sqlite3 ${dataPath}/soju.db
  '';

  home.activation.provision-soju-dir = lib.hm.dag.entryAfter ["writeBoundary"] ''
    mkdir -p "${dataPath}"
  '';
}
