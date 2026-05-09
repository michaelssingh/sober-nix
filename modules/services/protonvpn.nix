{
  pkgs,
  lib,
  config,
  ...
}:

{
  options = {
    sober.services.protonvpn.enable = lib.mkEnableOption "ProtonVPN";
  };

  config = lib.mkIf config.sober.services.protonvpn.enable {
    # environment.systemPackages = with pkgs; [
    #   protonvpn-gui
    # ];

    networking.networkmanager.enable = true;
  };
}
