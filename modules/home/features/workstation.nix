{ pkgs, ... }:

{
  imports = [
    ./dev.nix
    ./media-cli.nix
  ];

  # Workstation-specific Shell Functions
  programs.fish.functions = {
    live = ''
      argparse a/audio -- $argv

      # Find a matching stream in the registry
      set -l match (grep -i "$argv" ~/git/sober-nix/modules/home/features/livestreams.txt)
      if test -n "$match"
        set -l url (string split "|" $match)[2]
        
        set -l flags
        if set -q _flag_audio
          set flags "--vid=no"
        end

        # Load the file with optional flags
        mpv-queue $flags "$url"
      else
        echo "Stream not found: $argv"
      end
    '';
  };

  # Workstation-only Packages
  home.packages = with pkgs; [
    gemini-cli
    flyctl
    hydroxide
    ripgrep-all
    fastfetch
  ];

  # Workstation-specific Terminal Multiplexer
  programs.zellij = {
    enable = true;
    settings = {
      theme = "tokyonight";
      default_mode = "locked";
      pane_frames = false;
      mouse_mode = true;
      copy_on_select = true;
      mirror_session = true;

      themes.tokyonight = {
        fg = "#c0caf5";
        bg = "#1a1b26";
        black = "#15161e";
        red = "#f7768e";
        green = "#9ece6a";
        yellow = "#e0af68";
        blue = "#7aa2f7";
        magenta = "#bb9af7";
        cyan = "#7dcfff";
        white = "#a9b1d6";
        orange = "#ff9e64";
      };

      ui = {
        pane_frames = {
          rounded_corners = true;
        };
      };
    };
  };
}
