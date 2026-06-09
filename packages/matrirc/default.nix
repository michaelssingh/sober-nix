{
  lib,
  rustPlatform,
  fetchFromGitHub,
  openssl,
  pkg-config,
  sqlite,
}:

rustPlatform.buildRustPackage rec {
  pname = "matrirc";
  version = "unstable-2026-06-06";

  src = fetchFromGitHub {
    owner = "martinetd";
    repo = "matrirc";
    rev = "45f4d6d5482293162c48a77505fcb20cb0d9278b";
    sha256 = "058vrs90hp78qnvjnmrcd2fin736n1v2k29a5wv0jb2h2c540nhh";
  };

  cargoHash = "sha256-PaEh0uaftDiOxvzPsaGGU2jutkB69Xu1Z91Co9NFUC4=";

  nativeBuildInputs = [ pkg-config ];
  buildInputs = [
    openssl
    sqlite
  ];

  meta = with lib; {
    description = "An IRC gateway to Matrix";
    homepage = "https://github.com/martinetd/matrirc";
    maintainers = [ ];
  };
}
