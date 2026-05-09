{ inputs }:
final: prev: {
  gemini-cli = final.stdenv.mkDerivation {
    pname = "gemini-cli";
    version = "0.41.2";

    # Download the pre-built package from NPM registry
    src = final.fetchurl {
      url = "https://registry.npmjs.org/@google/gemini-cli/-/gemini-cli-0.41.2.tgz";
      hash = "sha256-iA1FpPhnluwh2GMISpkyAlHJoqSWlBmpLzMTvVQkYBg=";
    };

    nativeBuildInputs = [ final.makeWrapper ];

    dontUnpack = true;

    installPhase = ''
      mkdir -p $out/lib/node_modules/gemini-cli
      tar -xzf $src --strip-components=1 -C $out/lib/node_modules/gemini-cli

      mkdir -p $out/bin
      makeWrapper ${final.nodejs}/bin/node $out/bin/gemini-cli \
        --add-flags "$out/lib/node_modules/gemini-cli/bundle/gemini.js"
    '';
  };
}
