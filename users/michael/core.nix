{ pkgs, ... }:

{
  imports = [
    # ../../modules/home/ssh.nix
    ../../modules/home/cli.nix
    ../../modules/home/features/youtube.nix
    ../../modules/home/shell.nix
    ../../modules/home/nvim/nvim.nix
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
  home.stateVersion = "25.11";
}
