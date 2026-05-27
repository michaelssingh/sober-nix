{ pkgs, lib, ... }:

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
    pkgs.senpai-dev
    pkgs.harper
    pkgs.imv
    pkgs.yazi
    
    # Media
    pkgs.castero
    pkgs.kjv
  ];

  # Import Castero podcasts
  home.activation.import-castero = lib.hm.dag.entryAfter ["writeBoundary"] ''
    if [ -x "${pkgs.castero}/bin/castero" ]; then
      ${pkgs.castero}/bin/castero --import ${./podcasts.opml}
    fi
  '';
}
