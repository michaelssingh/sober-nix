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
    let
      typeParam = if entry ? isPlaylist && entry.isPlaylist then "playlist_id" else "channel_id";
    in
    {
      url = "https://www.youtube.com/feeds/videos.xml?${typeParam}=${entry.id}";
      inherit (entry) tags;
    };
in
{
  home.packages = with pkgs; [
    yt-dlp
    jq
    chafa
    fzf
    unstable.youtube-tui
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

      # Bindings
      bind-key o open-in-browser

      ignore-article "*" "link =~ \"shorts\""

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
}
