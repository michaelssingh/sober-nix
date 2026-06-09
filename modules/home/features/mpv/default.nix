{
  pkgs,
  ...
}:
{
  programs.mpv = {
    enable = true;
    scripts = [ pkgs.mpvScripts.mpv-playlistmanager ];
    config = {
      hwdec = "vaapi";
      vo = "gpu";
      gpu-context = "wayland";
      gpu-api = "opengl";
      ytdl-format = "bestvideo[height<=720][vcodec^=avc1]+bestaudio/best";
      cache = "yes";
      demuxer-max-bytes = "150MiB";
      demuxer-max-back-bytes = "75MiB";
      demuxer-readahead-secs = 30;
      profile = "fast";
    };
  };
}
