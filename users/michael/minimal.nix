{ pkgs, config, ... }:

{
  imports = [
    ../../modules/home/core/ssh.nix
    ../../modules/home/core/cli.nix
    ../../modules/home/core/shell.nix
    ../../modules/home/core/nvim/nvim.nix
    ../../modules/home/core/irc
    ../../modules/home/core/tmux.nix
  ];

  nixpkgs.config.allowUnfree = true;

  programs.git = {
    enable = true;

    # Everything moves into 'settings' now
    settings = {
      user = {
        name = "Michael S. Singh";
        email = "michael@sober.fyi"; # Put your actual email here
      };
      init = {
        defaultBranch = "main";
      };
      # Add any other extraConfig items here directly
      pull.rebase = true;
    };
  };

  home.sessionVariables = {
    FLAKE = "${config.home.homeDirectory}/git/sober-nix";
  };
  home.stateVersion = "25.11";
}
