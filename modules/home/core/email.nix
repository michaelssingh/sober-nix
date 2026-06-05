{ pkgs, config, ... }:
{
  sops.secrets.protonmail_bridge_password = { };

  programs.mbsync.enable = true;
  programs.notmuch.enable = true;

  accounts.email.accounts.protonmail = {
    primary = true;
    address = "michaelssingh@protonmail.com";
    userName = "michaelssingh@protonmail.com";
    realName = "Michael S. Singh";
    passwordCommand = "${pkgs.coreutils}/bin/cat ${config.sops.secrets.protonmail_bridge_password.path}";

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

    mbsync = {
      enable = true;
      create = "both";
      patterns = [
        "*"
        "!\"~\""
      ];
      extraConfig.account = {
        Timeout = "300";
      };
    };
    notmuch.enable = true;
    neomutt = {
      enable = true;
      extraMailboxes = [
        "All Mail"
        "Archive"
        "Drafts"
        "Sent"
        "Spam"
        "Starred"
        "Trash"
      ];
    };
    aerc = {
      enable = true;
      # Per-account settings are handled by Home Manager
      extraConfig = { };
    };
  };

  systemd.user.services.mbsync = {
    Unit = {
      Description = "mbsync synchronization";
    };
    Service = {
      ExecStart = "${pkgs.isync}/bin/mbsync -a";
    };
  };

  systemd.user.timers.mbsync = {
    Unit = {
      Description = "mbsync synchronization timer";
    };
    Timer = {
      OnCalendar = "*:0/5";
      Persistent = true;
    };
    Install = {
      WantedBy = [ "timers.target" ];
    };
  };

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
