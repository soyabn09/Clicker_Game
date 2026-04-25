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
ClickerGame-v3.0.0.exe
ClickerGame-v3.0.0-macos.zip
ClickerGame-v3.0.0-linux.tar.gz
```

## Build

```powershell
go build -tags gui -ldflags "-X main.appVersion=v3.0.0" -o ClickerGame-v3.0.0.exe .
```
