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
    command = lib.mkOption {
      type = lib.types.str;
      default = "sway";
      description = "Default session command for tuigreet";
    };
  };

  config = {
    services.greetd = {
      enable = true;
      settings = {
        default_session = {
          command = "${pkgs.tuigreet}/bin/tuigreet --time --remember --cmd 'bash -l -c ${cfg.command}'";
          user = "greeter";
        };
      };
    };

    security.pam.services.greetd.enableGnomeKeyring = true;
  };
}
