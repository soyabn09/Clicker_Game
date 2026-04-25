package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	g "github.com/AllenDang/giu"
)

var appVersion = "v3.0.0"

const releasesURL = "https://github.com/soyabn09/Game/releases"

//go:embed winres/icon.png
var iconBytes []byte

type screen int

const (
	screenMenu screen = iota
	screenGame
	screenCredits
)

type gameMode struct {
	ID          string
	Name        string
	Seconds     int32
	Description string
	Color       color.RGBA
}

func updaterLayout() []g.Widget {
	status := updaterSnapshot()

	title := "Checking for updates"
	switch status.State {
	case updateRequired:
		title = "Update required"
	case updateDownloading:
		title = "Downloading update"
	case updateRestarting:
		title = "Installing update"
	case updateFailed:
		title = "Update check failed"
	}

	widgets := []g.Widget{
		g.Align(g.AlignCenter).To(
			g.Style().SetFontSize(40).To(g.Label("Clicker Game")),
		),
		g.Dummy(0, 35),
		g.Align(g.AlignCenter).To(
			g.Style().SetFontSize(28).To(g.Label(title)),
		),
		g.Dummy(0, 15),
		g.Align(g.AlignCenter).To(
			g.Label(status.Message),
		),
	}

	if status.Latest != "" {
		widgets = append(widgets,
			g.Dummy(0, 10),
			g.Align(g.AlignCenter).To(
				g.Label("Current: "+appVersion+"    Latest: "+status.Latest),
			),
		)
	}

	if status.State == updateFailed {
		widgets = append(widgets,
			g.Dummy(0, 25),
			g.Align(g.AlignCenter).To(
				g.Button("Retry").Size(180, 45).OnClick(func() {
					go checkForUpdates()
				}),
			),
		)
	}

	return widgets
}

var (
	green  = color.RGBA{R: 0x2D, G: 0x7C, B: 0x46, A: 0xFF}
	yellow = color.RGBA{R: 0xFE, G: 0xE7, B: 0x5C, A: 0xFF}
	red    = color.RGBA{R: 0xEC, G: 0x41, B: 0x44, A: 0xFF}
	blue   = color.RGBA{R: 0x58, G: 0x65, B: 0xF2, A: 0xFF}

	modes = map[string]gameMode{
		"easy": {
			ID:          "easy",
			Name:        "EASY",
			Seconds:     60,
			Description: "YOU HAVE A MINUTE TO CLICK AS FAST AS YOU CAN",
			Color:       green,
		},
		"medium": {
			ID:          "medium",
			Name:        "MEDIUM",
			Seconds:     30,
			Description: "YOU HAVE 30 SECONDS TO CLICK AS FAST AS YOU CAN",
			Color:       yellow,
		},
		"hard": {
			ID:          "hard",
			Name:        "HARD",
			Seconds:     15,
			Description: "YOU HAVE 15 SECONDS TO CLICK AS FAST AS YOU CAN",
			Color:       red,
		},
		"custom": {
			ID:          "custom",
			Name:        "CUSTOM",
			Seconds:     60,
			Description: "YOU CAN APPLY ANY AMOUNT OF TIME TO YOUR SELF",
			Color:       blue,
		},
	}

	currentScreen = screenMenu
	activeMode    = modes["easy"]
	score         int32
	highScore     int32
	timeLeft      int32
	customSeconds int32 = 60
	running       bool
	lastTick      time.Time

	modalID      int
	modalTitle   string
	modalMessage string
)

func main() {
	if err := loadScores(); err != nil {
		fmt.Println("Failed to create score directory:", err)
	}
	go checkForUpdates()

	win := g.NewMasterWindow("Clicker Game", 1200, 800, 0)
	if icon, _, err := image.Decode(bytes.NewReader(iconBytes)); err == nil {
		win.SetIcon([]image.Image{icon})
	}
	win.Run(loop)
}

