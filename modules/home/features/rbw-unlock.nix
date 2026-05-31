{ config, lib, pkgs, ... }:

{
  # Ensure sops-nix is available
  sops.secrets.bw_master_password = {
    path = "%h/.config/rbw/master_password";
  };

  systemd.user.services.rbw-unlock = {
    Unit = {
      Description = "Unlock rbw on login";
      After = [ "graphical-session.target" ];
    };

    Service = {
      Type = "oneshot";
      ExecStart = "${pkgs.bash}/bin/bash -c '${pkgs.rbw}/bin/rbw unlock < ${config.home.homeDirectory}/.config/rbw/master_password'";
      Restart = "on-failure";
    };

    Install = {
      WantedBy = [ "graphical-session.target" ];
    };
  };
}
