{
  pkgs,
  ...
}:
{
  # Ensure the service starts automatically
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
