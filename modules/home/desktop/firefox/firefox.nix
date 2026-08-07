{
  pkgs,
  osConfig ? { },
  ...
}:

let
  addons = import ./extensions.nix;
in
{
  programs.firefox = {
    enable = true;
    configPath = ".mozilla/firefox";

    nativeMessagingHosts = [ pkgs.ff2mpv-rust ];

    policies = {
      DisableTelemetry = true;
      ExtensionSettings =
        (builtins.mapAttrs (_id: url: {
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
        "browser.tabs.remote.autostart" = true;
        "image.mem.decode_on_draw" = true;
        "browser.cache.disk.enable" = false; # RAM cache is faster
        "browser.tabs.unloadOnLowMemory" = true;
        "browser.sessionstore.max_tabs_undo" = 5;
        "browser.cache.memory.enable" = true;
        "browser.cache.memory.max_entry_size" = 5120;
        "browser.startup.homepage" = "https://sober.fyi";
      }
      // (
        if (osConfig.networking.hostName or "") == "ninox" then
          {
            # Hardware acceleration enabled for Ninox (AMD iGPU)
            "layers.acceleration.force-enabled" = true;
            "layers.acceleration.disabled" = false;
            "gfx.webrender.all" = true;
            "gfx.webrender.enabled" = true;
            "media.ffmpeg.vaapi.enabled" = true;
            "media.hardware-video-decoding.enabled" = true;
            "media.rdd-ffmpeg.enabled" = true;
          }
        else
          {
            # Hardware acceleration disabled for lower-powered hosts like Otus
            "layers.acceleration.force-enabled" = false;
            "layers.acceleration.disabled" = true;
            "gfx.webrender.force-disabled" = true;
            "gfx.webrender.all" = false;
            "media.ffmpeg.vaapi.enabled" = false;
            "media.hardware-video-decoding.enabled" = false;
            "media.ffvpx.enabled" = false;
            "media.rdd-ffmpeg.enabled" = false;
          }
      );

      # This pulls in the file we just created!
      userChrome = builtins.readFile ./userChrome.css;
      # userContent = builtins.readFile ./userContent.css;
    };
  };
}
