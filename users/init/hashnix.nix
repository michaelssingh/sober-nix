{ config, pkgs, inputs, lib, ... }:

let
  # Define the exact, clean text for the host bash files.
  # This avoids any Nix-generated boilerplate that would break on the host shell.
  bashrcText = ''
    # --- Modern Bash Config (Declarative) ---
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

    # Aliases
    alias hms="home-manager switch --flake github:michaelssingh/sober-nix#init@hashnix --extra-experimental-features 'nix-command flakes' --refresh"
    alias hack="~/nix-user-chroot ~/.nix fish -l"
    alias chat="tmux -L host -f ~/.tmux-host.conf new-session -A -s host -n dream 'dream'"

    if [ -f /etc/bash_completion ]; then . /etc/bash_completion; fi
  '';

  bashProfileText = ''
    # --- Modern Bash Profile (Declarative) ---
    if [ -f ~/.bashrc ]; then . ~/.bashrc; fi

    # Start host chat session in background on login
    if [[ $- == *i* ]] && [ -z "$TMUX" ]; then
      tmux -L host -f ~/.tmux-host.conf has-session -t host 2>/dev/null || tmux -L host -f ~/.tmux-host.conf new-session -d -s host -n dream 'dream'
    fi
  '';

  dreamrcText = ''
    # --- init's Dream Config (Declarative) ---
    DREAM_PRINT_TIMESTAMPS=true
    DREAM_PRINT_COLORS=true
    DREAM_TIMESTAMP_COLOR="cyan"
    DREAM_BRACKET_COLOR="blue"
    DREAM_BRACE_COLOR="magenta"
    DREAM_ANGLEBRACKET_COLOR="yellow"
    DREAM_TIMESTAMP_FORMAT="%H:%M"
    DREAM_ALERT_ON_USERNAME=true
  '';
in
{
  home.username = "init";
  home.homeDirectory = "/home/init";
  home.stateVersion = "24.11";

  # Global Sober System Options
  sober.isRemote = true;

  home.sessionVariables = {
    TZ = config.sober.timezone;
  };

  # Let Home Manager install and manage itself.
  programs.home-manager.enable = true;

  # Enable Nix Flakes permanently via Home Manager
  nix.package = pkgs.nix;
  nix.settings.experimental-features = [ "nix-command" "flakes" ];

  # Import core modules
  imports = [
    ../../modules/home/core/sober.nix
    ../../modules/home/core/nvim/nvim.nix
    ../../modules/home/core/tmux.nix
    ../../modules/home/core/shell.nix
  ];

  # Additional packages for the pubnix environment
  home.packages = with pkgs; [
    htop
    git
    nix # Ensure nix is in the profile path
  ];

  # 1. Disable HM's bash management to remove all Nix boilerplate from host files
  programs.bash.enable = false;

  # 2. Define clean "source" files in the store
  home.file.".bashrc_source".text = bashrcText;
  home.file.".bash_profile_source".text = bashProfileText;
  home.file.".dreamrc_source".text = dreamrcText;

  # 3. Activation script to sync these as real files to the host
  home.activation.syncHostConfig = lib.hm.dag.entryAfter [ "writeBoundary" ] ''
    $DRY_RUN_CMD rm -f $HOME/.bashrc $HOME/.bash_profile $HOME/.dreamrc
    $DRY_RUN_CMD cp ${config.home.file.".bashrc_source".source} $HOME/.bashrc
    $DRY_RUN_CMD cp ${config.home.file.".bash_profile_source".source} $HOME/.bash_profile
    $DRY_RUN_CMD cp ${config.home.file.".dreamrc_source".source} $HOME/.dreamrc
    $DRY_RUN_CMD chmod 644 $HOME/.bashrc $HOME/.bash_profile $HOME/.dreamrc
  '';
}
