{
  pkgs,
  ...
}:
{
  programs.mpv = {
    enable = true;
    scripts = [
      pkgs.mpvScripts.mpv-playlistmanager
      pkgs.mpvScripts.mpris
    ];
    config = {
      hwdec = "auto-safe";
      vo = "gpu";
      gpu-context = "auto";
      gpu-api = "auto";
      save-position-on-quit = "yes";
      write-filename-in-watch-later-config = "yes";
      ytdl-format = "bestvideo[height<=720][vcodec^=avc1]+bestaudio/best";
      cache = "yes";
      demuxer-max-bytes = "150MiB";
      demuxer-max-back-bytes = "75MiB";
      demuxer-readahead-secs = 30;
      profile = "fast";
    };
  };
}
