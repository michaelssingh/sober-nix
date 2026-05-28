{ pkgs, ... }:
{
  services.mako = {
    enable = true;
    settings = {
      font = "Inter 10";
      background-color = "#24283b";
      text-color = "#c0caf5";
      border-color = "#7aa2f7";
      border-radius = 5;
      border-size = 2;
      padding = "10";
      default-timeout = 5000;
    };
  };
}
