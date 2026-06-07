{ pkgs, config, ... }:

let
  colors = config.sober.theme.current.colors;
in
{
  # --- 1. The Shell (Fish) ---
  programs.fish = {
    enable = true;

    interactiveShellInit = ''
      ${if config.sober.isRemote then "fish_add_path ~/.nix-profile/bin" else ""}
      set fish_greeting # Disable greeting

      # Automatically add sops-decrypted SSH keys on login
      ssh-add ~/.ssh/nixbuild ~/.ssh/fly ~/.ssh/github ~/.ssh/hashnix 2>/dev/null

      ${if !config.sober.isRemote then ''
      function wg-up
        echo "Restarting WireGuard services in order..."
        sudo systemctl stop wg-quick-wg-sober
        sudo systemctl stop wg-quick-wg-fly
        sudo systemctl start wg-quick-wg-fly
        sleep 2
        sudo systemctl start wg-quick-wg-sober
        echo "WireGuard services restarted."
      end

      function wg-down
        echo "Restarting WireGuard services in order..."
        sudo systemctl stop wg-quick-wg-sober
        sudo systemctl stop wg-quick-wg-fly
        sudo systemctl start wg-quick-wg-fly
        sleep 2
        sudo systemctl start wg-quick-wg-sober
        echo "WireGuard services restarted."
      end
      '' else ""}

      # --- 1. Dynamic Fish Color Palette ---
      set -g fish_color_normal "${colors.fg}"
      set -g fish_color_command "${colors.cyan}"
      set -g fish_color_keyword "${colors.magenta}"
      set -g fish_color_quote "${colors.yellow}"
      set -g fish_color_redirection "${colors.fg}"
      set -g fish_color_end "#ff9e64"
      set -g fish_color_error "${colors.red}"
      set -g fish_color_param "#9d7cd8"
      set -g fish_color_comment "${colors.comment}"
      set -g fish_color_selection --background="#283457"
      set -g fish_color_search_match --background="#283457"
      set -g fish_color_operator "${colors.green}"
      set -g fish_color_escape "${colors.magenta}"
      set -g fish_color_autosuggestion "${colors.comment}"

      # --- 2. Completion Pager Colors
      set -g fish_pager_color_progress "${colors.comment}"
      set -g fish_pager_color_prefix "${colors.cyan}"
      set -g fish_pager_color_completion "${colors.fg}"
      set -g fish_pager_color_description "${colors.comment}"

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

      # AI Assistants
      gemini = "gemini-cli";
      chat = "gemini-cli";
      ai = "clemini";
    }
    // (
      if !config.sober.isRemote then
        {
          # Workstation-only aliases
          os = "nh os switch $argv /home/michael/git/sober-nix";
          mpv = "mpv-queue";
        }
      else
        {
          # Remote-only aliases
          hms = "PATH=$PATH:~/.nix-profile/bin home-manager switch --flake github:michaelssingh/sober-nix#init@hashnix --extra-experimental-features 'nix-command flakes' --refresh";
        }
    );
  };

  # --- 2. The Prompt (Starship) ---
  programs.starship = {
    enable = true;
    enableFishIntegration = true;
    settings = {
      add_newline = false;
      format = "$directory\n$character";
      right_format = "$all";
      palette = "tokyonight";

      palettes.tokyonight = {
        blue = colors.blue;
        purple = colors.magenta;
        green = colors.green;
        red = colors.red;
        yellow = colors.yellow;
        bg = colors.bg;
      };

      directory = {
        style = colors.blue;
        read_only = " 🔒";
      };

      character = {
        success_symbol = if config.sober.isRemote then "[Ω](cyan)" else "[λ](green)";
        error_symbol = if config.sober.isRemote then "[Ω](red)" else "[λ](red)";
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
        table_header_color = colors.cyan;
        all_cpu_color = colors.cyan;
        avg_cpu_color = colors.magenta;
        cpu_core_colors = [
          colors.blue
          colors.magenta
          colors.green
          colors.yellow
          "#ff9e64"
          colors.red
        ];
        ram_color = colors.green;
        swap_color = colors.yellow;
        rx_color = colors.green;
        tx_color = colors.red;
        widget_title_color = colors.magenta;
        border_color = colors.comment;
        selected_border_color = colors.accent;
        text_color = colors.fg;
        graph_color = colors.white;
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
    nixfmt # Nix formatter
  ];
}
