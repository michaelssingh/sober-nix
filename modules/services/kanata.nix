{
  lib,
  config,
  pkgs,
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

    # Early Boot (initrd) Kanata support for LUKS password prompt
    boot.initrd = {
      systemd.enable = true;
      kernelModules = [ "uinput" ];
      systemd.services.kanata = {
        description = "Kanata Keyboard Remapper (initrd)";
        unitConfig = {
          DefaultDependencies = false;
        };
        wantedBy = [ "initrd-root-device.target" ];
        before = [ "sysinit.target" ];
        after = [ "systemd-udevd.service" ];
        serviceConfig = {
          ExecStart = "${pkgs.kanata}/bin/kanata --cfg ${pkgs.writeText "kanata.kbd" (builtins.readFile ./kanata.kbd)} --devices /dev/input/by-path/platform-i8042-serio-0-event-kbd --passthrough";
          Restart = "always";
        };
      };
    };

    # Permissions
    hardware.uinput.enable = true;
    users.groups.uinput.members = [ "michael" ];
    users.groups.input.members = [ "michael" ];
  };
}
