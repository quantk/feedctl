## 1. Nix flake shell

- [x] 1.1 Add `flake.nix` with a default dev shell for common Linux and Darwin systems.
- [x] 1.2 Pin nixpkgs through `flake.lock`.
- [x] 1.3 Include `go_1_25`, `sqlite`, `gopls`, Go helper tools, and `pkg-config` in the dev shell.
- [x] 1.4 Configure the shell with `CGO_ENABLED=1`.

## 2. Direnv entry point

- [x] 2.1 Add a minimal `.envrc` that uses the flake dev shell.

## 3. Verification

- [x] 3.1 Verify `nix develop -c go version` reports Go 1.25.x.
- [x] 3.2 Verify `nix develop -c sqlite3 --version` succeeds.
- [x] 3.3 Verify `nix develop -c gopls version` succeeds.
- [x] 3.4 Verify `nix develop -c go env CGO_ENABLED` prints `1`.
- [x] 3.5 Verify `nix develop -c pkg-config --version` succeeds.
