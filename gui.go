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
	"errors"
	"fmt"
	g "github.com/AllenDang/giu"
	"github.com/AllenDang/imgui-go"
	"image"
	"image/color"
	// png decoder for icon
	_ "image/png"
	"os"
	path "path/filepath"
	"runtime"
	"strconv"
	"strings"
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
	InitGithubDownloader()
	discords = FindDiscords()

	customChoiceIdx = len(discords)

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

type CondWidget struct {
	predicate  bool
	ifWidget   func() g.Widget
	elseWidget func() g.Widget
}

func (w *CondWidget) Build() {
	if w.predicate {
		w.ifWidget().Build()
	} else if w.elseWidget != nil {
		w.elseWidget().Build()
	}
}

func getChosenInstall() *DiscordInstall {
	var choice *DiscordInstall
	if radioIdx == customChoiceIdx {
		choice = ParseDiscord(customDir, "")
		if choice == nil {
			g.OpenPopup("#invalid-custom-location")
		}
	} else {
		choice = discords[radioIdx].(*DiscordInstall)
	}
	return choice
}

func InstallLatestBuilds() (err error) {
	if IsDevInstall {
		return
	}

	err = installLatestBuilds()
	if err != nil {
		ShowModal("Uh Oh!", "Failed to install the latest Vencord builds from GitHub:\n"+err.Error())
	}
	return
}

func handlePatch() {
	choice := getChosenInstall()
	if choice != nil {
		choice.Patch()
	}
}

func handleUnpatch() {
	choice := getChosenInstall()
	if choice != nil {
		choice.Unpatch()
	}
}

func handleOpenAsar() {
	if acceptedOpenAsar || getChosenInstall().IsOpenAsar() {
		handleOpenAsarConfirmed()
		return
	}

	g.OpenPopup("#openasar-confirm")
}

func handleErr(di *DiscordInstall, err error, action string) {
	if errors.Is(err, os.ErrPermission) {
		switch runtime.GOOS {
		case "windows":
			err = errors.New("Permission denied. Make sure your Discord is fully closed (from the tray)!")
		case "darwin":
			// FIXME: This text is not selectable which is a bit mehhh
			command := "sudo chown -R \"${USER}:wheel\" " + di.path
			err = errors.New("Permission denied. Please grant the installer Full Disk Access in the system settings (privacy & security page).\n\nIf that also doesn't work, try running the following command in your terminal:\n" + command)
		default:
			err = errors.New("Permission denied. Maybe try running me as Administrator/Root?")
		}
	}

	ShowModal("Failed to "+action+" this Install", err.Error())
}

func onCustomInputChanged() {
	p := customDir
	if len(p) != 0 {
		// Select the custom option for people
		radioIdx = customChoiceIdx
	}

	dir := path.Dir(p)

	isNewDir := strings.HasSuffix(p, "/")
	wentUpADir := !isNewDir && dir != autoCompleteDir

	if isNewDir || wentUpADir {
		autoCompleteDir = dir
		// reset all the funnies
		autoCompleteIdx = 0
		lastAutoComplete = ""
		autoCompleteFile = ""
		autoCompleteCandidates = nil

		// Generate autocomplete items
		files, err := os.ReadDir(dir)
		if err == nil {
			for _, file := range files {
				autoCompleteCandidates = append(autoCompleteCandidates, file.Name())
			}
		}
	} else if !didAutoComplete {
		// reset auto complete and update our file
		autoCompleteFile = path.Base(p)
		lastAutoComplete = ""
	}

	if wentUpADir {
		autoCompleteFile = path.Base(p)
	}

	didAutoComplete = false
}

// go can you give me []any?
// to pass to giu RangeBuilder?
// yeeeeees
// actually returns []string like a boss
func makeAutoComplete() []any {
	input := strings.ToLower(autoCompleteFile)

	var candidates []any
	for _, e := range autoCompleteCandidates {
		file := strings.ToLower(e)
		if autoCompleteFile == "" || strings.HasPrefix(file, input) {
			candidates = append(candidates, e)
		}
	}
	return candidates
}

