{
  pkgs,
  ...
}:
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
      ytdl-format = "bestvideo[height<=720][vcodec^=avc1]+bestaudio/best";
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
