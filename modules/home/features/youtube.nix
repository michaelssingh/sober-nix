# home/features/youtube.nix
{
  config,
  pkgs,
  lib,
  ...
}:

let
  subscriptions = import ./subscriptions.nix;

  toNewsboatUrl =
    entry:
    {
      url = "https://www.youtube.com/feeds/videos.xml?channel_id=${entry.id}";
      inherit (entry) tags;
    };
in
{
  home.packages = with pkgs; [
    yt-dlp
    jq
    chafa
    fzf
  ];

  programs.newsboat = {
    enable = true;
    urls = builtins.map toNewsboatUrl (lib.toList subscriptions);
    extraConfig = ''
      # Default browser: w3m
      browser "${pkgs.w3m}/bin/w3m %u"

      # Macro 'm' to enqueue video in mpv
      # Sets browser temporarily to mpv-queue, opens, then reverts
      macro m set browser "mpv-queue %u" ; open-in-browser ; set browser "${pkgs.w3m}/bin/w3m %u"
      
      # Macro 'A' to queue all unread articles (YouTube only)
      macro A print-unread | queue-unread

      # Bindings
      bind-key o open-in-browser

      # Highlight Shorts (orange) - using link URL match
      highlight-article "link =~ \"shorts\"" color208 default

      color listnormal         color253 default
      color listfocus          color234 color111 bold
      color listnormal_unread  color147 default  bold
      color listfocus_unread   color234 color147 bold
      color info               color222 color235
      color article            color253 default
      bind-key j down
      bind-key k up
    '';
    };  programs.mpv = {
    enable = true;
    scripts = [ pkgs.mpvScripts.mpv-playlistmanager ];
  };

  home.file.".config/mpv/mpv.conf".text = ''
    hwdec=vaapi
    vo=gpu
    gpu-context=wayland
    gpu-api=opengl
    ytdl-format=bestvideo[height<=720][vcodec^=avc1]+bestaudio/best
    cache=yes
    demuxer-max-bytes=150MiB
    demuxer-max-back-bytes=75MiB
    demuxer-readahead-secs=30
    profile=fast
  '';
}
