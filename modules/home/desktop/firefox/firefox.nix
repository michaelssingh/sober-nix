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
        "browser.sessionstore.interval" = 1800000; # Protect your 128GB SSD
        "layers.acceleration.force-enabled" = false;
        "gfx.webrender.force-disabled" = true;
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
