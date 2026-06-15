{
  stdenv,
  fetchurl,
  autoPatchelfHook,
}:

stdenv.mkDerivation rec {
  pname = "antigravity";
  version = "1.0.8";

  src = fetchurl {
    url = "https://storage.googleapis.com/antigravity-public/antigravity-cli/1.0.8-5963827121094656/linux-x64/cli_linux_x64.tar.gz";
    sha512 = "78426a61f9295d75285d9cdd39e9b9dc2736346468c92ce86a6fde1bcc7a9d6dd32c7bed1903a5deacd874a05d75babf5b907e991a117ad63f33548113c61bc2";
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
