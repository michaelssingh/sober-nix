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
    pkgs.imv
    pkgs.yazi
    pkgs.typioca
    pkgs.gtypist
    pkgs.ttyper
    pkgs.tt

    # Media
    pkgs.castero
    pkgs.kjv

  ];

  home.file.".dictrc".text = ''
    server dict.org
  '';

  # Import Castero podcasts only when the OPML configuration changes
  home.file.".config/castero/podcasts.opml" = {
    source = ./podcasts.opml;
    onChange = ''
      if [ -x "${pkgs.castero}/bin/castero" ]; then
        ${pkgs.castero}/bin/castero --import "$HOME/.config/castero/podcasts.opml"
      fi
    '';
  };
}
