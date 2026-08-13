{
  lib,
  buildGoModule,
  makeWrapper,
  ffmpeg,
  yt-dlp,
  mpv,
}:

buildGoModule {
  pname = "clare";
  version = "0.2.0";

  src = ./.;

  vendorHash = "sha256-yAmydoJZXlipqhZsjojoPA3uoI8BhaU4sPzs9OZ1+3w=";

  doCheck = false;

  overrideModAttrs = _: {
    preBuild = ''
      export HOME=$(mktemp -d)
    '';
  };

  preBuild = ''
    export HOME=$(mktemp -d)
  '';

  nativeBuildInputs = [ makeWrapper ];

  checkFlags = [
    "-skip=TestLiveAPIIntegration"
  ];

  postInstall = ''
    wrapProgram $out/bin/clare \
      --suffix PATH : ${
        lib.makeBinPath [
          ffmpeg
          yt-dlp
          mpv
        ]
      }
  '';
}
