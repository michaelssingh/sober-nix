{
  config,
  lib,
  osConfig ? { },
  ...
}:
let
  colors = config.sober.theme.current.colors;
  termCmd = if ((osConfig.networking.hostName or "") == "otus") then "foot" else "ghostty";
in
{
  programs.fuzzel = {
    enable = true;
    settings = {
      main = {
        font = "Inter:size=11";
        terminal = termCmd;
        prompt = "'❯ '";
      };
      # --- Official Tokyo Night Fuzzel Logic (Ref: extras/fuzzel/) ---
      colors = {
        background = lib.removePrefix "#" colors.bg_dark + "ff";
        text = lib.removePrefix "#" colors.fg + "ff";
        match = lib.removePrefix "#" colors.blue1 + "ff";
        selection = lib.removePrefix "#" colors.bg_highlight + "ff";
        selection-match = lib.removePrefix "#" colors.blue1 + "ff";
        selection-text = lib.removePrefix "#" colors.fg + "ff";
        border = lib.removePrefix "#" colors.border_highlight + "ff";
      };
    };
  };
}
