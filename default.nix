{
  pkgs ? import <nixpkgs> {},
}: let
in {
  devEnv = with pkgs; stdenv.mkDerivation {
    name = "takt";
    nativeBuildInputs = [stdenv go musl socat];
    CGO_ENABLED = 0;
    ldflags = [
      "-linkmode external"
      "-extldflags '-static -L${musl}/lib'"
    ];
  };
}
