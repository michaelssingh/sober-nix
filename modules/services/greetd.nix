{
  pkgs,
  config,
  lib,
  ...
}:

let
  cfg = config.sober.services.greetd;
in
{
  options.sober.services.greetd = {
    enable = lib.mkEnableOption "greetd display manager" // {
      default = true;
    };
    greeter = lib.mkOption {
      type = lib.types.enum [
        "tuigreet"
        "regreet"
      ];
      default = "tuigreet";
      description = "Greeter engine: tuigreet (TUI) or regreet (GTK4 Wayland GUI)";
    };
    command = lib.mkOption {
      type = lib.types.str;
      default = "sway";
      description = "Default session command for the greeter";
    };
  };

  config = lib.mkIf cfg.enable (
    lib.mkMerge [
      (lib.mkIf (cfg.greeter == "tuigreet") {
        services.greetd = {
          enable = true;
          settings = {
            default_session = {
              command = "${pkgs.tuigreet}/bin/tuigreet --time --remember --cmd 'bash -l -c ${cfg.command}'";
              user = "greeter";
            };
          };
        };
      })

      (lib.mkIf (cfg.greeter == "regreet") {
        programs.regreet = {
          enable = true;
          font = {
            name = "Inter";
            package = pkgs.inter;
          };
          cursorTheme = {
            name = "Bibata-Modern-Classic";
            package = pkgs.bibata-cursors;
          };
          settings = {
            background = {
              fit = "Cover";
            };
            GTK = {
              application_prefer_dark_theme = true;
            };
          };
          extraCss = ''
            window {
              background-color: #1a1b26;
              color: #c0caf5;
            }
            button {
              border-radius: 8px;
              background-color: #24283b;
              color: #7aa2f7;
            }
            button:hover {
              background-color: #7aa2f7;
              color: #1a1b26;
            }
          '';
        };
      })

      {
        security.pam.services.greetd.enableGnomeKeyring = true;
      }
    ]
  );
}
