// Copyright 2023-2024 DERO Foundation. All rights reserved.
// Use of this source code in any form is governed by RESEARCH license.
// license can be found in the LICENSE file.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY
// EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL
// THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
// PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT,
// STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF
// THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package main

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	x "fyne.io/x/fyne/widget"
	"github.com/civilware/tela/logger"
)

type Res struct {
	bg          *canvas.Image
	bg2         *canvas.Image
	icon        *canvas.Image
	load        *canvas.Image
	loading     *x.AnimatedGif
	header      *canvas.Image
	dero        *canvas.Image
	gram        *canvas.Image
	block       *canvas.Image
	red_alert   *canvas.Image
	green_alert *canvas.Image
	mainBg      *canvas.Image
}

// Get app path
func AppPath() (result string) {
	result, _ = os.Getwd()
	if runtime.GOOS == "android" {
		result = a.Storage().RootURI().Path()
	} else if runtime.GOOS == "ios" {
		result = a.Storage().RootURI().Path()
	}

	return
}

func GetAccounts() (result []string, err error) {
	path := ""

	switch session.Network {
	case NETWORK_MAINNET:
		_, err = os.Stat(filepath.Join(AppPath(), "mainnet"))
		if err != nil {
			return
		} else {
			path = filepath.Join(AppPath(), "mainnet") + string(filepath.Separator)
		}
	case NETWORK_SIMULATOR:
		_, err = os.Stat(filepath.Join(AppPath(), "testnet_simulator"))
		if err != nil {
			return
		} else {
			path = filepath.Join(AppPath(), "testnet_simulator") + string(filepath.Separator)
		}
	default:
		_, err = os.Stat(filepath.Join(AppPath(), "testnet"))
		if err != nil {
			return
		} else {
			path = filepath.Join(AppPath(), "testnet") + string(filepath.Separator)
		}
	}

	matches, _ := filepath.Glob(path + "*.db")
	result = []string{}

	for _, match := range matches {
		check, _ := os.Stat(match)
		if !check.IsDir() {
			if strings.Contains(match, ".db") {
				split := strings.Split(match, string(filepath.Separator))
				pos := len(split) - 1
				match = split[pos]
				result = append(result, match)
			}
		}
	}

	/*
		if len(result) == 0 {
			// TODO: May do something here like start the user at Create/Restore Account window.
		}
	*/

	return
}

func findAccount() (result bool) {
	matches, err := filepath.Glob(session.Path)
	if err != nil {
		logger.Errorf("[Engram] findAccount: %s\n", err)
	}

	if len(matches) > 0 {
		result = true
	} else {
		result = false
	}

	return
}

func checkDir() (err error) {
	err = os.MkdirAll(filepath.Join(AppPath(), "mainnet"), os.ModePerm)
	if err != nil {
		return
	}
	err = os.MkdirAll(filepath.Join(AppPath(), "testnet"), os.ModePerm)
	if err != nil {
		return
	}
	err = os.MkdirAll(filepath.Join(AppPath(), "testnet_simulator"), os.ModePerm)
	if err != nil {
		return
	}
	err = os.MkdirAll(filepath.Join(AppPath(), "datashards"), os.ModePerm)
	if err != nil {
		return
	}

	return
}

// Write the content to uri
func writeToURI(content []byte, uri fyne.URIWriteCloser) (n int, err error) {
	defer uri.Close()

	return uri.Write(content)
}

// Read the content from uri
func readFromURI(uri fyne.URIReadCloser) ([]byte, error) {
	defer uri.Close()

	return io.ReadAll(uri)
}
