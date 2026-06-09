{
  lib,
  config,
  ...
}:

{
  options = {
    # This creates the switch: sober.services.kanata.enable
    sober.services.kanata.enable = lib.mkEnableOption "Kanata Keyboard Remapping";
  };

  config = lib.mkIf config.sober.services.kanata.enable {
    # This code runs ONLY when the switch is ON
    services.kanata = {
      enable = true;
      keyboards.default = {
        devices = [ "/dev/input/by-path/platform-i8042-serio-0-event-kbd" ];
        extraDefCfg = "process-unmapped-keys yes";
        config = builtins.readFile ./kanata.kbd;
      };
    };

    # Permissions
    hardware.uinput.enable = true;
    users.groups.uinput.members = [ "michael" ];
    users.groups.input.members = [ "michael" ];
  };
}
