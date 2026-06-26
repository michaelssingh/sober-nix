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

      # Added 'ff' to guarantee Mako parses the colors correctly
      background-color = "${colors.bg_dark}ff";
      text-color = "${colors.fg}ff";
      border-color = "${colors.accent}ff";

      border-radius = 5;
      border-size = 2;
      padding = "10";
      default-timeout = 5000;
      layer = "overlay";
      on-notify = "exec ${pkgs.pipewire}/bin/pw-play ${./tri-tone.mp3}";

      "app-name=System Deploy" = {
        width = 450; # Wide enough for paths/hashes
        height = 400; # High ceiling that shrinks to fit the log size
        default-timeout = 0; # Keeps it on screen until dismissed
      };
    };
  };
}
