# Clicker Game

Standalone Go GUI version of Clicker Game. This directory contains everything needed for the app and does not depend on the older source folders.

## Run

Install Go, then run:

```powershell
go run .
```

The app checks GitHub releases on startup. If a newer release exists, it downloads and installs the update before the game menu is available.

## Scores

Scores are stored in:

```text
%AppData%\Clicker Game\scores.json
```

The first run migrates old scores from:

```text
C:\CLICKER\EASY.txt
C:\CLICKER\MEDIUM.txt
C:\CLICKER\HARD.txt
C:\CLICKER\CUSTOM.txt
```

Set `CLICKER_DATA_DIR` to store `scores.json` somewhere else.

## Updates

The updater checks:

```text
https://api.github.com/repos/soyabn09/Game/releases/latest
```

Release assets are named with the Git tag. For tag `v3.0.0`, the workflow publishes:

```text
ClickerGame-v3.0.0-windows.exe
ClickerGame-v3.0.0-macos.zip
ClickerGame-v3.0.0-linux.tar.gz
```

## Build with GitHub CI

Builds are created only by the GitHub Actions release workflow. Do not build release files locally.

1. Commit and push all release-ready changes to GitHub.
2. Create a new version tag using the `vX.Y.Z` format:

   ```powershell
   git tag v3.0.0
   ```

   For a prerelease build, include a suffix after the version:

   ```powershell
   git tag v3.1.0-beta.1
   ```

   Tags containing `-` are published as GitHub prereleases.

3. Push the tag to GitHub:

   ```powershell
   git push origin v3.0.0
   ```

   For a prerelease tag:

   ```powershell
   git push origin v3.1.0-beta.1
   ```

4. Open the repository on GitHub and go to **Actions**.
5. Wait for the **Release** workflow to finish. It builds Windows, macOS, and Linux packages.
6. Go to **Releases** and confirm the new release contains the generated assets:

   ```text
   ClickerGame-v3.0.0-windows.exe
   ClickerGame-v3.0.0-macos.zip
   ClickerGame-v3.0.0-linux.tar.gz
   ```

   Prerelease builds use the prerelease tag in the asset names:

   ```text
   ClickerGame-v3.1.0-beta.1-windows.exe
   ClickerGame-v3.1.0-beta.1-macos.zip
   ClickerGame-v3.1.0-beta.1-linux.tar.gz
   ```
