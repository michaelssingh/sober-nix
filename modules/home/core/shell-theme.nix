{
  config,
  ...
}:
let
  colors = config.sober.theme.current.colors;
in
{
  programs.fzf = {
    enable = true;
    enableFishIntegration = true;
    # --- Official Tokyo Night FZF Logic (Ref: extras/fzf/) ---
    defaultOptions = [
      "--highlight-line"
      "--info=inline-right"
      "--ansi"
      "--layout=reverse"
      "--border=none"
      "--color=bg+:${colors.bg_visual}"
      "--color=bg:${colors.bg_dark}"
      "--color=border:${colors.border_highlight}"
      "--color=fg:${colors.fg}"
      "--color=gutter:${colors.bg_dark}"
      "--color=header:${colors.orange}"
      "--color=hl+:${colors.blue1}"
      "--color=hl:${colors.blue1}"
      "--color=info:${colors.fg_dark}"
      "--color=marker:${colors.magenta2}"
      "--color=pointer:${colors.magenta2}"
      "--color=prompt:${colors.blue1}"
      "--color=query:${colors.fg}:regular"
      "--color=scrollbar:${colors.border_highlight}"
      "--color=separator:${colors.orange}"
      "--color=spinner:${colors.magenta2}"
    ];
  };

  # --- Official Tokyo Night Fish Pager (Ref: extras/fish/) ---
  programs.fish = {
    enable = true;
    interactiveShellInit = ''
      set -g fish_color_normal "${colors.fg}"
      set -g fish_color_command "${colors.cyan}"
      set -g fish_color_keyword "${colors.magenta}"
      set -g fish_color_quote "${colors.yellow}"
      set -g fish_color_redirection "${colors.fg}"
      set -g fish_color_end "${colors.orange}"
      set -g fish_color_error "${colors.red}"
      set -g fish_color_param "${colors.magenta}"
      set -g fish_color_comment "${colors.comment}"
      set -g fish_color_selection --background="${colors.bg_visual}"
      set -g fish_color_search_match --background="${colors.bg_visual}"
      set -g fish_color_operator "${colors.green}"
      set -g fish_color_escape "${colors.magenta}"
      set -g fish_color_autosuggestion "${colors.comment}"

      set -g fish_pager_color_progress "${colors.comment}"
      set -g fish_pager_color_prefix "${colors.cyan}"
      set -g fish_pager_color_completion "${colors.fg}"
      set -g fish_pager_color_description "${colors.comment}"
      set -g fish_pager_color_selected_background --background="${colors.bg_visual}"
    '';
  };
}
