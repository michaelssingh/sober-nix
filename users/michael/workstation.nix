{ pkgs, ... }:

{
  imports = [
    ./core.nix

    # Workstation only modules
    ../../modules/home/theme.nix
    ../../modules/home/sway/sway.nix
    ../../modules/home/firefox/firefox.nix
    ../../modules/home/email.nix
  ];

  # GUI-Only Packages
  home.packages = with pkgs; [
    zoom-us
    google-chrome
  ];
}
