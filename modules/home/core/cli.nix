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
    pkgs.harper
    pkgs.imv
    pkgs.yazi
    pkgs.typioca

    # Media
   pkgs.castero
    pkgs.kjv

  ];

  home.file.".dictrc".text = ''
    server dict.org
  '';

  # Import Castero podcasts
  home.activation.import-castero = lib.hm.dag.entryAfter [ "writeBoundary" ] ''
    if [ -x "${pkgs.castero}/bin/castero" ]; then
      ${pkgs.castero}/bin/castero --import ${./podcasts.opml}
    fi
  '';
}
