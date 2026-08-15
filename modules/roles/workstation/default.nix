{ pkgs, ... }:

{
  # Disable kernel ALSA power save to prevent DAC power down / speaker popping
  boot.extraModprobeConfig = ''
    options snd_hda_intel power_save=0 power_save_controller=N
  '';

  # --- 1. Sound (Pipewire) ---
  security.rtkit.enable = true;
  services.pipewire = {
    enable = true;
    alsa.enable = true;
    alsa.support32Bit = true;
    pulse.enable = true;
    jack.enable = true;

    wireplumber = {
      enable = true;
      extraConfig = {
        "10-disable-suspend" = {
          "monitor.alsa.rules" = [
            {
              matches = [
                { "node.name" = "~alsa_output.*"; }
              ];
              actions = {
                update-props = {
                  "session.suspend-timeout-seconds" = 0;
                };
              };
            }
          ];
        };
      };
    };

    extraConfig.pipewire."99-input-agc-denoise" = {
      "context.modules" = [
        {
          name = "libpipewire-module-echo-cancel";
          args = {
            "monitor.mode" = true;
            "source.props" = {
              "node.name" = "echo_cancelled_source";
              "node.description" = "Microphone (AGC & Noise Cancellation)";
              "priority.driver" = 2000;
              "priority.session" = 2000;
            };
            "aec.args" = {
              "webrtc.gain_control" = true;
              "webrtc.extended_filter" = true;
              "webrtc.noise_suppression" = true;
              "webrtc.voice_detection" = true;
            };
          };
        }
      ];
    };
  };

  # --- 2. Graphics (Hardware Acceleration) ---
  hardware.graphics = {
    enable = true;
  };

  # --- 3. Fonts ---
  fonts.packages = with pkgs; [
    nerd-fonts.jetbrains-mono
    nerd-fonts.symbols-only
    noto-fonts-color-emoji
    font-awesome
  ];

  # --- 4. System Services ---
  # XDG Portals (Needed for file dialogs)
  # xdg.portal = {
  #   enable = true;
  #   wlr.enable = true;
  #   extraPortals = [ pkgs.xdg-desktop-portal-gtk ];
  # };
  xdg.portal = {
    enable = true;
    extraPortals = [ pkgs.xdg-desktop-portal-gtk ];

    # ADD THIS BLOCK:
    config = {
      common = {
        default = [ "gtk" ];
      };
    };
  };
  # Keyring (Needed for saving WiFi passwords)
  services.gnome.gnome-keyring.enable = true;
  security.pam.services.login.enableGnomeKeyring = true;
  security.pam.services.swaylock = {
    fprintAuth = false;
  };

  # Polkit rule to permit wheel group users to enroll/manage fingerprints
  security.polkit.extraConfig = ''
    polkit.addRule(function(action, subject) {
      if (action.id.indexOf("net.reactivated.fprint.") === 0 && subject.isInGroup("wheel")) {
        return polkit.Result.YES;
      }
    });
  '';
  # DRM/KMS Hardware Accelerated High-Res TTY Virtual Console
  services.kmscon = {
    enable = true;
    hwRender = true;
    extraConfig = "font-size=16";
    fonts = [
      {
        name = "JetBrainsMono Nerd Font";
        package = pkgs.nerd-fonts.jetbrains-mono;
      }
    ];
  };

  # --- 5. Firmware & Microcode ---
  services.fwupd.enable = true;
  hardware.enableRedistributableFirmware = true;
}
