{
  config,
  ...
}:
let
  colors = config.sober.theme.current.colors;
in
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

      # Theme
      color index ${colors.fg} default ".*"
      color index_author ${colors.blue} default ".*"
      color index_number ${colors.comment} default
      color index_subject ${colors.fg} default ".*"
      color sidebar_indicator ${colors.accent} ${colors.bg_dark}
      color sidebar_highlight ${colors.fg} ${colors.bg_highlight}
      color sidebar_divider ${colors.comment} ${colors.bg_dark}
      color status ${colors.fg_sidebar} ${colors.bg_sidebar}
      color indicator ${colors.accent} ${colors.bg_visual}
      color tree ${colors.comment} default

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
