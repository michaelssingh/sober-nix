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
      os = ''
        nh os switch $argv /home/michael/git/sober-nix
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

      # Modern replacements
      ps = "procs";
      du = "dust";
      top = "btm";
      htop = "btm";
      dig = "doggo";
      sed = "sd";
      curl = "xh";
      http = "xh";

      # Edit current directory
      e = "nvim .";
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

  # Bottom (btm) - Modern, fast system monitor
  programs.bottom = {
    enable = true;
    settings = {
      flags = {
        avg_cpu = true;
        temperature_type = "c";
      };
      colors = {
        table_header_color = "#7dcfff";
        all_cpu_color = "#7dcfff";
        avg_cpu_color = "#bb9af7";
        cpu_core_colors = [
          "#7aa2f7"
          "#bb9af7"
          "#9ece6a"
          "#e0af68"
          "#ff9e64"
          "#f7768e"
        ];
        ram_color = "#9ece6a";
        swap_color = "#e0af68";
        rx_color = "#9ece6a";
        tx_color = "#f7768e";
        widget_title_color = "#bb9af7";
        border_color = "#565f89";
        selected_border_color = "#7aa2f7";
        text_color = "#c0caf5";
        graph_color = "#a9b1d6";
      };
    };
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

  # Custom configurations for tools without modules
  xdg.configFile."xh/config.yml".text = ''
    # xh configuration
    style: auto
    default-scheme: https
  '';

  xdg.configFile."procs/config.toml".text = ''
    # procs configuration
    theme = "dark"

    [[columns]]
    kind = "Pid"
    style = "BrightYellow"

    [[columns]]
    kind = "User"
    style = "BrightGreen"

    [[columns]]
    kind = "Command"
    style = "BrightBlue"
  '';

  # --- 4. Essential CLI Tools ---
  home.packages = with pkgs; [
    bat
    ripgrep # Fast grep
    fd # Fast find
    jq # JSON processor
    btop # System monitor (C++)
    bottom # System monitor (Rust)
    procs # ps replacement (Rust)
    doggo # DNS client (Go/Modern)
    dust # du replacement (Rust)
    ouch # Compression (Rust)
    sd # sed replacement (Rust)
    xh # httpie/curl replacement (Rust)
    hexyl # hex viewer (Rust)
    bandwhich # network monitor (Rust)
    grex # regex generator (Rust)
    nixfmt-rfc-style # Nix formatter
  ];
}
