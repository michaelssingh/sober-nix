{
  config,
  pkgs,
  ...
}:
let
  colors = config.sober.theme.current.colors;
in
{
  services.mako = {
    enable = true;
    settings = {
      font = "Inter 10";
      background-color = colors.bg_dark;
      text-color = colors.fg;
      border-color = colors.accent;
      border-radius = 5;
      border-size = 2;
      padding = "10";
      width = 450;
      height = 400;
      default-timeout = 5000;
      layer = "overlay";
      on-notify = "exec ${pkgs.pipewire}/bin/pw-play ${./tri-tone.mp3}";
    };
  };
}
