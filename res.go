// Copyright 2021-2022 DERO Foundation. All rights reserved.
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
	//"image/color"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"

	"github.com/kbinani/screenshot"
)

type Res struct {
	bg           *canvas.Image
	bg2          *canvas.Image
	bg3          *canvas.Image
	bg300        *canvas.Image
	icon         *canvas.Image
	icon_sm      *canvas.Image
	load         *canvas.Image
	header       *canvas.Image
	spacer       *canvas.Image
	dero         *canvas.Image
	enter        *canvas.Image
	gram         *canvas.Image
	gram_footer  *canvas.Image
	login_footer *canvas.Image
	rpc_header   *canvas.Image
	nr_header    *canvas.Image
	rpc_footer   *canvas.Image
	nr_footer    *canvas.Image
	nft_header   *canvas.Image
	nft_footer   *canvas.Image
	home_header  *canvas.Image
	home_footer  *canvas.Image
}

// Get app path
func AppPath() string {
	app, _ := os.Executable()
	path := filepath.Dir(app)

	return path
}

func LoadAsset(p string) fyne.Resource {
	a := AppPath()
	r, _ := fyne.LoadResourceFromPath(a + p)

	return r
}

func GetAccounts() (result []string) {
	path := ""
	if !session.Network {
		path = AppPath() + string(filepath.Separator) + "testnet" + string(filepath.Separator)
	} else {
		path = AppPath() + string(filepath.Separator) + "mainnet" + string(filepath.Separator)
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

	if len(result) == 0 {
		// TODO: May do something here like start the user at Create/Restore Account window.
	}

	return
}

func findAccount() (result bool) {
	/*
		path := AppPath() + string(filepath.Separator)
		folder := ""
		if !session.Network {
			folder = "testnet"
		} else {
			folder = "mainnet"
		}
	*/

	matches, err := filepath.Glob(session.Path)
	if err != nil {
		fmt.Printf("%s\n", err)
	}

	if len(matches) > 0 {
		result = true
	} else {
		result = false
	}

	return
}

func checkDir() (err error) {
	_, err = os.Stat(AppPath() + string(filepath.Separator) + "testnet")
	if os.IsNotExist(err) {
		err = os.MkdirAll("testnet", 0755)
		if err != nil {
			panic(err)
		}
	}

	_, err = os.Stat(AppPath() + string(filepath.Separator) + "mainnet")
	if os.IsNotExist(err) {
		err = os.MkdirAll("mainnet", 0755)
		if err != nil {
			panic(err)
		}
	}

	return
}

func GetResolution() (x float32, y float32) {
	r := screenshot.GetDisplayBounds(0)

	x = float32(r.Dx())
	y = float32(r.Dy())

	return
}