func makeRadioOnChange(i int) func() {
	return func() {
		radioIdx = i
	}
}

func renderFilesDirErr() g.Widget {
	return g.Layout{
		g.Dummy(0, 50),
		g.Style().
			SetColor(g.StyleColorText, DiscordRed).
			SetFontSize(30).
			To(
				g.Align(g.AlignCenter).To(
					g.Label("Error: Failed to create: "+FilesDirErr.Error()),
					g.Label("Resolve this error, then restart me!"),
				),
			),
	}
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
						&CondWidget{id == "#scuffed-install", func() g.Widget {
							return g.Column(
								g.Dummy(0, 10),
								g.Button("Take me there!").OnClick(func() {
									// this issue only exists on windows so using Windows specific path is oki
									username := os.Getenv("USERNAME")
									programData := os.Getenv("PROGRAMDATA")
									g.OpenURL("file://" + path.Join(programData, username))
								}).Size(200, 30),
							)
						}, nil},
						g.Dummy(0, 20),
						&CondWidget{isOpenAsar,
							func() g.Widget {
								return g.Row(
									g.Button("Accept").
										OnClick(func() {
											acceptedOpenAsar = true
											g.CloseCurrentPopup()
										}).
										Size(100, 30),
									g.Button("Cancel").
										OnClick(func() {
											g.CloseCurrentPopup()
										}).
										Size(100, 30),
								)
							},
							func() g.Widget {
								return g.Button("Ok").
									OnClick(func() {
										g.CloseCurrentPopup()
									}).
									Size(100, 30)
							},
						},
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
	candidates := makeAutoComplete()
	wi, _ := win.GetSize()
	w := float32(wi) - 96

	var currentDiscord *DiscordInstall
	if radioIdx != customChoiceIdx {
		currentDiscord = discords[radioIdx].(*DiscordInstall)
	}
	var isOpenAsar = currentDiscord != nil && currentDiscord.IsOpenAsar()

	layout := g.Layout{
		g.Dummy(0, 20),
		g.Separator(),
		g.Dummy(0, 5),

		g.Style().SetFontSize(30).To(
			g.Label("Please select a mode"),
		),

		&CondWidget{len(discords) == 0, func() g.Widget {
			return g.Label("No Discord installs found. You first need to install Discord.")
		}, nil},

		g.Style().SetFontSize(20).To(
			g.RangeBuilder("Discords", discords, func(i int, v any) g.Widget {
				d := v.(*DiscordInstall)
				//goland:noinspection GoDeprecation
				text := strings.Title(d.branch) + " - " + d.path
				if d.isPatched {
					text += " [PATCHED]"
				}
				return g.RadioButton(text, radioIdx == i).
					OnChange(makeRadioOnChange(i))
			}),

			g.RadioButton("Custom Install Location", radioIdx == customChoiceIdx).
				OnChange(makeRadioOnChange(customChoiceIdx)),
		),

		g.Dummy(0, 5),
		g.Style().
			SetStyle(g.StyleVarFramePadding, 16, 16).
			SetFontSize(20).
			To(
				g.InputText(&customDir).Hint("The custom location").
					Size(w - 16).
					Flags(g.InputTextFlagsCallbackCompletion).
					OnChange(onCustomInputChanged).
					// this library has its own autocomplete but it's broken
					Callback(
						func(data imgui.InputTextCallbackData) int32 {
							if len(candidates) == 0 {
								return 0
							}
							// just wrap around
							if autoCompleteIdx >= len(candidates) {
								autoCompleteIdx = 0
							}

							// used by change handler
							didAutoComplete = true

							start := len(customDir)
							// Delete previous auto complete
							if lastAutoComplete != "" {
								start -= len(lastAutoComplete)
								data.DeleteBytes(start, len(lastAutoComplete))
							} else if autoCompleteFile != "" { // delete partial input
								start -= len(autoCompleteFile)
								data.DeleteBytes(start, len(autoCompleteFile))
							}

							// Insert auto complete
							lastAutoComplete = candidates[autoCompleteIdx].(string)
							data.InsertBytes(start, []byte(lastAutoComplete))
							autoCompleteIdx++

							return 0
						},
					),
			),
		g.RangeBuilder("AutoComplete", candidates, func(i int, v any) g.Widget {
			dir := v.(string)
			return g.Label(dir)
		}),

		g.Dummy(0, 20),

		g.Style().SetFontSize(20).To(
			g.Row(
				g.Style().
					SetColor(g.StyleColorButton, DiscordGreen).
					SetDisabled(GithubError != nil).
					To(
						g.Button("Install").
							OnClick(handlePatch).
							Size((w-40)/4, 50),
						Tooltip("Patch the selected Discord Install"),
					),
				g.Style().
					SetColor(g.StyleColorButton, DiscordBlue).
					SetDisabled(GithubError != nil).
					To(
						g.Button("Reinstall / Repair").
							OnClick(func() {
								if IsDevInstall {
									handlePatch()
								} else {
									err := InstallLatestBuilds()
									if err == nil {
										handlePatch()
									}
								}
							}).
							Size((w-40)/4, 50),
						Tooltip("Reinstall & Update Vencord"),
					),
				g.Style().
					SetColor(g.StyleColorButton, DiscordRed).
					To(
						g.Button("Uninstall").
							OnClick(handleUnpatch).
							Size((w-40)/4, 50),
						Tooltip("Unpatch the selected Discord Install"),
					),
				g.Style().
					SetColor(g.StyleColorButton, Ternary(isOpenAsar, DiscordRed, DiscordGreen)).
					To(
						g.Button(Ternary(isOpenAsar, "Uninstall OpenAsar", Ternary(currentDiscord != nil, "Install OpenAsar", "(Un-)Install OpenAsar"))).
							OnClick(handleOpenAsar).
							Size((w-40)/4, 50),
						Tooltip("Manage OpenAsar"),
					),
			),
		),

		InfoModal("#high-score", "High Score!", "You have set a new high score."),
		InfoModal("#modal"+strconv.Itoa(modalId), modalTitle, modalMessage),
	}

	return layout
}

