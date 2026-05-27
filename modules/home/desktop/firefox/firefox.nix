{ pkgs, ... }:

let
  addons = import ./extensions.nix;
in
{
  programs.firefox = {
    enable = true;

    nativeMessagingHosts = [ pkgs.ff2mpv-rust ];

    policies = {
      DisableTelemetry = true;
      ExtensionSettings =
        (builtins.mapAttrs (id: url: {
          install_url = url;
          installation_mode = "force_installed";
        }) addons)
        // {
          "*" = {
            installation_mode = "allowed";
          };
        };
    };

    profiles.x = {
      isDefault = true;
      id = 0;
      name = "x";
      settings = {
        "extensions.activeThemeID" = "firefox-compact-dark@mozilla.org";
        "browser.theme.content-theme" = 0;
        "browser.theme.toolbar-theme" = 0;
        "toolkit.legacyUserProfileCustomizations.stylesheets" = true;
        "layout.css.prefers-color-scheme.content-override" = 0;
        "browser.uidensity" = 1;
        "browser.compactmode.show" = true;
        "browser.sessionstore.interval" = 600000; # 10 minutes for crash resilience
        "network.prefetch-next" = false; # Do not prefetch links
        "layers.acceleration.force-enabled" = false;
        "gfx.webrender.enabled" = false;
        "media.ffmpeg.vaapi.enabled" = true;
        "media.ffvpx.enabled" = false;
        "media.rdd-ffmpeg.enabled" = true;
        "browser.tabs.remote.autostart" = true;
        "image.mem.decode_on_draw" = true;
        "browser.cache.disk.enable" = false; # RAM cache is faster
        "media.mediasource.vp9.enabled" = false;
        "media.av1.enabled" = false;
        "dom.ipc.processCount" = 2;
        "browser.tabs.unloadOnLowMemory" = true;
        "browser.sessionstore.max_tabs_undo" = 5;
        "browser.cache.memory.enable" = true;
        "browser.cache.memory.max_entry_size" = 5120;
        "browser.startup.homepage" = "https://sober.fyi";
      };

      # This pulls in the file we just created!
      userChrome = builtins.readFile ./userChrome.css;
      # userContent = builtins.readFile ./userContent.css;
    };
  };
}
