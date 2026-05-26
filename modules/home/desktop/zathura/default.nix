{ pkgs, ... }:
{
  programs.zathura = {
    enable = true;
    options = {
      highlight-color = "rgba(255, 255, 0, 0.4)";
      highlight-active-color = "rgba(255, 165, 0, 0.4)";
    };
  };
}
