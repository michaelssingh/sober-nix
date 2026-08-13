{ pkgs, ... }:

{
  home.packages = with pkgs; [
    # General Utilities
    pkgs.oci-cli
    pkgs.flyctl
    pkgs.rbw
    pkgs.dict
    pkgs.hydroxide
    pkgs.gemini-cli
    jq
    yq-go
    htop
    bottom
    nh # Nix Helper
    pkgs.socat
    pkgs.yazi
    pkgs.fastfetch
    pkgs.ripgrep-all

    # Typing practice tools
    pkgs.typioca
    pkgs.gtypist
    pkgs.ttyper
    pkgs.tt
  ];

  home.file.".dictrc".text = ''
    server dict.org
  '';

  home.file.".oci/config".text = ''
    [DEFAULT]
    user=ocid1.user.oc1..aaaaaaaavolviu7og6yniw6zaqrbqo3ealz44cocfrrvn2kmyrafuqqfithq
    fingerprint=82:d2:c1:3c:88:28:d4:23:29:c9:f1:3e:48:10:38:55
    tenancy=ocid1.tenancy.oc1..aaaaaaaazggxz72tdpddkfm4pbbswotzv5ryd6amcveo7rfsv2dst6q2spga
    region=us-ashburn-1
    key_file=/home/michael/.oci/oci_api_key.pem
  '';
}
