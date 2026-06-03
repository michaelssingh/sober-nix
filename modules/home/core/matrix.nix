{ pkgs, ... }:

{
  programs.iamb = {
    enable = true;

    settings = {
      default_profile = "matrix.org";

      profiles."matrix.org" = {
        user_id = "@michaelssingh:matrix.org";
      };

      settings = {
        # General application settings
        notifications.enabled = true;
        
        # Configure image previews
        # iamb supports Sixel, which works well with Foot
        image_preview = {
          protocol.type = "sixel";
        };
        
        # UI Preferences
        message.read_receipt_send = true;
        message.typing_notice_send = true;
      };
    };
  };
}
