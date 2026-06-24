{
  pkgs,
  inputs,
  ...
}:

{
  imports = [
    inputs.sops-nix.homeManagerModules.sops
    ../michael/minimal.nix
    ../../modules/home/core/sober.nix
  ];

  # Allow unfree packages (needed for the sprite CLI)
  nixpkgs.config.allowUnfree = true;

  # Global Sober System Options
  sober.isRemote = true;

  home.username = "sprite";
  home.homeDirectory = "/home/sprite";
  home.stateVersion = "25.11";

  programs.home-manager.enable = true;

  # Sops-Nix Key Source for Home-Manager
  sops.age.keyFile = "/home/sprite/.config/sops/age/keys.txt";
  sops.defaultSopsFile = ../../secrets/secrets.yaml;

  home.packages = with pkgs; [
    git
    htop
    ripgrep
    fd
    jq
  ];
}
