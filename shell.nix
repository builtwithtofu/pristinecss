{ pkgs ? import <nixpkgs> { } }:

pkgs.mkShell {
  buildInputs = [
    pkgs.gnumake
    pkgs.go
    pkgs.git
    pkgs.gotestfmt
  ];

  shellHook = ''
    export GOPATH=$HOME/go
    echo "Go version: $(go version)"
  '';
}