func loop() {
	g.PushWindowPadding(48, 48)

	g.SingleWindow().
		RegisterKeyboardShortcuts(
			g.WindowShortcut{Key: g.KeyUp, Callback: func() {
				if radioIdx > 0 {
					radioIdx--
				}
			}},
			g.WindowShortcut{Key: g.KeyDown, Callback: func() {
				if radioIdx < customChoiceIdx {
					radioIdx++
				}
			}},
		).
		Layout(
			g.Align(g.AlignCenter).To(
				g.Style().SetFontSize(40).To(
					g.Label("Vencord Installer"),
				),
			),

			g.Dummy(0, 20),
			g.Style().SetFontSize(20).To(
				g.Row(
					g.Label(Ternary(IsDevInstall, "Dev Install: ", "Files will be downloaded to: ")+FilesDir),
					g.Style().
						SetColor(g.StyleColorButton, DiscordBlue).
						SetStyle(g.StyleVarFramePadding, 4, 4).
						To(
							g.Button("Open Directory").OnClick(func() {
								g.OpenURL("file://" + FilesDir)
							}),
						),
				),
				&CondWidget{!IsDevInstall, func() g.Widget {
					return g.Label("To customise this location, set the environment variable 'VENCORD_USER_DATA_DIR' and restart me").Wrapped(true)
				}, nil},
				g.Dummy(0, 10),
				g.Label("Cloud Version: "+InstallerTag+" ("+InstallerGitHash+")", "")),
				g.Label("Local Version: "+InstallerTag+" ("+InstallerGitHash+")"+Ternary(IsInstallerOutdated, " - OUTDATED", "")),
			),

			&CondWidget{
				predicate:  FilesDirErr != nil,
				ifWidget:   renderFilesDirErr,
				elseWidget: renderInstaller,
			},
		)

	g.PopStyle()
}
