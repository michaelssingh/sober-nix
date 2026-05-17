{ pkgs, config, ... }:

{
  imports = [
    ../../modules/home/core/ssh.nix
    ../../modules/home/core/cli.nix
    ../../modules/home/core/shell.nix
    ../../modules/home/core/nvim/nvim.nix
    ../../modules/home/features/youtube.nix
    ../../modules/home/features/blogs.nix
    ../../modules/home/features/mpv
  ];

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

  home.username = "michael";
  home.homeDirectory = "/home/michael";
  home.sessionVariables = {
    FLAKE = "${config.home.homeDirectory}/git/sober-nix";
  };
  home.stateVersion = "25.11";
}
