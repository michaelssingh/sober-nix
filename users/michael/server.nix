{ config, pkgs, inputs, lib, ... }:

{
  imports = [
    inputs.sops-nix.homeManagerModules.sops
    ./core.nix
    ../../modules/home/core/sober.nix
  ];

  # Global Sober System Options
  sober.isRemote = true;

  home.username = "michael";
  home.homeDirectory = "/home/michael";
  home.stateVersion = "23.11";

  programs.home-manager.enable = true;

  # Sops-Nix Key Source for Home-Manager
  sops.age.keyFile = "/home/michael/.config/sops/age/keys.txt";
  sops.defaultSopsFile = ../../secrets/secrets.yaml;

  home.packages = with pkgs; [
    git
    htop
    ripgrep
    fd
    jq
  ];
}
