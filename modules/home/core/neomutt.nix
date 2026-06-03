{ pkgs, ... }:
{
  programs.neomutt = {
    enable = true;
    vimKeys = true;
    sidebar.enable = true;
    extraConfig = ''
      # Sidebar Navigation
      bind index <up> sidebar-prev
      bind index <down> sidebar-next
      bind index <right> sidebar-open
      bind index <left> sidebar-toggle-visible

      # HTML Rendering
      auto_view text/html
      set mailcap_path = "~/.mailcap"
      alternative_order text/plain text/enriched text/html

      # Sane Defaults
      set sort = threads
      set sort_aux = last-date-received
      set mail_check = 60
      set timeout = 30
      set editor = "nvim"
      set sleep_time = 0 
      set markers = no
    '';
  };
}
