# home/features/youtube.nix
{ config, pkgs, lib, ... }:

let
  csvData = builtins.readFile ./subscriptions.csv;
  lines = lib.splitString "\n" csvData;
  # Skip header and filter empty lines
  entries = lib.filter (l: l != "" && !lib.hasPrefix "Channel Id" l) lines;
  
  toNewsboatUrl = line:
    let
      parts = lib.splitString "," line;
      id = lib.head parts;
    in "https://www.youtube.com/feeds/videos.xml?channel_id=${id}";
in
{
  home.packages = with pkgs; [ yt-dlp jq chafa ];

  programs.newsboat = {
    enable = true;
    urls = builtins.map (l: { url = toNewsboatUrl l; }) entries;
    extraConfig = ''
      color listnormal         color253 default
      color listfocus          color234 color111 bold
      color listnormal_unread  color147 default  bold
      color listfocus_unread   color234 color147 bold
      color info               color222 color235
      color article            color253 default
      bind-key j down
      bind-key k up
    '';
  };

  programs.mpv = {
    enable = true;
    config = {
      hwdec = "vaapi";
      vo = "gpu";
      gpu-context = "wayland";
      gpu-api = "opengl";
      ytdl-format = "bestvideo[height<=1080][vcodec^=avc1]+bestaudio/best";
      cache = "yes";
      demuxer-max-bytes = "50MiB";
      demuxer-max-back-bytes = "25MiB";
      demuxer-readahead-secs = 20;
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
}
