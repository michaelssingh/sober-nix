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
        "text/html" = "${pkgs.w3m}/bin/w3m -T text/html -dump";
      };
    };
  };
}
