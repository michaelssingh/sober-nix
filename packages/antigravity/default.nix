{
  stdenv,
  fetchurl,
  autoPatchelfHook,
}:

stdenv.mkDerivation rec {
  pname = "antigravity";
  version = "1.0.5";

  src = fetchurl {
    url = "https://storage.googleapis.com/antigravity-public/antigravity-cli/1.0.5-5009297080451072/linux-x64/cli_linux_x64.tar.gz";
    sha512 = "72082d89ea71e101c7beb1630241428d53f68c702995897a2bbf55f162a9f71c8d2d7f98e7b310449cfbdb0b053f6e1b869c1f8d9fb23c95851e980d563d8924";
  };

  nativeBuildInputs = [ autoPatchelfHook ];

  dontUnpack = false;
  sourceRoot = ".";

  installPhase = ''
    mkdir -p $out/bin
    tar -xzf $src
    cp antigravity $out/bin/agy
    chmod +x $out/bin/agy
  '';
}
