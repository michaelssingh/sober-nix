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
    nix # Ensure nix is in the profile path
  ];

  # Shell Configuration (Codified in HM, but synced as real files for host compatibility)
  programs.bash = {
    enable = true;
    # These are our modern settings
    shellAliases = {
      hms = "home-manager switch --flake github:michaelssingh/sober-nix#init@hashnix --extra-experimental-features 'nix-command flakes' --refresh";
      hack = "~/nix-user-chroot ~/.nix fish -l";
      chat = "tmux -L host -f ~/.tmux-host.conf new-session -A -s host -n dream 'dream'";
    };
    bashrcExtra = ''
      export CLICOLOR=1
      export LSCOLORS=GxFxCxDxBxegedabagaced
      alias ls='ls --color=auto'
      alias ll='ls -lh'
      alias la='ls -A'
      alias grep='grep --color=auto'
      HISTSIZE=5000
      HISTFILESIZE=10000
      HISTCONTROL=ignoreboth:erasedups
      shopt -s histappend
      PS1='\[\033[01;32m\]\u@\h\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\]\$ '

      if [ -f /etc/bash_completion ]; then
        . /etc/bash_completion
      fi
    '';
    profileExtra = ''
      if [[ $- == *i* ]] && [ -z "$TMUX" ]; then
        tmux -L host -f ~/.tmux-host.conf has-session -t host 2>/dev/null || tmux -L host -f ~/.tmux-host.conf new-session -d -s host -n dream 'dream'
      fi
    '';
  };

  # CRITICAL: This block copies the Bash config from the Nix store to real files.
  # This prevents the "broken symlink" issue on the host shell while keeping
  # the configuration codified in your Flake.
  home.activation.syncBashConfig = config.lib.hm.dag.entryAfter [ "writeBoundary" ] ''
    $DRY_RUN_CMD rm -f $HOME/.bashrc $HOME/.bash_profile
    $DRY_RUN_CMD cp ${config.home.file.".bashrc".source} $HOME/.bashrc
    $DRY_RUN_CMD cp ${config.home.file.".bash_profile".source} $HOME/.bash_profile
    $DRY_RUN_CMD chmod 644 $HOME/.bashrc $HOME/.bash_profile
  '';

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
