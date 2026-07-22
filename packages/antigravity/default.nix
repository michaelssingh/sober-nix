{
  stdenv,
  fetchurl,
  autoPatchelfHook,
}:

stdenv.mkDerivation rec {
  pname = "antigravity";
  version = "1.1.5";

  src = fetchurl {
    url = "https://storage.googleapis.com/antigravity-public/antigravity-cli/1.1.5-5958982624477184/linux-x64/cli_linux_x64.tar.gz";
    sha512 = "906bff59c73bed630274f67efc77e3fa2064fe126f4d7521c7e58dc6773d8d1f3ce8688f1178f7ef65776728f782a8ebc211189617cfbb42e7df48f5614ad1f5";
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
