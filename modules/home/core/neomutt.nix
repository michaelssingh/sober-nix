{ pkgs, ... }:
{
  programs.neomutt = {
    enable = true;
    vimKeys = true;
    extraConfig = ''
      set from = "michaelssingh@protonmail.com"
      set realname = "Michael S. Singh"
      set smtp_url = "smtp://michaelssingh%40protonmail.com:794P%2BznPzfkl3fu4N1OleY1ojcziCH88or0tgz46v6w=@127.0.0.1:1025"
      set imap_user = "michaelssingh@protonmail.com"
      set folder = "imap://michaelssingh%40protonmail.com:794P%2BznPzfkl3fu4N1OleY1ojcziCH88or0tgz46v6w=@127.0.0.1:1143"
      set spoolfile = "+INBOX"
    '';
  };
}
