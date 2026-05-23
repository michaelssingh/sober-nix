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
        "text/html" = "w3m -dump -T text/html";
      };
    };
  };
}
