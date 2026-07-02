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
  version = "0.1.19";

  src = ./.;

  vendorHash = "sha256-yAmydoJZXlipqhZsjojoPA3uoI8BhaU4sPzs9OZ1+3w=";

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
