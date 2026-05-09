{ pkgs, ... }:

{
  # --- Himalaya Email Configuration ---
  # Configure Himalaya to use the local Hydroxide bridge
  xdg.configFile."himalaya/config.toml".text = ''
    [accounts.protonmail]
    default = true
    email = "michaelssingh@protonmail.com"
    display-name = "Michael S. Singh"
    downloads-dir = "${builtins.getEnv "HOME"}/Downloads"

    # Backend settings using the local bridge (Hydroxide)
    backend = "imap"
    imap-host = "127.0.0.1"
    imap-port = 1143
    imap-encryption = "none" # Hydroxide bridges local, encryption handled at tunnel

    sender = "smtp"
    smtp-host = "127.0.0.1"
    smtp-port = 1025
    smtp-encryption = "none"
  '';

  # Ensure Hydroxide starts as a service
  systemd.user.services.hydroxide = {
    Unit = {
      Description = "Hydroxide Proton Mail Bridge";
      After = [ "network.target" ];
    };

    Service = {
      ExecStart = "${pkgs.hydroxide}/bin/hydroxide serve";
      Restart = "always";
      RestartSec = 10;
    };

    Install = {
      WantedBy = [ "default.target" ];
    };
  };
}
