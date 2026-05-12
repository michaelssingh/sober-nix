{ inputs }:
final: prev: {
  gemini-cli = final.buildNpmPackage {
    pname = "gemini-cli";
    version = "0.41.2";

    src = final.fetchFromGitHub {
      owner = "google-gemini";
      repo = "gemini-cli";
      rev = "v0.41.2";
      hash = "sha256-4jwEviWYzan97pVn0RWfWU4XS8c27L4ZJUwa2iGlFxY=";
    };

    npmDepsHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";

    # Disable integration tests during build to save time/resources
    makeCacheWritable = true;
    npmBuildFlags = [ "--" "--skip-tests" ];

    # Fedora base on the builder might need these
    nativeBuildInputs = [ final.python3 final.pkg-config ];

    meta = {
      description = "Official Gemini CLI";
      homepage = "https://github.com/google-gemini/gemini-cli";
      license = final.lib.licenses.asl20;
    };
  };
}
