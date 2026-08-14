_:

{
  services.swaync = {
    enable = true;
    settings = {
      positionX = "right";
      positionY = "top";
      layer = "top";
      control-center-width = 380;
      control-center-height = 600;
      notification-window-width = 350;
      timeout = 5;
      timeout-low = 2;
      timeout-critical = 0;
      fit-to-screen = true;
    };
  };
}
