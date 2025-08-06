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

	"github.com/blang/semver"

	"github.com/deroproject/derohe/globals"
	"github.com/deroproject/derohe/walletapi"
)

// Constants
const (
	DEFAULT_SIMULATOR_WALLET_PORT    = 30025
	DEFAULT_SIMULATOR_DAEMON_PORT    = 20000
	DEFAULT_TESTNET_WALLET_PORT      = 40403
	DEFAULT_TESTNET_DAEMON_PORT      = 40402
	DEFAULT_TESTNET_WORK_PORT        = 40400
	DEFAULT_WALLET_PORT              = 10103
	DEFAULT_DAEMON_PORT              = 10102
	DEFAULT_WORK_PORT                = 10100
	DEFAULT_LOCAL_TESTNET_DAEMON     = "127.0.0.1:40402"
	DEFAULT_LOCAL_TESTNET_P2P        = "127.0.0.1:40401"
	DEFAULT_LOCAL_TESTNET_WORK       = "0.0.0.0:40400"
	DEFAULT_REMOTE_TESTNET_DAEMON    = "testnetexplorer.dero.io:40402"
	DEFAULT_LOCAL_DAEMON             = "127.0.0.1:10102"
	DEFAULT_LOCAL_P2P                = "127.0.0.1:10101"
	DEFAULT_LOCAL_WORK               = "0.0.0.0:10100"
	DEFAULT_REMOTE_DAEMON            = "node.derofoundation.org:11012"
	DEFAULT_CONFIRMATION_TIMEOUT     = 5
	DEFAULT_DAEMON_RECONNECT_TIMEOUT = 10
	DEFAULT_USERADDR_SHORTEN_LENGTH  = 10
	NETWORK_MAINNET                  = "Mainnet"
	NETWORK_TESTNET                  = "Testnet"
	NETWORK_SIMULATOR                = "Simulator"
)

// Globals
var version = semver.MustParse("0.6.1")
var a fyne.App
var engram Engram
var session Session
var gnomon Gnomon
var msgbox MessageBox
var messages Messages
var status Status
var tx Transfers
var res Res
var colors Colors
var cyberdeck Cyberdeck
var themes Theme
var rpc_client Client
var Connected bool
var nav Navigation
var ui UI

func main() {
	// Initialize application
	a = app.NewWithID("Engram")
	a.Settings().SetTheme(themes.main)

	session.Window = a.NewWindow("Engram")
	session.Window.SetMaster()
	session.Window.SetCloseIntercept(func() {
		if engram.Disk != nil {
			closeWallet()
		}

		session.Window.Close()
		os.Exit(0)
	})
	session.Window.SetPadded(false)
	session.Domain = "app.main.loading"
	session.Window.CenterOnScreen()

	// Load resources
	loadResources()

	a.SetIcon(resourceIconPng)
	session.Window.SetIcon(resourceIconPng)

	// Init colors
	colors.Network = color.RGBA{R: 67, G: 239, B: 67, A: 255}
	colors.Account = color.RGBA{R: 233, G: 228, B: 233, A: 0xff}
	colors.DarkMatter = color.RGBA{21, 23, 30, 255}
	colors.Red = color.RGBA{R: 214, B: 74, G: 70, A: 255}
	colors.DarkGreen = color.RGBA{17, 127, 78, 0xff}
	colors.Green = color.RGBA{19, 202, 105, 0xff}
	colors.Blue = color.RGBA{R: 27, B: 249, G: 127, A: 255}
	colors.Gray = color.RGBA{R: 99, B: 110, G: 99, A: 0xff}
	colors.Yellow = color.RGBA{244, 208, 11, 255}
	colors.Cold = color.RGBA{60, 73, 92, 255}
	colors.Flint = color.RGBA{44, 44, 52, 0xff}

	// Init objects
	status.Canvas = canvas.NewText("", colors.Network)
	status.Network = canvas.NewText("", colors.Network)
	session.BalanceText = canvas.NewText("", colors.Account)
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
	status.Gnomon = canvas.NewCircle(colors.Red)
	status.Gnomon.StrokeColor = colors.Red
	status.Gnomon.StrokeWidth = 0
	status.Gnomon.Refresh()
	status.EPOCH = canvas.NewCircle(colors.Red)
	status.EPOCH.StrokeColor = colors.Red
	status.EPOCH.StrokeWidth = 0
	status.EPOCH.Refresh()

	fmt.Printf("Engram v%s (Beta)\n", version)
	fmt.Printf("Copyright 2023-2024 DERO Foundation. All rights reserved.\n")
	fmt.Printf("OS: %s ARCH: %s GOMAXPROCS: %d\n\n", runtime.GOOS, runtime.GOARCH, runtime.GOMAXPROCS(0))
	fmt.Printf("\"Insist on yourself; never imitate. Your own gift you can present every moment with the \ncumulative force of a whole life's cultivation; but of the adopted talent of another, \nyou have only an extemporaneous, half possession.\"\n\n")

	// Map arguments for DERO network (TODO: Fully support console arguments)
	globals.Arguments = make(map[string]interface{})
	globals.Arguments["--debug"] = false
	globals.Arguments["--testnet"] = false
	globals.Arguments["--daemon-address"] = "127.0.0.1:10102"
	globals.Arguments["--p2p-bind"] = "127.0.0.1:10101"
	globals.Arguments["--rpc-server"] = true
	globals.Arguments["--rpc-bind"] = "127.0.0.1:10103"
	globals.Arguments["--allow-rpc-password-change"] = true
	globals.Arguments["--rpc-login"] = newRPCUsername() + ":" + newRPCPassword()
	globals.Arguments["--offline"] = false
	globals.Arguments["--remote"] = false

	initSettings()
	globals.Initialize()

	session.Domain = "app.main"

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