func loop() {
	updateTimer()

	widgets := layoutForScreen()
	if modalID > 0 {
		g.OpenPopup("#modal" + strconv.Itoa(modalID))
	}
	widgets = append(widgets, modal())

	g.PushWindowPadding(48, 48)
	g.SingleWindow().Layout(widgets...)
	g.PopStyle()
}

func modal() g.Widget {
	return g.PopupModal("#modal" + strconv.Itoa(modalID)).
		Flags(g.WindowFlagsAlwaysAutoResize).
		Layout(
			g.Align(g.AlignCenter).To(
				g.Style().SetFontSize(30).To(g.Label(modalTitle)),
				g.Dummy(0, 10),
				g.Style().SetFontSize(20).To(g.Label(modalMessage)),
				g.Dummy(0, 15),
				g.Button("OK").Size(180, 32).OnClick(func() {
					g.CloseCurrentPopup()
					modalID = 0
				}),
			),
		)
}

func layoutForScreen() []g.Widget {
	if !canUseGame() {
		return updaterLayout()
	}

	switch currentScreen {
	case screenGame:
		return gameLayout()
	case screenCredits:
		return creditsLayout()
	default:
		return menuLayout()
	}
}

func menuLayout() []g.Widget {
	return []g.Widget{
		g.Align(g.AlignCenter).To(
			g.Style().SetFontSize(40).To(g.Label("Clicker Game")),
		),
		g.Dummy(0, 20),
		g.Align(g.AlignCenter).To(
			g.Style().SetFontSize(30).To(g.Label("Please select a difficulty:")),
		),
		g.Dummy(0, 20),
		g.Align(g.AlignCenter).To(
			g.Row(
				modeButton(modes["easy"], "Easy Difficulty"),
				modeButton(modes["medium"], "Medium Difficulty"),
				modeButton(modes["hard"], "Hard Difficulty"),
				modeButton(modes["custom"], "Custom Option"),
			),
		),
		g.Dummy(0, 30),
		g.Align(g.AlignCenter).To(
			g.Button("Credits").Size(220, 40).OnClick(func() {
				currentScreen = screenCredits
			}),
		),
		g.Dummy(0, 30),
		g.Align(g.AlignCenter).To(
			g.Label("Version: " + appVersion),
		),
	}
}

func modeButton(mode gameMode, tooltip string) g.Widget {
	return g.Style().
		SetColor(g.StyleColorButton, mode.Color).
		To(
			g.Button(strings.Title(strings.ToLower(mode.Name))).
				OnClick(func() { startMode(mode) }).
				Size(190, 60),
			Tooltip(tooltip),
		)
}

func gameLayout() []g.Widget {
	return []g.Widget{
		g.Align(g.AlignCenter).To(
			g.Style().SetFontSize(40).To(g.Label(activeMode.Name)),
		),
		g.Dummy(0, 20),
		g.Row(
			g.Style().SetFontSize(28).To(g.Label("Score: "+strconv.Itoa(int(score)))),
			g.Dummy(60, 0),
			g.Style().SetFontSize(28).To(g.Label("High Score: "+strconv.Itoa(int(highScore)))),
			g.Dummy(60, 0),
			g.Style().SetFontSize(28).To(g.Label("Time: "+strconv.Itoa(int(timeLeft)))),
		),
		g.Dummy(0, 20),
		g.Align(g.AlignCenter).To(
			g.Label(activeMode.Description),
		),
		g.Dummy(0, 20),
		customControls(),
		g.Dummy(0, 15),
		g.Align(g.AlignCenter).To(
			g.Row(
				g.Style().SetColor(g.StyleColorButton, activeMode.Color).To(
					g.Button("CLICK ME").Size(240, 70).OnClick(click),
				),
				g.Button("RESTART").Size(180, 70).OnClick(restart),
			),
		),
		g.Dummy(0, 20),
		g.Align(g.AlignCenter).To(
			g.Row(
				g.Button("RESET HIGHSCORE").Size(220, 45).OnClick(resetHighScore),
				g.Button("BACK TO MENU").Size(220, 45).OnClick(func() {
					stopGame()
					currentScreen = screenMenu
				}),
			),
		),
		g.Dummy(0, 30),
		g.Align(g.AlignCenter).To(
			g.Label("(c) 2020-2026 Soayb Nandhla    Version: " + appVersion),
		),
	}
}

