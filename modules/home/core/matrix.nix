{ pkgs, ... }:

{
  programs.iamb = {
    enable = true;

    settings = {
      default_profile = "matrix.org";

      profiles = {
        "matrix.org" = {
          user_id = "@michaelssingh:matrix.org";
        };
        athene = {
          user_id = "@init:sober.fyi";
          url = "http://sober-athene.flycast:6167";
        };
      };

      settings = {
        # General application settings
        # Configure notifications
        notifications = {
          enabled = true;
          show_message = true;
          via = "desktop";
        };

        # Configure image previews
        # iamb supports Sixel, which works well with Foot
        image_preview = {
          protocol.type = "sixel";
        };

        # UI Preferences
        message.read_receipt_send = true;
        message.typing_notice_send = true;
        username_display = "username";
      };
    };
  };
}
