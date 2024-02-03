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
	"fmt"
	"image/color"
	"os"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/mobile"

	"github.com/deroproject/derohe/globals"
	"github.com/deroproject/derohe/walletapi"
)

func main() {
	// Initialize application
	a = app.NewWithID(appID) // Engram
	a.Settings().SetTheme(themes.main)

	session.Window = a.NewWindow(appName)
	session.Window.SetMaster()
	session.Window.SetCloseIntercept(func() {
		if engram.Disk != nil {
			closeWallet()
		}

		session.Window.Close()
		os.Exit(0)
	})
	session.Window.SetPadded(false)
	session.Domain = appMainLoading
	session.Window.CenterOnScreen()

	// Load resources
	loadResources()

	a.SetIcon(resourceIconPng)
	session.Window.SetIcon(resourceIconPng)

	// Init colors
	colors.Network = color.RGBA{
		R: 67,
		G: 239,
		B: 67,
		A: 255,
	}
	colors.Account = color.RGBA{
		R: 233,
		G: 228,
		B: 233,
		A: 0xff,
	}
	colors.DarkMatter = color.RGBA{
		21,
		23,
		30,
		255,
	}
	colors.Red = color.RGBA{
		R: 214,
		B: 74,
		G: 70,
		A: 255,
	}
	colors.DarkGreen = color.RGBA{
		17,
		127,
		78,
		0xff,
	}
	colors.Green = color.RGBA{
		19,
		202,
		105,
		0xff,
	}
	colors.Blue = color.RGBA{
		R: 27,
		B: 249,
		G: 127,
		A: 255,
	}
	colors.Gray = color.RGBA{
		R: 99,
		B: 110,
		G: 99,
		A: 0xff,
	}
	colors.Yellow = color.RGBA{
		244,
		208,
		11,
		255,
	}
	colors.Cold = color.RGBA{
		60,
		73,
		92,
		255,
	}
	colors.Flint = color.RGBA{
		44,
		44,
		52,
		0xff,
	}

	// Init objects
	status.Canvas = canvas.NewText(
		string_,
		colors.Network,
	)
	status.Network = canvas.NewText(
		string_,
		colors.Network,
	)
	session.BalanceText = canvas.NewText(
		string_,
		colors.Account,
	)
	status.Connection = canvas.NewCircle(colors.Red)
	status.Connection.StrokeColor = colors.Red
	status.Connection.StrokeWidth = 0
	status.Connection.Refresh()
	status.Sync = canvas.NewCircle(colors.Red)
	status.Sync.StrokeColor = colors.Red
	status.Sync.StrokeWidth = 0
	status.Sync.Refresh()
	status.Cyberdeck = canvas.NewCircle(colors.Red)
	status.Cyberdeck.StrokeColor = colors.Red
	status.Cyberdeck.StrokeWidth = 0
	status.Cyberdeck.Refresh()

	fmt.Printf(
		engramBeta,
		version,
	)
	fmt.Printf(copyrightNotice)
	fmt.Printf(
		osArchGoMax,
		runtime.GOOS,
		runtime.GOARCH,
		runtime.GOMAXPROCS(0),
	)
	fmt.Printf(quote)

	// Map arguments for DERO network (TODO: Fully support console arguments)
	globals.Arguments = make(map[string]interface{})
	globals.Arguments[stringFlagdebug] = false
	globals.Arguments[stringFlagtestnet] = false
	globals.Arguments[stringFlagdaemonaddress] = DEFAULT_LOCAL_DAEMON
	globals.Arguments[stringFlagp2pbind] = DEFAULT_LOCAL_P2P
	globals.Arguments[stringFlagrpcserver] = true
	globals.Arguments[stringFlagrpcbind] = DEFAULT_LOCAL_WALLET_RPC
	globals.Arguments[stringFlagallowrpcpasschange] = true
	globals.Arguments[stringFlagrpclogin] = newRPCUsername() + singlecolon + newRPCPassword()
	globals.Arguments[stringFlageoffline] = false
	globals.Arguments[stringFlagremote] = false

	initSettings()
	globals.Initialize()

	session.Domain = appMain

	// Intercept mobile back button event
	session.Window.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		if ev.Name == mobile.KeyBack {
			if session.LastDomain != nil {
				session.Window.SetContent(layoutTransition())
				session.Window.SetContent(session.LastDomain)
			} else {
				if engram.Disk != nil {
					session.LastDomain = layoutDashboard()
					session.LastDomain = session.Window.Content()
					session.Window.SetContent(layoutTransition())
					session.Window.SetContent(layoutDashboard())
				} else {
					session.LastDomain = layoutMain()
					session.LastDomain = session.Window.Content()
					session.Window.SetContent(layoutTransition())
					session.Window.SetContent(layoutMain())
				}
			}
		}
	})

	// Check if mobile device
	if a.Driver().Device().IsMobile() {
		go walletapi.Initialize_LookupTable(1, 1<<21)

		ui.MaxWidth = 3600
		ui.MaxHeight = 6800

		ui.Width = ui.MaxWidth * 0.9
		ui.Height = ui.MaxHeight
		ui.Padding = ui.MaxWidth * 0.05

		resizeWindow(ui.MaxWidth, ui.MaxHeight)
		session.Window.SetContent(layoutFrame())
		session.Window.SetFixedSize(true)

		session.Window.ShowAndRun()
	} else {
		go walletapi.Initialize_LookupTable(1, 1<<24)
		ui.MaxWidth = 360
		ui.MaxHeight = 680

		ui.Width = ui.MaxWidth * 0.9
		ui.Height = ui.MaxHeight
		ui.Padding = ui.MaxWidth * 0.05

		resizeWindow(ui.MaxWidth, ui.MaxHeight)
		session.Window.SetContent(layoutMain())
		session.Window.SetFixedSize(true)

		session.Window.ShowAndRun()
	}
}
