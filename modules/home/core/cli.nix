{ pkgs, ... }:

{
  home.packages = with pkgs; [
    # General Utilities
    pkgs.oci-cli
    pkgs.flyctl
    pkgs.rbw
    pkgs.dict
    pkgs.hydroxide
    pkgs.gemini-cli
    jq
    yq-go
    htop
    bottom
    nh # Nix Helper
    pkgs.socat
    pkgs.yazi
    pkgs.fastfetch
    pkgs.ripgrep-all

    # Typing practice tools
    pkgs.typioca
    pkgs.gtypist
    pkgs.ttyper
    pkgs.tt
  ];

  home.file.".dictrc".text = ''
    server dict.org
  '';
}
