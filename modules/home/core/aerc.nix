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
      };
      filters = {
        "text/plain" = "wrap -w 100 | colorize";
        "text/html" =
          "${pkgs.bubblewrap}/bin/bwrap --unshare-net --dev-bind / / --ro-bind-try /etc/resolv.conf /etc/resolv.conf ${pkgs.w3m}/bin/w3m -T text/html -O UTF-8 -o display_link_number=1 -dump | colorize";
      };
    };
  };
}
