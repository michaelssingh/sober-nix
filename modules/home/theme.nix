{ pkgs, ... }:

{
  # --- Cursor ---
  home.pointerCursor = {
    package = pkgs.simp1e-cursors;
    name = "Simp1e-Tokyo-Night";
    size = 24;
    gtk.enable = true;
    x11.enable = true;
  };

  # --- GTK Theme ---
  gtk = {
    enable = true;
    theme = {
      name = "TokyoNight-Dark";
      package = pkgs.tokyonight-gtk-theme;
    };
    iconTheme = {
      name = "Papirus-Dark";
      package = pkgs.papirus-icon-theme;
    };
  };

  # --- Qt Theme (Optional but recommended for uniform look) ---
  qt = {
    enable = true;
    platformTheme.name = "gtk";
    style.name = "gtk2";
  };
}
