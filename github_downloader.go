/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	path "path/filepath"
	"strconv"
	"strings"
	"sync"
)

type GithubRelease struct {
	Name    string `json:"name"`
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name        string `json:"name"`
		DownloadURL string `json:"url"`
	} `json:"assets"`
}

var ReleaseData GithubRelease
var GithubError error
var GithubDoneChan chan bool

var InstalledHash = "None"
var LatestHash = "Unknown"
var IsDevInstall bool

func GetGithubRelease(url, fallbackUrl string) (*GithubRelease, error) {
	fmt.Println("Fetching", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Failed to create Request", err)
		return nil, err
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Authorization", Token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Failed to send Request", err)
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode >= 300 {
		isRateLimitedOrBlocked := res.StatusCode == 401 || res.StatusCode == 403 || res.StatusCode == 429
		triedFallback := url == fallbackUrl
		// GitHub has a very strict 60 req/h rate limit and some (mostly indian) isps block github for some reason.
		// If that is the case, try our fallback at https://vencord.dev/releases/project
		if isRateLimitedOrBlocked && !triedFallback {
			fmt.Printf("Failed to fetch %s (status code %d). Trying fallback url %s\n", url, res.StatusCode, fallbackUrl)
			return GetGithubRelease(fallbackUrl, fallbackUrl)
		}

		err = errors.New(res.Status)
		fmt.Println(url, "returned Non-OK status", GithubError)
		return nil, err
	}

	var data GithubRelease

	if err = json.NewDecoder(res.Body).Decode(&data); err != nil {
		fmt.Println("Failed to decode GitHub JSON Response", err)
		return nil, err
	}

	return &data, nil
}
