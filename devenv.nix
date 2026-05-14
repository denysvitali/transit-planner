{ pkgs, ... }:

{
  packages = with pkgs; [
    flutter
    go
    git
    jdk21
  ];

  languages.go.enable = true;

  scripts.check-flutter.exec = ''
    flutter pub get
    flutter analyze --no-fatal-infos --no-fatal-warnings
    flutter test
  '';

  scripts.check-go.exec = ''
    go test ./...
  '';
}
