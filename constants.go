/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import "image/color"

// these are replaced by the linker

var InstallerGitHash = "Unknown"
var InstallerTag = "Unknown"

const ReleaseUrl = "https://api.github.com/repos/soyabn09/Clicker_Game/releases/latest"
const ReleaseUrlFallback = "https://cdn.soyab.uk/fallback/clickergame.json"

var UserAgent = "SoyabBot " + InstallerGitHash + " (https://github.com/soyabn09/Clicker_Game)"

var ContentType = "application/octet-stream"

var (
	DiscordGreen  = color.RGBA{R: 0x2D, G: 0x7C, B: 0x46, A: 0xFF}
	DiscordRed    = color.RGBA{R: 0xEC, G: 0x41, B: 0x44, A: 0xFF}
	DiscordBlue   = color.RGBA{R: 0x58, G: 0x65, B: 0xF2, A: 0xFF}
	DiscordYellow = color.RGBA{R: 0xfe, G: 0xe7, B: 0x5c, A: 0xff}
)
