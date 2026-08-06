{ pkgs, config, ... }:

{
  imports = [
    ../../modules/home/core/ssh.nix
    ../../modules/home/core/cli.nix
    ../../modules/home/core/shell.nix
    ../../modules/home/core/nvim/nvim.nix
    ../../modules/home/core/tmux.nix
  ];

  home.packages = with pkgs; [
    procps
  ];

  programs.git = {
    enable = true;
    lfs.enable = true;

    signing = {
      key = "${config.home.homeDirectory}/.ssh/github";
      signByDefault = true;
    };

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
      gpg = {
        format = "ssh";
      };
    };
  };

  home.sessionVariables = {
    FLAKE = "${config.home.homeDirectory}/git/sober-nix";
  };
  home.stateVersion = "25.11";
}
