package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"strconv"
	"time"

	g "github.com/AllenDang/giu"
)

var appVersion = "3.0.2"

const (
	windowWidth  = 450
	windowHeight = 410
)

//go:embed winres/icon.png
var iconBytes []byte

type screen int

const (
	screenMenu screen = iota
	screenGame
)

type gameMode struct {
	ID          string
	Name        string
	Seconds     int32
	Description string
	Color       color.RGBA
}

var (
	green  = color.RGBA{R: 0x16, G: 0x5C, B: 0x32, A: 0xFF}
	yellow = color.RGBA{R: 0xB8, G: 0x92, B: 0x12, A: 0xFF}
	red    = color.RGBA{R: 0xA8, G: 0x1F, B: 0x24, A: 0xFF}
	blue   = color.RGBA{R: 0x25, G: 0x38, B: 0xA8, A: 0xFF}

	modes = map[string]gameMode{
		"easy": {
			ID:          "easy",
			Name:        "EASY",
			Seconds:     60,
			Description: "YOU HAVE 60 SECONDS TO CLICK AS FAST AS YOU CAN",
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
			Description: "YOU CAN APPLY ANY AMOUNT OF TIME TO YOURSELF",
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
	timerEndsAt   time.Time

	modalID      int
	modalOpen    bool
	modalTitle   string
	modalMessage string
)

func main() {
	if err := loadScores(); err != nil {
		fmt.Println("Failed to create score directory:", err)
	}

	win := g.NewMasterWindow("How fast can you click?", windowWidth, windowHeight, g.MasterWindowFlagsNotResizable)
	if icon, _, err := image.Decode(bytes.NewReader(iconBytes)); err == nil {
		win.SetIcon([]image.Image{icon})
	}
	go refreshUI()
	win.Run(loop)
}

func refreshUI() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		g.Update()
	}
}

func loop() {
	updateTimer()
	ensureScoreFileAvailable()

	widgets := layoutForScreen()
	widgets = append(widgets, modal())

	g.PushWindowPadding(16, 22)
	g.SingleWindow().Layout(widgets...)
	g.PopStyle()
}

func ensureScoreFileAvailable() {
	if err := ensureScoresFile(); err != nil && modalID == 0 {
		showModal("Save Error", err.Error())
	}
}

func modal() g.Widget {
	return g.Custom(func() {
		if modalID <= 0 {
			return
		}

		name := modalTitle + "##modal"
		if modalOpen {
			g.OpenPopup(name)
			modalOpen = false
		}

		g.SetNextWindowSize(300, 0)
		g.PopupModal(name).
			Layout(
				g.Dummy(0, 4),
				g.Label(modalMessage).Wrapped(true),
				g.Dummy(0, 12),
				g.Align(g.AlignCenter).To(
					g.Button("OK").Size(120, 28).OnClick(func() {
						g.CloseCurrentPopup()
						modalID = 0
						modalOpen = false
					}),
				),
			).Build()
	})
}

func layoutForScreen() []g.Widget {
	switch currentScreen {
	case screenGame:
		return gameLayout()
	default:
		return menuLayout()
	}
}

func menuLayout() []g.Widget {
	return []g.Widget{
		g.Align(g.AlignCenter).To(
			g.Style().SetFontSize(18).To(g.Label("MAIN MENU")),
		),
		g.Dummy(0, 16),
		g.Align(g.AlignCenter).To(
			modeButton(modes["easy"], "Easy Difficulty"),
		),
		g.Dummy(0, 3),
		g.Align(g.AlignCenter).To(
			modeButton(modes["medium"], "Medium Difficulty"),
		),
		g.Dummy(0, 3),
		g.Align(g.AlignCenter).To(
			modeButton(modes["hard"], "Hard Difficulty"),
		),
		g.Dummy(0, 3),
		g.Align(g.AlignCenter).To(
			modeButton(modes["custom"], "Custom Option"),
		),
		g.Dummy(0, 3),
		g.Align(g.AlignCenter).To(
			g.Row(
				g.Label("(c) 2020-2026 Soyab Nandhla"),
				g.Dummy(54, 0),
				g.Label("Version: "+appVersion),
			),
		),
	}
}

