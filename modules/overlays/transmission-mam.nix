{ inputs, ... }:
final: prev:
{
  transmission-mam = (import inputs.nixpkgs-4-0-5 {
    system = final.system;
    config = { allowUnfree = true; };
  }).transmission_4.overrideAttrs (old: {
    pname = "transmission-mam";
  });
}
