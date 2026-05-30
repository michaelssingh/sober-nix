{
  lib,
  buildGoModule,
  fetchFromGitHub,
  makeWrapper,
  mpv,
  go,
}:

buildGoModule rec {
  pname = "goanime";
  version = "1.8.5";

  inherit go;

  src = fetchFromGitHub {
    owner = "alvarorichard";
    repo = "Goanime";
    rev = "v${version}";
    hash = "sha256-t229WQCB6kAPdT54lhl+SLaQSDehG00ixpvBexgwzmA=";
  };

  vendorHash = "sha256-R+KuUJhHVVslO06isPENfWNB2zw4T6qpdAFWn9/Rjd4=";

  doCheck = false;

  nativeBuildInputs = [ makeWrapper ];

  postInstall = ''
    wrapProgram $out/bin/goanime \
      --prefix PATH : ${lib.makeBinPath [ mpv ]}
  '';

  meta = with lib; {
    description = "A TUI tool to browse, stream, and download anime";
    homepage = "https://github.com/alvarorichard/Goanime";
    license = licenses.mit;
    maintainers = [ ];
  };
}
