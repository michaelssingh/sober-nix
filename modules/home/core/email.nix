{ pkgs, config, ... }:
{
  sops.secrets.protonmail_bridge_password = { };

  programs.mbsync.enable = true;
  programs.notmuch = {
    enable = true;
    new.tags = [
      "unread"
      "inbox"
      "new"
    ];
  };

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
      extraConfig.channel = {
        Sync = "All";
        Expunge = "Both";
      };
      extraConfig.account = {
        Timeout = "300";
        PipelineDepth = "1";
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
      extraAccounts = {
        source = "maildir://~/Maildir/protonmail";
      };
    };
  };

  systemd.user.services.mbsync = {
    Unit = {
      Description = "mbsync synchronization";
    };
    Service = {
      ExecStart = "${pkgs.writeShellScript "mbsync-and-notmuch" ''
                ${pkgs.isync}/bin/mbsync -a 2>/dev/null || true
                ${pkgs.notmuch}/bin/notmuch new 2>/dev/null || true
                ${pkgs.notmuch}/bin/notmuch search --format=json tag:new 2>/dev/null | ${pkgs.python3}/bin/python3 -c '
        import sys, json, subprocess
        try:
            data = json.load(sys.stdin)
            for msg in data:
                authors = msg.get("authors", "Unknown")
                subject = msg.get("subject", "(No Subject)")
                subprocess.run(["${pkgs.libnotify}/bin/notify-send", "-a", "aerc", "-i", "mail-unread", f"New Email from {authors}", subject])
        except Exception:
            pass
        '
                ${pkgs.notmuch}/bin/notmuch tag -new tag:new 2>/dev/null || true
      ''}";
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
