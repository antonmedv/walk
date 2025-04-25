# Release

1. Bump version in [main.go](main.go).
2. Commit changes.
3. Create a release on [GitHub](https://github.com/antonmedv/walk/releases).
4. Run `npx zx .github/scripts/build.mjs` to build and upload binaries.
5. Update version in [install.sh](install.sh)
