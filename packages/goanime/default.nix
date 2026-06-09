{
  lib,
  buildGoModule,
  fetchFromGitHub,
}:

buildGoModule rec {
  pname = "goanime";
  version = "1.8.5";

  src = fetchFromGitHub {
    owner = "alvarorichard";
    repo = "GoAnime";
    rev = "v${version}";
    sha256 = "0q6f60c7phcvqqi4s6x16x491dj8gqcrcy1yfl7l1sl101cvsvdp";
  };

  vendorHash = "sha256-R+KuUJhHVVslO06isPENfWNB2zw4T6qpdAFWn9/Rjd4=";

  subPackages = [ "cmd/goanime" ];

  meta = with lib; {
    description = "A TUI tool to browse, stream, and download anime in PT-BR and EN";
    homepage = "https://github.com/alvarorichard/GoAnime";
    license = licenses.mit;
    mainProgram = "goanime";
  };
}
