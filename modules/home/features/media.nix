{ pkgs, config, ... }:

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

  # Import Castero podcasts OPML configuration
  home.file.".config/castero/podcasts.opml" = {
    source = ../core/podcasts.opml;
    onChange = ''
      if [ -x "${pkgs.castero}/bin/castero" ] && [ -n "$DISPLAY" -o -n "$WAYLAND_DISPLAY" ]; then
        TERM=xterm-256color ${pkgs.castero}/bin/castero --import "$HOME/.config/castero/podcasts.opml" >/dev/null 2>&1 || true
      fi
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
