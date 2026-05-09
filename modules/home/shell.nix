{ pkgs, ... }:

{
  # --- 1. The Shell (Fish) ---
  programs.fish = {
    enable = true;

    interactiveShellInit = ''
      set fish_greeting # Disable greeting
      # --- 1. Tokyonight Fish Color Palette ---
      set -g fish_color_normal "#c0caf5"
      set -g fish_color_command "#7dcfff"
      set -g fish_color_keyword "#bb9af7"
      set -g fish_color_quote "#e0af68"
      set -g fish_color_redirection "#c0caf5"
      set -g fish_color_end "#ff9e64"
      set -g fish_color_error "#f7768e"
      set -g fish_color_param "#9d7cd8"
      set -g fish_color_comment "#565f89"
      set -g fish_color_selection --background="#283457"
      set -g fish_color_search_match --background="#283457"
      set -g fish_color_operator "#9ece6a"
      set -g fish_color_escape "#bb9af7"
      set -g fish_color_autosuggestion "#565f89"

      # --- 2. Completion Pager Colors
      set -g fish_pager_color_progress "#565f89"
      set -g fish_pager_color_prefix "#7dcfff"
      set -g fish_pager_color_completion "#c0caf5"
      set -g fish_pager_color_description "#565f89"

      # --- 3. Completion Behavior Settings ---
      # Tab through completions instead of just listing them
      set -g fish_complete_path $fish_complete_path /etc/fish/completions
    '';
    functions = {
      bw-ssh-init = ''
        set -l item_name (or $argv[1] "GitHub")
        export BW_SESSION=$(bw unlock --raw)
        bw get item "$item_name" --raw | jq -r '.notes' | ssh-add -
      '';
    };
    shellAliases = {
      # Replace standard ls with eza
      ls = "eza";
      ll = "eza -l --icons --git";
      la = "eza -la --icons --git";
      lt = "eza --tree --icons"; # Tree view!

      # Replace cat with bat
      cat = "bat";

      # Edit current directory
      e = "nvim .";

      # System switch
      os = "nh os switch .";
    };
  };

  # --- 2. The Prompt (Starship) ---
  programs.starship = {
    enable = true;
    enableFishIntegration = true;
    settings = {
      add_newline = false;
      format = "$directory$character";
      right_format = "$all";
      palette = "tokyonight";

      palettes.tokyonight = {
        blue = "#7aa2f7";
        purple = "#bb9af7";
        green = "#9ece6a";
        red = "#f7768e";
        yellow = "#e0af68";
        bg = "#1a1b26";
      };

      directory = {
        style = "blue";
        read_only = " 🔒";
      };

      # The "Owl" Identity for Nix Shells ❄️
      nix_shell = {
        symbol = "❄️ ";
        style = "purple bold";
      };
    };
  };

  # --- 3. Navigation Tools ---
  programs.bat = {
    enable = true;
    config = {
      theme = "base16"; # Matches your aesthetic
    };
  };

  programs.eza = {
    enable = true;
    git = true;
    icons = "auto";
    extraOptions = [
      "--group-directories-first"
      "--header"
    ];
  };

  # Zoxide (Smarter 'cd')
  programs.zoxide = {
    enable = true;
    enableFishIntegration = true;
  };

  # FZF (Fuzzy Finder)
  programs.fzf = {
    enable = true;
    enableFishIntegration = true;
    defaultCommand = "fd --type f"; # Faster than 'find'
  };

  # Direnv (Auto-loading dev environments)
  programs.direnv = {
    enable = true;
    nix-direnv.enable = true;
  };

  # --- 4. Essential CLI Tools ---
  home.packages = with pkgs; [
    bat
    ripgrep # Fast grep
    fd # Fast find
    jq # JSON processor
    btop # System monitor
    nixfmt-rfc-style # Nix formatter
    fastfetch # System info (optional, fun)
    bitwarden-cli # Bitwarden CLI
  ];
}
