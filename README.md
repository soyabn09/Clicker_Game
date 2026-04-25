# Clicker Game

Clicker Game is a standalone Windows desktop game built with Go and GIU.

## Requirements

Install these tools before building:

- [MSYS2](https://www.msys2.org/)
- Go and GCC from the **MSYS2 MINGW64** environment
- `go-winres` for embedding the Windows icon and version metadata

Open **MSYS2 MINGW64** and install the required packages:

```bash
pacman -S --needed git mingw-w64-x86_64-go mingw-w64-x86_64-gcc mingw-w64-x86_64-SDL2
```

Install `go-winres`:

```bash
export GOROOT=/mingw64/lib/go
export GOPATH=/mingw64

go install github.com/tc-hib/go-winres@latest
```

## Run From Source

From **MSYS2 MINGW64**, open the repository folder and run:

```bash
export GOROOT=/mingw64/lib/go
export GOPATH=/mingw64

CGO_ENABLED=1 go run -tags static .
```

## Build The Windows EXE

From **MSYS2 MINGW64**, open the repository folder and run:

```bash
export GOROOT=/mingw64/lib/go
export GOPATH=/mingw64

go-winres make

CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -v -tags static \
  -ldflags "-s -w -H=windowsgui -extldflags=-static" \
  -o "ClickerGame.exe" .
```

The built application will be created as:

```text
ClickerGame.exe
```

## Version Updates

Before creating a release, update both places:

- `appVersion` in `main.go`
- `FileVersion` and `ProductVersion` in `winres/winres.json`

Then rebuild `ClickerGame.exe` and upload it manually to your GitHub Releases page.

## Scores

Scores are stored in the current Windows user's config directory:

```text
%AppData%\Clicker Game\scores.json
```

Set `CLICKER_DATA_DIR` to store `scores.json` somewhere else.

## Windows Resources

Windows icon and metadata files live in:

```text
winres/
```

Run `go-winres make` again after changing `winres/winres.json` or replacing the icon.
