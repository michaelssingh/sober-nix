{ pkgs, config, ... }:
{
  programs.neomutt = {
    enable = true;
    vimKeys = true;
    extraConfig = ''
      set from = "michaelssingh@protonmail.com"
      set realname = "Michael S. Singh"
      set smtp_url = "smtp://michaelssingh%40protonmail.com@127.0.0.1:1025"
      set imap_user = "michaelssingh@protonmail.com"
      set folder = "imap://michaelssingh%40protonmail.com@127.0.0.1:1143"
      set spoolfile = "+INBOX"
      set smtp_pass = "`cat ${config.sops.secrets.protonmail_bridge_password.path}`"
      set imap_pass = "`cat ${config.sops.secrets.protonmail_bridge_password.path}`"

      # HTML Rendering
      auto_view text/html
      set mailcap_path = "~/.mailcap"

      # Navigation Fix
      # bind index <enter> display-message

      # Sane Defaults
      set sort = threads
      set sort_aux = last-date-received
      set sidebar_visible = yes
      set sidebar_width = 20
      set sidebar_short_path = yes
      set mail_check = 60
      set timeout = 30
      set editor = "nvim"

    '';
  };
}
