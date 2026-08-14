# home/features/youtube.nix
{
  pkgs,
  lib,
  config,
  ...
}:

let
  theme = config.sober.theme.current;
  inherit (theme) colors;
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
    urls = map toNewsboatUrl (lib.toList subscriptions);
    extraConfig = ''
      # Default browser: w3m
      browser "${pkgs.w3m}/bin/w3m %u"

      # Macro 'm' to enqueue video in mpv
      # Sets browser temporarily to mpv-queue, opens, then reverts
      macro m set browser "mpv-queue %u" ; open-in-browser ; set browser "${pkgs.w3m}/bin/w3m %u"

      # Bindings
      bind-key o open-in-browser

      ignore-article "*" "link =~ \"shorts\""

      # sober.theme (${theme.name}) Color Scheme
      color listnormal        ${colors.fg} default
      color listfocus         ${colors.black} ${colors.accent} bold
      color listnormal_unread ${colors.cyan} default bold
      color listfocus_unread  ${colors.black} ${colors.cyan} bold
      color info              ${colors.yellow} ${colors.bg_dark}
      color article           ${colors.fg} default
      color background        default default

      bind-key j down
      bind-key k up
    '';
  };

  xdg.desktopEntries.newsboat = {
    name = "Newsboat RSS Reader";
    genericName = "RSS/Atom Feed Reader";
    comment = "Terminal RSS/Atom feed reader for YouTube and Blogs";
    exec = "newsboat";
    terminal = true;
    categories = [
      "Network"
      "News"
      "ConsoleOnly"
    ];
    type = "Application";
    icon = "newsboat";
  };
}
