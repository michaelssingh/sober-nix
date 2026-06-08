{ pkgs, ... }:

{
  home.packages = [ pkgs.mosh ];

  home.sessionVariables = {
    # Disable predictive echo (the annoying underlines) globally
    MOSH_PREDICTION_DISPLAY = "never";
    # Allow Mosh to set the terminal title
    MOSH_TITLE_TERM = "1";
  };

  # Optional: Alias for quick mosh + tmux attachment
  # Usage: mt <host>
  programs.fish.functions = {
    mt = ''
      mosh $argv -- tmux new-session -A -s main
    '';
  };
}
