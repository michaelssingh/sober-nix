{
  pkgs,
  inputs,
  config,
  lib,
  ...
}:

let
  emailPersonal = "michaelssingh@protonmail.com";
  # Use the dynamic system theme instead of static import
  colors = config.sober.theme.current.colors;
in
{
  imports = [
    inputs.sops-nix.homeManagerModules.sops
    ./core.nix
    ../../modules/home/core/sober.nix
    ../../modules/home/core/mosh.nix
    ../../modules/home/core/irc

    ../../modules/home/features/workstation.nix
    ../../modules/home/desktop/qutebrowser
    ../../modules/home/desktop/firefox/firefox.nix
    ../../modules/home/core/email.nix
    ../../modules/home/core/aerc.nix
    ../../modules/home/core/neomutt.nix
  ];

  programs.rbw = {
    enable = true;
    settings = {
      email = emailPersonal;
      base_url = "https://vault.bitwarden.com";
      identity_url = "https://identity.bitwarden.com";
      pinentry = pkgs.pinentry-tty;
    };
  };

  xdg.userDirs = {
    enable = true;
    createDirectories = true;
    setSessionVariables = true;
    extraConfig = {
      SCREENSHOTS = "${config.home.homeDirectory}/pictures/screenshots";
    };
  };

  # Sops-Nix Key Source for Home-Manager
  sops.age.keyFile = "/home/michael/.config/sops/age/keys.txt";
  sops.defaultSopsFile = ../../secrets/secrets.yaml;

  sops.secrets."nixbuild.key" = {
    path = "/home/michael/.ssh/nixbuild";
    mode = "0600";
  };
  sops.secrets."fly.key" = {
    path = "/home/michael/.ssh/fly";
    mode = "0600";
  };
  sops.secrets."github.key" = {
    path = "/home/michael/.ssh/github";
    mode = "0600";
  };
  sops.secrets."glaucidium.key" = {
    path = "/home/michael/.ssh/glaucidium";
    mode = "0600";
  };
  sops.secrets."oci_api_key" = {
    path = "/home/michael/.oci/oci_api_key.pem";
    mode = "0600";
  };

  sober.theme.active = "tokyonight-storm";

  home.packages = with pkgs; [ cachix ];

  # Provision Cachix Auth Token from SOPS secret
  home.activation.provisionCachixToken = lib.hm.dag.entryAfter [ "writeBoundary" ] ''
        if ${pkgs.sops}/bin/sops -d --extract '["cachix_auth_token"]' ${../../secrets/secrets.yaml} > /tmp/cachix_token 2>/dev/null; then
          token=$(cat /tmp/cachix_token)
          mkdir -p ${config.home.homeDirectory}/.config/cachix
          cat <<EOF > ${config.home.homeDirectory}/.config/cachix/cachix.dhall
    { authToken =
        "$token"
    , hostname = "https://cachix.org"
    , binaryCaches = [] : List { name : Text, secretKey : Text }
    }
    EOF
          chmod 600 ${config.home.homeDirectory}/.config/cachix/cachix.dhall
          rm -f /tmp/cachix_token
        fi
  '';

  programs.zathura = {
    enable = true;
    package = pkgs.zathura.override {
      plugins = [
        pkgs.zathuraPkgs.zathura_djvu
        pkgs.zathuraPkgs.zathura_pdf_mupdf
      ];
    };
    options = {
      # Dynamic Palette from System Theme
      default-bg = colors.bg;
      default-fg = colors.fg;
      statusbar-bg = colors.black;
      statusbar-fg = colors.white;
      inputbar-bg = colors.bg;
      inputbar-fg = colors.yellow;
      notification-bg = colors.bg;
      notification-fg = colors.fg;
      notification-error-bg = colors.red;
      notification-error-fg = colors.fg;
      notification-warning-bg = colors.yellow;
      notification-warning-fg = colors.bg;
      completion-bg = colors.black;
      completion-fg = colors.white;
      completion-highlight-bg = colors.comment;
      completion-highlight-fg = colors.fg;
      recolor-lightcolor = colors.bg;
      recolor-darkcolor = colors.fg;
      recolor = true; # Enable dark mode rendering by default
    };
  };
}
