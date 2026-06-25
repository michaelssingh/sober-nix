{ lib, buildGoModule, makeWrapper, ffmpeg, yt-dlp, mpv }:

buildGoModule {
  pname = "clare";
  version = "0.1.2";

  src = ./.;

  vendorHash = "sha256-SMhllO87YlmySHroKfPq1pHb67CwHaZ3XMp3t983etc=";

  nativeBuildInputs = [ makeWrapper ];

  postInstall = ''
    wrapProgram $out/bin/clare \
      --prefix PATH : ${lib.makeBinPath [ ffmpeg yt-dlp mpv ]}
  '';
}
