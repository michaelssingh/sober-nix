# home/features/youtube.nix
{
  config,
  pkgs,
  lib,
  ...
}:

{
  # Ensure these packages are available for the youtube workflow
  home.packages = with pkgs; [
    yt-dlp # For fetching video info and streams
    jq # Useful for parsing JSON, e.g., finding channel IDs
    chafa # For displaying image thumbnails in newsboat (optional)
  ];

  programs.newsboat = {
    enable = true;
    urls = [
      # Al Jazeera English
      {
        url = "https://www.youtube.com/feeds/videos.xml?channel_id=UCO3pA7nshlI86HscS8UvLhA";
        tags = [
          "news"
          "live"
        ];
      }
      # Yahoo Finance
      {
        url = "https://www.youtube.com/feeds/videos.xml?channel_id=UC0X6e_p5n_0W25m_6_v8Hcg";
        tags = [
          "finance"
          "live"
        ];
      }
      # Bloomberg
      {
        url = "https://www.youtube.com/feeds/videos.xml?channel_id=UCIALMKvObZNtJ6AmdCLP7Lg";
        tags = [
          "finance"
          "news"
        ];
      }
      # Your existing channel
      {
        url = "https://www.youtube.com/feeds/videos.xml?channel_id=UClx2N7nQJ0k2S3rJ7P25W6A";
        tags = [ "tech" ];
      }
    ];

    extraConfig = ''
      macro a set browser "${pkgs.mpv}/bin/mpv https://www.youtube.com/c/aljazeeraenglish/live" ; open-in-browser ; set browser "${pkgs.mpv}/bin/mpv"
      macro b set browser "${pkgs.mpv}/bin/mpv https://www.youtube.com/watch?v=dp8PhLsUcFE" ; open-in-browser ; set browser "${pkgs.mpv}/bin/mpv"
      macro f set browser "${pkgs.mpv}/bin/mpv https://www.youtube.com/c/YahooFinance/live" ; open-in-browser ; set browser "${pkgs.mpv}/bin/mpv"
      color listnormal         color253 default
      color listfocus          color234 color111 bold
      color listnormal_unread  color147 default  bold
      color listfocus_unread   color234 color147 bold
      color info               color222 color235
      color article            color253 default

      # Navigation
      bind-key j down
      bind-key k up
    '';
  };

  # Optional: Define XDG Base Directory Specification
  # This is good practice for managing dotfiles
  xdg.enable = true;
  xdg.dataHome = "${config.home.homeDirectory}/.local/share";
  xdg.configHome = "${config.home.homeDirectory}/.config";
  xdg.cacheHome = "${config.home.homeDirectory}/.cache";

  programs.mpv = {
    enable = true;
    config = {
      # Use Hardware Acceleration for the AMD A-series APU
      hwdec = "vaapi";
      vo = "gpu";
      gpu-context = "wayland"; # Use "x11" if you are using i3 instead of Sway

      # Force 480p and H.264 (avc1)
      # This is the most efficient format for your specific CPU
      ytdl-format = "bestvideo[height<=480][vcodec^=avc1]+bestaudio/best[ext=m4a]/best[height<=480]";

      # Modern cache settings (cache-initial and cache-secs are obsolete)
      cache = true;
      demuxer-max-bytes = "50MiB";
      demuxer-readahead-secs = 20;

      # Performance profile
      profile = "fast";
    };
    scripts = [ pkgs.mpvScripts.mpv-playlistmanager ];
  };

  home.file.".config/mpv/livestreams.m3u".text = ''
    #EXTM3U
    #EXTINF:-1,Al Jazeera English
    https://www.youtube.com/c/aljazeeraenglish/live
    #EXTINF:-1,Bloomberg Technology
    https://www.youtube.com/@markets/live
    #EXTINF:-1,Yahoo Finance
    https://www.youtube.com/c/yahoofinance/live
    #EXTINF:-1,DW News
    https://www.youtube.com/@dwnews/live
  '';
  home.shellAliases = {
    tv = "mpv --playlist=$HOME/.config/mpv/livestreams.m3u";
  };
}
