{ config, pkgs, inputs, ... }:

{
  home.username = "init";
  home.homeDirectory = "/home/init";
  home.stateVersion = "24.11";

  # Let Home Manager install and manage itself.
  programs.home-manager.enable = true;

  # Enable Nix Flakes permanently via Home Manager
  nix.package = pkgs.nix;
  nix.settings.experimental-features = [ "nix-command" "flakes" ];

  # Import your existing core modules
  imports = [
    ../../modules/home/core/nvim/nvim.nix
    ../../modules/home/core/tmux.nix
  ];

  # Additional packages for the pubnix environment
  home.packages = with pkgs; [
    htop
    git
  ];

  # Fish Shell Configuration
  programs.fish = {
    enable = true;
    interactiveShellInit = ''
      set -g fish_greeting
    '';
    shellInit = ''
      # Source Nix environment in chroot
      if test -e ~/.nix-profile/etc/profile.d/nix.fish
        source ~/.nix-profile/etc/profile.d/nix.fish
      end
      fish_add_path -g ~/.nix-profile/bin
    '';
  };

  programs.starship = {
    enable = true;
    enableFishIntegration = true;
  };
}
