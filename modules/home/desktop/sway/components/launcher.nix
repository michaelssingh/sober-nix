{
  programs.fuzzel = {
    enable = true;
    settings = {
      main = {
        font = "FiraCode Nerd Font Mono:size=11";
        prompt = "'❯  '";
        icon-theme = "Papirus-Dark";
        width = 40;
      };
      colors = {
        background = "1a1b26ff";
        text = "c0caf5ff";
        match = "bb9af7ff";
        selection = "7aa2f7ff";
        selection-text = "1a1b26ff";
        border = "7aa2f7ff";
      };
      border = {
        width = 2;
        radius = 5;
      };
    };
  };
}
