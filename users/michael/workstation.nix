{ pkgs, inputs, config, ... }:

let
  emailPersonal = "michaelssingh@protonmail.com";
  emailWork = "michael@sober.fyi";
  # Use the dynamic system theme instead of static import
  colors = config.sober.theme.current.colors;
in
{
  imports = [
    inputs.sops-nix.homeManagerModules.sops
    ./core.nix
    ../../modules/home/core/sober.nix
    ../../modules/home/core/mosh.nix

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
  sops.secrets."hashnix.key" = {
    path = "/home/michael/.ssh/hashnix";
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
    (inputs.nixpkgs-pinned.legacyPackages.${pkgs.stdenv.hostPlatform.system}.transmission_4)
    stig
    chawan
    dict
    rbw
    qutebrowser
    antigravity
    strix-paste
    nheko
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