func customControls() g.Widget {
	if activeMode.ID != "custom" {
		return g.Dummy(0, 0)
	}

	return g.Align(g.AlignCenter).To(
		g.Row(
			g.Label("Custom seconds:"),
			g.InputInt(&customSeconds).Label("##custom-seconds").Size(140),
			g.Button("APPLY").Size(120, 32).OnClick(func() {
				timeLeft = validateCustomSeconds(true)
			}),
		),
	)
}

func creditsLayout() []g.Widget {
	return []g.Widget{
		g.Align(g.AlignCenter).To(
			g.Style().SetFontSize(40).To(g.Label("Credits")),
		),
		g.Dummy(0, 20),
		g.Align(g.AlignCenter).To(
			g.Style().SetFontSize(22).To(g.Label("Developer: Soayb Nandhla")),
		),
		g.Dummy(0, 25),
		g.Align(g.AlignCenter).To(
			g.Row(
				g.Button("RELEASES").Size(180, 45).OnClick(func() {
					if err := openExternalURL(releasesURL); err != nil {
						showModal("Open Error", err.Error())
					}
				}),
				g.Button("BACK").Size(180, 45).OnClick(func() {
					currentScreen = screenMenu
				}),
			),
		),
	}
}

func Tooltip(label string) g.Widget {
	return g.Style().
		SetStyle(g.StyleVarWindowPadding, 10, 8).
		SetStyleFloat(g.StyleVarWindowRounding, 8).
		To(g.Tooltip(label))
}

func openExternalURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

func startMode(mode gameMode) {
	activeMode = mode
	score = 0
	running = false
	if mode.ID == "custom" {
		timeLeft = validateCustomSeconds(false)
	} else {
		timeLeft = mode.Seconds
	}
	highScore = int32(getHighScore(mode.ID))
	currentScreen = screenGame
}

func click() {
	if timeLeft <= 0 {
		return
	}
	if !running {
		running = true
		lastTick = time.Now()
	}
	score++
}

func restart() {
	score = 0
	running = false
	if activeMode.ID == "custom" {
		timeLeft = validateCustomSeconds(true)
	} else {
		timeLeft = activeMode.Seconds
	}
}

func resetHighScore() {
	highScore = 0
	if err := setHighScore(activeMode.ID, 0); err != nil {
		showModal("Save Error", err.Error())
	}
}

func updateTimer() {
	if !running {
		return
	}

	now := time.Now()
	for timeLeft > 0 && now.Sub(lastTick) >= time.Second {
		timeLeft--
		lastTick = lastTick.Add(time.Second)
	}
	if timeLeft <= 0 {
		finishGame()
	}
}

func finishGame() {
	running = false
	if score > highScore {
		highScore = score
		if err := setHighScore(activeMode.ID, int(score)); err != nil {
			showModal("Save Error", err.Error())
			return
		}
		showModal("High Score!", "TIMES UP\nSCORE: "+strconv.Itoa(int(score))+"\nYou have set a new high score.")
		return
	}
	showModal("Times Up", "TIMES UP\nSCORE: "+strconv.Itoa(int(score)))
}

func stopGame() {
	running = false
}

func validateCustomSeconds(alert bool) int32 {
	if customSeconds >= 3601 {
		if alert {
			showModal("Invalid Time", "The number is above 1 hour\nNumber reset to default")
		}
		customSeconds = 60
		return 60
	}
	if customSeconds <= 0 {
		if alert {
			showModal("Invalid Time", "The number is below 0\nNumber reset to default")
		}
		customSeconds = 60
		return 60
	}
	return customSeconds
}

func showModal(title, message string) {
	modalID++
	modalTitle = title
	modalMessage = message
}
