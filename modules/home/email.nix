{ pkgs, ... }:

{
  # --- Himalaya Email Configuration ---
  xdg.configFile."himalaya/config.toml".text = ''
    [accounts.protonmail]
    default = true
    email = "michaelssingh@protonmail.com"
    display-name = "Michael S. Singh"
    downloads-dir = "${builtins.getEnv "HOME"}/Downloads"
    backend.type = "proton"
    message.send.backend.type = "proton"
  '';
}
