/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"fmt"
	"runtime"
)

var IsInstallerOutdated = false

func CheckSelfUpdate() {
	fmt.Println("Checking for Installer Updates...")

	res, err := GetGithubRelease(ReleaseUrl, ReleaseUrlFallback)
	if err == nil {
		IsInstallerOutdated = res.TagName != InstallerTag
	}
}

func GetInstallerDownloadLink() string {
	switch runtime.GOOS {
	case "windows":
		return "https://github.com/soyabn09/Clicker_Game/releases/latest/download/ClickerGame.exe"
	case "darwin":
		return "https://github.com/soyabn09/Clicker_Game/releases/latest/download/ClickerGame.MacOS.zip"
	default:
		return ""
	}
}

func GetInstallerDownloadMarkdown() string {
	link := GetInstallerDownloadLink()
	if link == "" {
		return ""
	}
	return " [Download the latest version](" + link + ")"
}
