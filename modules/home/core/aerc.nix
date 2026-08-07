{ pkgs, ... }:
{
  programs.aerc = {
    enable = true;
    extraConfig = {
      general = {
        default-save-path = "~/Downloads";
        unsafe-accounts-conf = true;
      };
      ui = {
        sidebar-width = "20";
        sidebar-visible = "true";
        threading-enabled = "true";
        index-format = "%D | %-15.15n | %s";
      };
      compose = {
        editor = "nvim";
        header-layout = "To,Cc,Subject";
        address-book-cmd = "${pkgs.writeShellScript "aerc-address-book" ''
                    query="$1"
                    if [ -z "$query" ]; then
                      search="*"
                    else
                      search="from:*$query* or to:*$query*"
                    fi
                    ${pkgs.notmuch}/bin/notmuch address --deduplicate=address "$search" 2>/dev/null | ${pkgs.python3}/bin/python3 -c '
          import sys, re
          for line in sys.stdin:
              line = line.strip()
              m = re.search(r"^(.*?)\s*<([^>]+)>$", line)
              if m:
                  name = m.group(1).strip("\" \t")
                  email = m.group(2).strip()
                  print(email + "\t" + name)
              elif "@" in line:
                  print(line + "\t")
          '
        ''} %s";
      };
      filters = {
        "text/plain" = "wrap -w 100 | colorize";
        "text/html" =
          "${pkgs.bubblewrap}/bin/bwrap --unshare-net --dev-bind / / ${pkgs.w3m}/bin/w3m -T text/html -O UTF-8 -o display_link_number=1 -dump | colorize";
      };
      triggers = {
        new-email = "exec notify-send \"New email from %n\" \"%s\"";
      };
    };
  };
}
