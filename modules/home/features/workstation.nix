{ pkgs, ... }:

{
  imports = [
    ./dev.nix
    ./media-cli.nix
  ];

  # Workstation-specific Shell Functions
  programs.fish.functions = {
    yt-search = ''
      if test -z "$argv"
        echo "Usage: yt-search <query>"
        return 1
      end
      
      set -l query (string join " " $argv)
      
      # Fetch results and let fzf handle selection
      set -l selection (yt-dlp --get-title --get-id --skip-download "ytsearch10:$query" | \
        paste - - | \
        awk -F'\t' '{print $1 " | " $2}' | \
        fzf --delimiter=' | ' --with-nth=1 --height=40% --reverse)
      
      if test -n "$selection"
        set -l id (string split "|" $selection)[2] | string trim
        mpv "https://www.youtube.com/watch?v=$id"
      end
    '';
    live = ''
      argparse a/audio -- $argv
      set -l socket "/tmp/mpv-socket"

      # Find a matching stream in the registry
      set -l match (grep -i "$argv" ~/git/sober-nix/modules/home/features/livestreams.txt)
      if test -n "$match"
        set -l url (string split "|" $match)[2]
        
        # Load the file
        mpv-queue "$url"
        
        # Wait until the socket is actually available (max 2 seconds)
        set -l count 0
        while not test -S "$socket"; and test $count -lt 20
          sleep 0.1
          set count (math $count + 1)
        end

        # Determine desired video state
        set -l video_state "yes"
        if set -q _flag_audio
          set video_state "no"
        end

        # Apply the state if socket is available
        if test -S "$socket"
          echo '{"command": ["set_property", "video", "'$video_state'"]}' | socat - "$socket"
        end
      else
        echo "Stream not found: $argv"
      end
    '';
    queue-unread = ''
      while read -l url
        # Filter only for YouTube video URLs
        if string match -q "*youtube.com/watch?v=*" "$url"; or string match -q "*youtu.be/*" "$url"
          mpv-queue "$url"
        end
      end
    '';
  };

  # Workstation-only Packages
  home.packages = with pkgs; [
    # DevOps / Cloud
    k9s
    kubectl
    opentofu
    awscli2
    oci-cli

    gemini-cli
    antigravity
    himalaya
    flyctl
    hydroxide
    ripgrep-all
    fastfetch
    bitwarden-cli
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
