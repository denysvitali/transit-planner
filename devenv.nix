{ pkgs, ... }:

{
  packages = with pkgs; [
    flutter
    go
    git
    jdk21
    sqlc
  ];

  languages.go.enable = true;

  scripts.generate-sqlc.exec = ''
    sqlc generate
  '';

  scripts.check-sqlc.exec = ''
    sqlc generate
    git diff --exit-code -- router/gtfsdb/db
  '';

  scripts.check-flutter.exec = ''
    flutter pub get
    flutter analyze --no-fatal-infos --no-fatal-warnings
    flutter test
  '';

  scripts.check-go.exec = ''
    check-sqlc
    go test ./...
  '';

  scripts.check.exec = ''
    check-go
    check-flutter
  '';
}
