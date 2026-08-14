_:

{
  programs.hyprlock = {
    enable = true;
    settings = {
      general = {
        disable_loading_bar = true;
        grace = 0;
        hide_cursor = true;
        no_fade_in = false;
      };

      background = [
        {
          monitor = "";
          path = "screenshot";
          blur_passes = 3;
          blur_size = 8;
          color = "rgba(26, 27, 38, 1.0)";
        }
      ];

      input-field = [
        {
          monitor = "";
          size = "250, 50";
          outline_thickness = 3;
          dots_size = 0.2;
          dots_spacing = 0.6;
          dots_center = true;
          outer_color = "rgba(122, 162, 247, 1.0)";
          inner_color = "rgba(26, 27, 38, 0.8)";
          font_color = "rgba(192, 202, 245, 1.0)";
          fade_on_empty = false;
          placeholder_text = "<i>Password...</i>";
          hide_input = false;
          position = "0, -50";
          halign = "center";
          valign = "center";
        }
      ];

      label = [
        {
          monitor = "";
          text = "$TIME";
          color = "rgba(187, 154, 247, 1.0)";
          font_size = 64;
          font_family = "Inter Bold";
          position = "0, 150";
          halign = "center";
          valign = "center";
        }
      ];
    };
  };
}
