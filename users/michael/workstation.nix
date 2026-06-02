{ pkgs, inputs, ... }:

let
  emailPersonal = "michaelssingh@protonmail.com";
  emailWork = "michael@sober.fyi";
  theme = import ../../modules/home/core/theme.nix;
in
{
  imports = [
    inputs.sops-nix.homeManagerModules.sops
    ./core.nix

    # Workstation only modules
    ../../modules/home/desktop/sway
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
      # pinentry = pkgs.writeShellScriptBin "pinentry-bemenu-tokyonight" ''
      #   ${pkgs.pinentry-bemenu}/bin/pinentry-bemenu \
      #     --nb="${theme.colors.background}" \
      #     --nf="${theme.colors.foreground}" \
      #     --hb="${theme.colors.blue}" \
      #     --hf="${theme.colors.foreground}" \
      #     --sb="${theme.colors.blue}" \
      #     --sf="${theme.colors.foreground}" \
      #     --fb="${theme.colors.background}" \
      #     --ff="${theme.colors.foreground}" \
      #     --tb="${theme.colors.background}" \
      #     --tf="${theme.colors.blue}" \
      #     "$@"
      # '';
    };
  };

  # Sops-Nix Key Source for Home-Manager
  sops.age.keyFile = "/home/michael/.config/sops/age/keys.txt";
  sops.defaultSopsFile = ../../secrets/secrets.yaml;

  sops.secrets."ssh/nixbuild" = {
    path = "/home/michael/.ssh/nixbuild";
    mode = "0600";
  };
  sops.secrets."ssh/fly" = {
    path = "/home/michael/.ssh/fly";
    mode = "0600";
  };
  sops.secrets."ssh/github" = {
    path = "/home/michael/.ssh/github";
    mode = "0600";
  };

  # GUI-Only Packages
  home.packages = with pkgs; [
    foot
    fuzzel
    aerc
    neomutt
    swaylock
    swayidle
    (inputs.nixpkgs-pinned.legacyPackages.${pkgs.system}.transmission_4)
    stig
    chawan
    dict
    rbw
    qutebrowser
  ];
  programs.zathura = {
    enable = true;
    package = pkgs.zathura.override {
      plugins = [
        pkgs.zathuraPkgs.zathura_djvu
        pkgs.zathuraPkgs.zathura_pdf_mupdf
      ];
    };
    options = {
      # Tokyonight Storm Palette
      default-bg = "#24283b";
      default-fg = "#c0caf5";
      statusbar-bg = "#1f2335";
      statusbar-fg = "#a9b1d6";
      inputbar-bg = "#24283b";
      inputbar-fg = "#ff9e64";
      notification-bg = "#24283b";
      notification-fg = "#c0caf5";
      notification-error-bg = "#f7768e";
      notification-error-fg = "#c0caf5";
      notification-warning-bg = "#e0af68";
      notification-warning-fg = "#24283b";
      highlight-color = "#e0af68";
      highlight-active-color = "#9ece6a";
      completion-bg = "#1f2335";
      completion-fg = "#a9b1d6";
      completion-highlight-bg = "#414868";
      completion-highlight-fg = "#c0caf5";
      recolor-lightcolor = "#24283b";
      recolor-darkcolor = "#c0caf5";
      recolor = true; # Enable dark mode rendering by default
    };
  };
}