func modeButton(mode gameMode, tooltip string) g.Widget {
	return g.Style().
		SetColor(g.StyleColorButton, mode.Color).
		To(
			g.Button(mode.Name).
				OnClick(func() { startMode(mode) }).
				Size(383, 28),
			Tooltip(tooltip),
		)
}

func gameLayout() []g.Widget {
	return []g.Widget{
		g.Align(g.AlignCenter).To(
			g.Style().SetFontSize(24).To(g.Label(activeMode.Name)),
		),
		g.Dummy(0, 10),
		g.Align(g.AlignCenter).To(
			g.Row(
				g.Style().SetFontSize(20).To(g.Label("Score: "+strconv.Itoa(int(score)))),
				g.Dummy(18, 0),
				g.Style().SetFontSize(20).To(g.Label("High Score: "+strconv.Itoa(int(highScore)))),
				g.Dummy(18, 0),
				g.Style().SetFontSize(20).To(g.Label("Time: "+strconv.Itoa(int(timeLeft)))),
			),
		),
		g.Dummy(0, 20),
		g.Align(g.AlignCenter).To(
			g.Label(activeMode.Description),
		),
		g.Dummy(0, 10),
		customControls(),
		g.Dummy(0, 10),
		g.Align(g.AlignCenter).To(
			g.Row(
				g.Style().SetColor(g.StyleColorButton, activeMode.Color).To(
					g.Button("CLICK ME").Size(170, 50).OnClick(click),
				),
				g.Button("RESTART").Size(120, 50).OnClick(restart),
			),
		),
		g.Dummy(0, 10),
		g.Align(g.AlignCenter).To(
			g.Row(
				g.Button("RESET HIGHSCORE").Size(178, 40).OnClick(resetHighScore),
				g.Button("BACK TO MENU").Size(178, 40).OnClick(func() {
					stopGame()
					currentScreen = screenMenu
				}),
			),
		),
		g.Dummy(0, 12),
		g.Align(g.AlignCenter).To(
			g.Row(
				g.Label("(c) 2020-2026 Soyab Nandhla"),
				g.Dummy(54, 0),
				g.Label("Version: "+appVersion),
			),
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
			g.InputInt(&customSeconds).Label("##custom-seconds").Size(120),
			g.Button("APPLY").Size(120, 40).OnClick(func() {
				timeLeft = validateCustomSeconds(true)
				if running {
					timerEndsAt = time.Now().Add(time.Duration(timeLeft) * time.Second)
				}
				highScore = int32(getHighScore(scoreKey()))
			}),
		),
	)
}

func Tooltip(label string) g.Widget {
	return g.Style().
		SetStyle(g.StyleVarWindowPadding, 10, 8).
		SetStyleFloat(g.StyleVarWindowRounding, 8).
		To(g.Tooltip(label))
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
	highScore = int32(getHighScore(scoreKey()))
	currentScreen = screenGame
}

func click() {
	if timeLeft <= 0 {
		return
	}
	if !running {
		running = true
		timerEndsAt = time.Now().Add(time.Duration(timeLeft) * time.Second)
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
	highScore = int32(getHighScore(scoreKey()))
}

func resetHighScore() {
	highScore = 0
	if err := setHighScore(scoreKey(), 0); err != nil {
		showModal("Save Error", err.Error())
	}
}

func updateTimer() {
	if !running {
		return
	}

	now := time.Now()
	remaining := timerEndsAt.Sub(now)
	if remaining <= 0 {
		timeLeft = 0
		finishGame()
		return
	}
	timeLeft = int32((remaining + time.Second - 1) / time.Second)
}

func finishGame() {
	running = false
	if score > highScore {
		highScore = score
		if err := setHighScore(scoreKey(), int(score)); err != nil {
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

func scoreKey() string {
	if activeMode.ID != "custom" {
		return activeMode.ID
	}
	return "custom:" + strconv.Itoa(int(validateCustomSeconds(false)))
}

func showModal(title, message string) {
	modalID++
	modalOpen = true
	modalTitle = title
	modalMessage = message
}
