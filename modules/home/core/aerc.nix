{ pkgs, ... }:
{
  programs.aerc = {
    enable = true;
    extraConfig = {
      general = {
        default-save-path = "~/Downloads";
        unsafe-accounts-conf = true;
      };
      filters = {
        "text/html" = "chawan";
      };
    };
  };
}
