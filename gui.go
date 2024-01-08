//go:build gui || (!gui && !cli)

/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"strings"

	g "github.com/AllenDang/giu"

	// png decoder for icon
	_ "image/png"
	"strconv"
)

var (
	discords        []any
	radioIdx        int
	customChoiceIdx int

	customDir              string
	autoCompleteDir        string
	autoCompleteFile       string
	autoCompleteCandidates []string
	autoCompleteIdx        int
	lastAutoComplete       string
	didAutoComplete        bool

	modalId      = 0
	modalTitle   = "Oh No :("
	modalMessage = "You should never see this"

	acceptedOpenAsar bool

	win *g.MasterWindow
)

//go:embed winres/icon.png
var iconBytes []byte

func main() {

	go func() {
		<-GithubDoneChan
		g.Update()
	}()

	go func() {
		CheckSelfUpdate()
		g.Update()
	}()

	win = g.NewMasterWindow("Clicker Game", 1200, 800, 0)

	icon, _, err := image.Decode(bytes.NewReader(iconBytes))
	if err != nil {
		fmt.Println("Failed to load application icon", err)
		fmt.Println(iconBytes, len(iconBytes))
	} else {
		win.SetIcon([]image.Image{icon})
	}
	win.Run(loop)
}

func Tooltip(label string) g.Widget {
	return g.Style().
		SetStyle(g.StyleVarWindowPadding, 10, 8).
		SetStyleFloat(g.StyleVarWindowRounding, 8).
		To(
			g.Tooltip(label),
		)
}

func InfoModal(id, title, description string) g.Widget {
	return RawInfoModal(id, title, description, false)
}

func RawInfoModal(id, title, description string, isOpenAsar bool) g.Widget {
	isDynamic := strings.HasPrefix(id, "#modal") && !strings.Contains(description, "\n")
	return g.Style().
		SetStyle(g.StyleVarWindowPadding, 30, 30).
		SetStyleFloat(g.StyleVarWindowRounding, 12).
		To(
			g.PopupModal(id).
				Flags(g.WindowFlagsNoTitleBar | Ternary(isDynamic, g.WindowFlagsAlwaysAutoResize, 0)).
				Layout(
					g.Align(g.AlignCenter).To(
						g.Style().SetFontSize(30).To(
							g.Label(title),
						),
						g.Style().SetFontSize(20).To(
							g.Label(description).Wrapped(isDynamic),
						),
					),
				),
		)
}

func ShowModal(title, desc string) {
	modalTitle = title
	modalMessage = desc
	modalId++
	g.OpenPopup("#modal" + strconv.Itoa(modalId))
}

func renderInstaller() g.Widget {
	// wi, _ := win.GetSize()
	// w := float32(wi) - 96

	layout := g.Layout{}

	return layout
}

func loop() {
	g.PushWindowPadding(48, 48)

	g.SingleWindow().
		Layout(
			g.Align(g.AlignCenter).To(
				g.Style().SetFontSize(40).To(
					g.Label("Clicker Game"),
				),
			),
			g.Dummy(0, 20),
			g.Style().SetFontSize(20).To(
				g.Label("Cloud Version: "+InstallerTag+" ("+InstallerGitHash+")", "")),
			g.Label("Local Version: "+InstallerTag+" ("+InstallerGitHash+")"+Ternary(IsInstallerOutdated, " - OUTDATED", "")),
			g.Dummy(0, 20),
			g.Separator(),
			g.Dummy(0, 5),
			g.Style().SetFontSize(30).To(
				g.Label("Please select a difficulty:"),
			),
			g.Style().SetFontSize(20).To(
				g.Row(
					g.Style().
						SetColor(g.StyleColorButton, DiscordGreen).
						To(
							g.Button("Easy").
								OnClick(EasyMode).
								Size(20, 50),
							Tooltip("Easy Difficulty"),
						),
					g.Style().
						SetColor(g.StyleColorButton, DiscordYellow).
						To(
							g.Button("Medium").
								OnClick(MediumMode).
								Size(20, 50),
							Tooltip("Medium Difficulty"),
						),
					g.Style().
						SetColor(g.StyleColorButton, DiscordRed).
						To(
							g.Button("Hard").
								OnClick(HardMode).
								Size(20, 50),
							Tooltip("Hard Difficulty"),
						),
					g.Style().
						SetColor(g.StyleColorButton, DiscordBlue).
						To(
							g.Button("Custom").
								OnClick(CustomMode).
								Size(20, 50),
							Tooltip("Custom Option"),
						),
				),
			),
			InfoModal("#high-score", "High Score!", "You have set a new high score."),
			InfoModal("#modal"+strconv.Itoa(modalId), modalTitle, modalMessage),
		)

	g.PopStyle()
}
