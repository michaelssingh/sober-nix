{ config, lib, pkgs, ... }:

{
  home.packages = [ pkgs.soju ];

  xdg.configFile."soju/config".text = ''
    listen irc+insecure://0.0.0.0:6697
    listen unix+admin://${config.home.homeDirectory}/.local/share/soju/admin.sock
    db sqlite3 ${config.home.homeDirectory}/.local/share/soju/soju.db
  '';

  home.activation.provision-soju = lib.hm.dag.entryAfter ["writeBoundary"] ''
    # Create data directory
    mkdir -p ${config.home.homeDirectory}/.local/share/soju

    # Start soju temporarily if not running
    started_soju=false
    if ! ${pkgs.procps}/bin/pgrep -x soju > /dev/null; then
      ${pkgs.soju}/bin/soju -config ${config.home.homeDirectory}/.config/soju/config &
      SOJU_PID=$!
      started_soju=true
      # Wait for admin socket
      for i in {1..20}; do
        if [ -S ${config.home.homeDirectory}/.local/share/soju/admin.sock ]; then break; fi
        sleep 0.5
      done
    fi

    # Provision user and network
    ${pkgs.soju}/bin/sojuctl -config ${config.home.homeDirectory}/.config/soju/config user create -username init -password "pineapple" || true
    ${pkgs.soju}/bin/sojuctl -config ${config.home.homeDirectory}/.config/soju/config user run init network create -addr irc.libera.chat -name libera || true
    ${pkgs.soju}/bin/sojuctl -config ${config.home.homeDirectory}/.config/soju/config user run init certfp generate -network libera || true

    # Stop temporary soju if we started it
    if [ "$started_soju" = true ]; then
      kill $SOJU_PID
      wait $SOJU_PID || true
    fi
  '';
}
