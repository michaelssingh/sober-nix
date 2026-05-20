{ config, lib, pkgs, ... }:

let
  # On Fly.io, we use the persistent volume. On local machine, we use the standard XDG path.
  # This detection is based on the existence of the Fly mount point.
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

  home.activation.provision-soju = lib.hm.dag.entryAfter ["writeBoundary"] ''
    # Create persistent data directory
    mkdir -p ${dataPath}

    # Start soju temporarily if not running
    started_soju=false
    if ! ${pkgs.procps}/bin/pgrep -x soju > /dev/null; then
      ${pkgs.soju}/bin/soju -config ${config.home.homeDirectory}/.config/soju/config &
      SOJU_PID=$!
      started_soju=true
      # Wait for admin socket
      for i in {1..20}; do
        if [ -S ${dataPath}/admin.sock ]; then break; fi
        sleep 0.5
      done
    fi

    # 1. Provision user
    ${pkgs.soju}/bin/sojuctl -config ${config.home.homeDirectory}/.config/soju/config user create -username init -password "pineapple" || true

    # 2. Provision network (Basic setup)
    ${pkgs.soju}/bin/sojuctl -config ${config.home.homeDirectory}/.config/soju/config user run init network create -addr irc.libera.chat -name libera || true

    # 3. Configure Authentication on Libera
    # Use SASL PLAIN with provided credentials
    ${pkgs.soju}/bin/sojuctl -config ${config.home.homeDirectory}/.config/soju/config user run init network update -name libera -user "init" -pass "dT4d8y3Tz*kavNrmue4YzDsX3^VdU%9UA%8U" -sasl plain || true
    
    # Still generate CertFP for future use if desired
    ${pkgs.soju}/bin/sojuctl -config ${config.home.homeDirectory}/.config/soju/config user run init certfp generate -network libera || true

    # Stop temporary soju if we started it
    if [ "$started_soju" = true ]; then
      kill $SOJU_PID
      wait $SOJU_PID || true
    fi
  '';
}
