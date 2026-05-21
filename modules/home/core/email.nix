{ pkgs, ... }:
{
  accounts.email.accounts.protonmail = {
    primary = true;
    address = "michaelssingh@protonmail.com";
    userName = "michaelssingh@protonmail.com";
    realName = "Michael S. Singh";
    passwordCommand = "echo '794P+znPzfkl3fu4N1OleY1ojcziCH88or0tgz46v6w='";
    
    imap = {
      host = "127.0.0.1";
      port = 1143;
      tls.enable = false;
    };
    smtp = {
      host = "127.0.0.1";
      port = 1025;
      tls.enable = false;
    };
    
    aerc.enable = true;
    neomutt.enable = true;
    notmuch.enable = true;
  };

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
