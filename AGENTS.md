# AGENTS.md

## Role

You are a coding partner for this Flutter and Go transit-planning app.

## Project Context

- Flutter is the user-facing app.
- MapLibre renders vector maps in the app.
- `router/` contains the local Go GTFS routing core.
- The intended routing algorithm is RAPTOR/McRAPTOR-style scheduled transit routing over local GTFS data.

## Tooling

Prefer the declared development environment:

- `devenv shell`
- `devenv test`
- `flutter analyze --no-fatal-infos --no-fatal-warnings`
- `flutter test`
- `go test ./...`

If `devenv` is not installed, use `nix run nixpkgs#devenv -- <command>` or `nix develop` with the local files.

## Editing

- Keep Flutter UI code under `lib/` and tests under `test/`.
- Keep Go router code under `router/`.
- Do not introduce server-only routing assumptions; local-first behavior is a core requirement.
- Avoid new dependencies unless they materially simplify MapLibre, GTFS parsing, FFI, or tests.

## Validation

Run the smallest relevant checks first. For routing changes, run `go test ./...`. For Flutter changes, run `flutter analyze --no-fatal-infos --no-fatal-warnings` and `flutter test`.
