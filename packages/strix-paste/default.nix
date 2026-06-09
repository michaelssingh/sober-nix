{
  lib,
  buildGoModule,
  makeWrapper,
  wl-clipboard,
}:

buildGoModule {
  pname = "strix-paste";
  version = "0.1.0";
  src = ./.;

  vendorHash = null;

  nativeBuildInputs = [ makeWrapper ];

  postInstall = ''
    wrapProgram $out/bin/strix-paste \
      --prefix PATH : ${lib.makeBinPath [ wl-clipboard ]}
  '';

  meta = {
    description = "CLI helper to upload content to the sober-strix pastebin";
    license = lib.licenses.mit;
    mainProgram = "strix-paste";
  };
}
