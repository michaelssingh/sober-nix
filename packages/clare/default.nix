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
  version = "0.1.37";

  src = ./.;

  vendorHash = "sha256-yAmydoJZXlipqhZsjojoPA3uoI8BhaU4sPzs9OZ1+3w=";

  overrideModAttrs = (_: {
    preBuild = ''
      export HOME=$(mktemp -d)
    '';
  });

  preBuild = ''
    export HOME=$(mktemp -d)
  '';

  nativeBuildInputs = [ makeWrapper ];

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
