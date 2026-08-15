{ pkgs, config, ... }:

let
  theme = config.sober.theme.current;
in
{
  sops.secrets.anilist_token = { };
  sops.secrets.mal_token = { };

  home.sessionVariables = {
    ANILIST_TOKEN_FILE = config.sops.secrets.anilist_token.path;
    MAL_TOKEN_FILE = config.sops.secrets.mal_token.path;
  };

  # Workstation-specific Shell Functions for Livestreams
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

  home.packages = with pkgs; [
    (pkgs.writeShellApplication {
      name = "mpv-queue";
      runtimeInputs = [
        socat
        mpv
        coreutils
      ];
      text = ''
        SOCKET="/tmp/mpv-socket"

        # 1. Identify the actual URL/File vs flags
        # We assume the last argument is the target if it doesn't look like a flag,
        # or we just take the first non-flag argument.
        TARGET=""
        args=("$@")
        for i in "''${!args[@]}"; do
          if [[ "''${args[$i]}" != -* ]]; then
            TARGET="''${args[$i]}"
            # Remove target from args so we only have flags left
            unset "args[$i]"
            break
          fi
        done

        if [ -S "$SOCKET" ] && socat -t 1 - "$SOCKET" <<< '{ "command": ["get_property", "idle-active"] }' >/dev/null 2>&1; then
          # Already running: just append the target
          if [ -n "$TARGET" ]; then
            echo "{ \"command\": [\"loadfile\", \"$TARGET\", \"append-play\"] }" | socat -t 1 - "$SOCKET"
          fi
        else
          # Not running: start fresh
          rm -f "$SOCKET"
          # Launch detached
          nohup mpv --idle=yes --input-ipc-server="$SOCKET" --force-window=yes --really-quiet "''${args[@]}" "$TARGET" >/dev/null 2>&1 </dev/null &
          
          # Wait for socket
          for _ in {1..20}; do
            if [ -S "$SOCKET" ]; then break; fi
            sleep 0.1
          done
        fi
      '';
    })
    pkgs.socat
    pkgs.w3m
    pkgs.chafa
    pkgs.ani-skip
    pkgs.tyto
    pkgs.clare
    pkgs.castero
    pkgs.kjv
    pkgs.sox
  ];

  home.file.".w3m/config".text = ''
    ext_image_viewer 1
    external_image_viewer ${pkgs.chafa}/bin/chafa
    extbrowser ${pkgs.firefox}/bin/firefox %s
  '';

  # Import Castero podcasts OPML configuration & dynamic sober.theme configuration
  home.file.".config/castero/podcasts.opml" = {
    source = ../core/podcasts.opml;
    onChange = ''
      if [ -x "${pkgs.castero}/bin/castero" ] && [ -n "$DISPLAY" -o -n "$WAYLAND_DISPLAY" ]; then
        TERM=xterm-256color ${pkgs.castero}/bin/castero --import "$HOME/.config/castero/podcasts.opml" >/dev/null 2>&1 || true
      fi
    '';
  };

  home.file.".config/castero/castero.conf" = {
    force = true;
    text = ''
      # Declaratively generated from sober.theme (${theme.name})
      [client]
      restrict_memory_usage = False
      delete_feed_confirmation = False
      reload_feeds_threshold = 10
      max_episodes = -1
      retain_absent_episodes = False
      default_layout = 1
      disable_vertical_borders = False
      clean_html_descriptions = True
      refresh_delay = 30
      player = mpv
      execute_command =
      proxy_http =
      proxy_https =
      add_only_unplayed_episodes = False

      [feeds]
      reload_on_start = False

      [downloads]
      custom_download_dir =
      request_timeout = 3

      [colors]
      color_foreground = ${if theme.isDark then "cyan" else "blue"}
      color_background = ${if theme.isDark then "black" else "white"}
      color_foreground_alt = ${if theme.isDark then "black" else "white"}
      color_background_alt = ${if theme.isDark then "blue" else "cyan"}
      color_foreground_dim = ${if theme.isDark then "black" else "white"}
      color_foreground_status = cyan
      color_foreground_heading = yellow
      color_foreground_dividers = magenta

      [playback]
      seek_distance_forward = 30
      seek_distance_backward = 10
      default_playback_speed = 1.0
      default_volume = 100
      volume_adjust_distance = 5
      resume_rewind_distance = 0

      [keys]
      key_help = h
      key_exit = q
      key_add_feed = a
      key_remove = d
      key_reload = r
      key_reload_selected = R
      key_save = s
      key_delete = x
      key_up = UP
      key_right = RIGHT
      key_down = DOWN
      key_left = LEFT
      key_scroll_up = PPAGE
      key_scroll_down = NPAGE
      key_play_selected = ENTER
      key_add_selected = SPACE
      key_clear = c
      key_clear_progress = z
      key_next = n
      key_execute = e
      key_invert = i
      key_filter = /
      key_mark_played = m
      key_pause_play = p
      key_pause_play_alt = k
      key_seek_forward = f
      key_seek_forward_alt = l
      key_seek_backward = b
      key_seek_backward_alt = j
      key_rate_increase = ]
      key_rate_decrease = [
      key_volume_increase = =
      key_volume_decrease = -
      key_show_url = u
    '';
  };

  # Desktop entries for Rofi / Fuzzel application launchers
  xdg.desktopEntries = {
    clare = {
      name = "Clare Media Player";
      genericName = "Anime, Movies & TV Shows TUI";
      comment = "Terminal media streaming client for anime, movies, and TV shows";
      exec = "clare";
      terminal = true;
      categories = [
        "AudioVideo"
        "Video"
        "Player"
        "ConsoleOnly"
      ];
      type = "Application";
      icon = "mpv";
    };
    castero = {
      name = "Castero Podcasts";
      genericName = "Terminal Podcast Client";
      comment = "Terminal TUI podcast client";
      exec = "castero";
      terminal = true;
      categories = [
        "AudioVideo"
        "Audio"
        "ConsoleOnly"
      ];
      type = "Application";
      icon = "multimedia-audio-player";
    };
  };
}
