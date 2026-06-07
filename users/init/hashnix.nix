{ config, pkgs, inputs, ... }:

{
  home.username = "init";
  home.homeDirectory = "/home/init";
  home.stateVersion = "24.11";

  # Set remote flag for conditional configs (e.g. tmux bar at top)
  sober.isRemote = true;

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

  # Shell Configuration
  programs.bash = {
    enable = true;
    shellAliases = {
      hms = "home-manager switch --flake github:michaelssingh/sober-nix#init@hashnix --extra-experimental-features 'nix-command flakes' --refresh";
    };
  };

  programs.fish = {
    enable = true;
    shellAliases = {
      hms = "home-manager switch --flake github:michaelssingh/sober-nix#init@hashnix --extra-experimental-features 'nix-command flakes' --refresh";
    };
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
