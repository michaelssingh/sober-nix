{
  lib,
  stdenvNoCC,
  makeWrapper,
  gnugrep,
  gnused,
  curl,
  fzf,
  ffmpeg,
  aria2,
  yt-dlp,
  openssl,
  mpv,
  ani-skip,
  src,
}:

stdenvNoCC.mkDerivation {
  pname = "ani-cli";
  version = "4.14-custom";

  inherit src;

  patches = [ ./always-subtitles.patch ];

  nativeBuildInputs = [ makeWrapper ];

  postPatch = ''
    substituteInPlace ani-cli \
      --replace 'version_number="4.14.1"' 'version_number="4.14.1-sober"' \
      --replace 'nohup $player_function $skip_flag $audio_flag --tls-verify=no --force-media-title' 'nohup $player_function --script=@LUA_SCRIPT@ $skip_flag $audio_flag --tls-verify=no --force-media-title' \
      --replace '$player_function $skip_flag $refr_flag $audio_flag --tls-verify=no --force-media-title' '$player_function --script=@LUA_SCRIPT@ $skip_flag $refr_flag $audio_flag --tls-verify=no --force-media-title'
  '';

  installPhase = ''
    runHook preInstall

    install -Dm755 ani-cli $out/bin/ani-cli
    install -Dm644 ${./save-position.lua} $out/share/ani-cli/save-position.lua

    substituteInPlace $out/bin/ani-cli \
      --replace '@LUA_SCRIPT@' "$out/share/ani-cli/save-position.lua"

    wrapProgram $out/bin/ani-cli \
      --prefix PATH : ${
        lib.makeBinPath [
          openssl
          gnugrep
          gnused
          curl
          fzf
          ffmpeg
          aria2
          yt-dlp
          mpv
          ani-skip
        ]
      }

    runHook postInstall
  '';

  meta = {
    homepage = "https://github.com/pystardust/ani-cli";
    description = "Cli tool to browse and play anime";
    license = lib.licenses.gpl3Plus;
    mainProgram = "ani-cli";
  };
}
