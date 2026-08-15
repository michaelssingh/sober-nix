{
  osConfig ? { },
  pkgs,
  ...
}:
let
  hostname = osConfig.networking.hostName or "";
  isNinox = hostname == "ninox";
in
{
  programs.mpv = {
    enable = true;
    scripts = with pkgs.mpvScripts; [
      mpv-playlistmanager
      mpris
      uosc
      thumbfast
      sponsorblock-minimal
      mpv-webm
    ];
    config = {
      hwdec = "vaapi";
      vo = "gpu";
      gpu-context = "wayland";
      gpu-api = "opengl";
      vd-lavc-dr = "yes";
      save-position-on-quit = "yes";
      save-watch-history = "yes";
      write-filename-in-watch-later-config = "yes";
      ytdl-format =
        if isNinox then
          "bestvideo[height<=1080]+bestaudio/best"
        else
          "bestvideo[height<=720][vcodec^=avc1]+bestaudio/best";
      ytdl-raw-options = "cookies-from-browser=firefox";
      cache = "yes";
      demuxer-max-bytes = "150MiB";
      demuxer-max-back-bytes = "75MiB";
      demuxer-readahead-secs = "30";
      profile = "fast";

      osc = "no";
      osd-bar = "no";
      border = "no";
    };
  };
}
