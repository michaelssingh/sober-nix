{ pkgs, ... }:

{
  home.packages = with pkgs; [
    # Utilities
    jq
    yq-go
    htop
    bottom
    nh # Nix Helper

    pkgs.socat
    pkgs.senpai
    pkgs.harper
    ];
}
