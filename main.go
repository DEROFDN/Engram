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
//
// Build win: -ldflags -H=windowsgui

package main

import (
	"fmt"
	"image/color"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/blang/semver"

	"github.com/civilware/Gnomon/indexer"
	"github.com/civilware/Gnomon/storage"

	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/globals"
	"github.com/deroproject/derohe/rpc"
	"github.com/deroproject/derohe/transaction"
	"github.com/deroproject/derohe/walletapi"
	"github.com/deroproject/derohe/walletapi/rpcserver"
)

type App struct {
	App    fyne.App
	Window fyne.Window
	Focus  bool
}

type Colors struct {
	Network    color.Color
	Account    color.Color
	Blue       color.Color
	Red        color.Color
	Green      color.Color
	Gray       color.Color
	Yellow     color.Color
	DarkMatter color.Color
	Cold       color.Color
	Flint      color.Color
}

type Session struct {
	Window          fyne.Window
	DesktopMode     bool
	Domain          string
	Network         bool
	Mode            string
	Language        int
	ID              string
	Link            string
	Type            string
	Daemon          string
	WalletOpen      bool
	Username        string
	Balance         uint64
	BalanceUSD      string
	BalanceText     *canvas.Text
	BalanceUSDText  *canvas.Text
	ModeText        *canvas.Text
	IDText          *canvas.Text
	LinkText        *canvas.Text
	Path            string
	Name            string
	Password        string
	PasswordConfirm string
	DaemonHeight    int64
	WalletHeight    int64
	RPCServer       *rpcserver.RPCServer
	Verified        bool
	Dashboard       string
	Error           string
	Gif             *AnimatedGif
	NewUser         string
}

type Cyberdeck struct {
	active   int
	mode     int
	user     string
	pass     string
	button   *widget.Button
	userText *widget.Entry
	passText *widget.Entry
	toggle   *widget.Button
	progress *widget.ProgressBar
	status   *canvas.Text
	server   *rpcserver.RPCServer
	interval int
	checkbox *widget.Check
}

type Engram struct {
	Disk *walletapi.Wallet_Disk
}

type History struct {
	Window fyne.Window
}

type Gnomon struct {
	Active int
	Index  *indexer.Indexer
	DB     *storage.GravitonStore
	Path   string
}

type Relay struct {
	Text     string
	Duration uint64
	Default  string
}

type Status struct {
	Canvas        *canvas.Text
	Message       string
	Network       *canvas.Text
	Connection    *canvas.Circle
	Sync          *canvas.Circle
	Cyberdeck     *canvas.Circle
	Animation     *fyne.Animation
	Authenticator *widget.ProgressBar
	Daemon        *ImageButton
	Netrunner     *ImageButton
}

type Transfers struct {
	Address    *rpc.Address
	PaymentID  uint64
	Amount     uint64
	Comment    string
	GasStorage uint64
	Fees       uint64
	Pending    []rpc.Transfer
	TX         *transaction.Transaction
	TXID       crypto.Hash
	Proof      string
	Ringsize   uint64
	SendAll    bool
	Size       float32
	Status     string
	OfflineTX  bool
	Filename   string
}

type Messages struct {
	Contact string
	Data    []string
	Box     *widget.List
	List    binding.ExternalStringList
	Height  uint64
	Message string
}

type InstallContract struct {
	TXID string
}

type DaemonRPC struct {
	Jsonrpc string       `json:"jsonrpc"`
	ID      string       `json:"id"`
	Method  string       `json:"method"`
	Params  DaemonParams `json:"params"`
}

type DaemonParams struct {
	Name string `json:"name"`
}

type DaemonCheckRPC struct {
	Jsonrpc string            `json:"jsonrpc"`
	ID      string            `json:"id"`
	Method  string            `json:"method"`
	Params  DaemonCheckParams `json:"params"`
}

type DaemonCheckParams struct {
	Address                 string      `json:"address"`
	SCID                    crypto.Hash `json:"scid"`
	Merkle_Balance_TreeHash string      `json:"treehash,omitempty"`
	TopoHeight              int64       `json:"topoheight,omitempty"`
}

type SmartContractRPC struct {
	Jsonrpc string   `json:"jsonrpc"`
	ID      string   `json:"id"`
	Method  string   `json:"method"`
	Params  ScParams `json:"params"`
}

type ScRPC struct {
	Name     string `json:"name"`
	Datatype string `json:"datatype"`
	Value    string `json:"value"`
}

type ScParams struct {
	Scid     string  `json:"scid"`
	Ringsize int     `json:"ringsize"`
	ScRPC    []ScRPC `json:"sc_rpc"`
}

type Miner struct {
	Window fyne.Window
}

type Daemon struct {
	Window fyne.Window
}

// Constants
const (
	MIN_WIDTH                     = 380
	MIN_HEIGHT                    = 800
	DEFAULT_TESTNET_WALLET_PORT   = 40403
	DEFAULT_TESTNET_DAEMON_PORT   = 40402
	DEFAULT_TESTNET_WORK_PORT     = 40400
	DEFAULT_WALLET_PORT           = 10103
	DEFAULT_DAEMON_PORT           = 10102
	DEFAULT_WORK_PORT             = 10100
	DEFAULT_LOCAL_TESTNET_DAEMON  = "127.0.0.1:40402"
	DEFAULT_LOCAL_TESTNET_P2P     = "127.0.0.1:40401"
	DEFAULT_LOCAL_TESTNET_WORK    = "0.0.0.0:40400"
	DEFAULT_REMOTE_TESTNET_DAEMON = "testnetexplorer.dero.io:40402"
	DEFAULT_LOCAL_DAEMON          = "127.0.0.1:10102"
	DEFAULT_LOCAL_P2P             = "127.0.0.1:10101"
	DEFAULT_LOCAL_WORK            = "0.0.0.0:10100"
	DEFAULT_REMOTE_DAEMON         = "89.38.99.117:10102" // "https://rwallet.dero.live"
	MESSAGE_LIMIT                 = 144
)

// Globals
var version = semver.MustParse("0.1.2")
var appl fyne.App
var engram Engram
var session Session
var gnomon Gnomon
var messages Messages
var history History
var miner Miner
var daemon Daemon
var rs Relay
var status Status
var tx Transfers
var res Res
var colors Colors
var cyberdeck Cyberdeck

// Main application
func main() {
	// Initialize applications
	appl = app.New() // Engram

	t := &eTheme{}
	appl.Settings().SetTheme(t)

	session.Window = appl.NewWindow("Engram")
	session.Window.SetMaster()
	session.Window.SetCloseIntercept(func() {
		if node.Active != 0 {
			stopDaemon()
			fmt.Print("[Engram] Daemon closed.\n")
		}
		if gnomon.Index != nil {
			stopGnomon()
		}
		if engram.Disk != nil {
			closeWallet()
			fmt.Print("[Engram] Wallet closed.\n")
		}
		fmt.Print("[Engram] Grace achieved.\n")
		session.Window.Close()
	})
	session.Window.SetPadded(false)
	session.Domain = "app.main.loading"
	session.Window.CenterOnScreen()

	// Load resources
	loadResources()

	appl.SetIcon(resourceIconPng)
	session.Window.SetIcon(resourceIconPng)

	// Init colors
	colors.Network = color.RGBA{R: 67, G: 239, B: 67, A: 255}
	colors.Account = color.RGBA{R: 233, G: 228, B: 233, A: 0xff}
	colors.DarkMatter = color.RGBA{19, 25, 34, 255}
	colors.Red = color.RGBA{R: 214, B: 74, G: 70, A: 255}
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

	resizeWindow(MIN_WIDTH, MIN_HEIGHT)
	session.Window.SetFixedSize(true)

	// Check if mobile device
	if appl.Driver().Device().IsMobile() {
		session.DesktopMode = false
		//fmt.Printf("[Engram] Generating precompute table... ")
		go walletapi.Initialize_LookupTable(1, 1<<21)
		//fmt.Printf("Done\n")
	} else {
		session.DesktopMode = true
		//fmt.Printf("[Engram] Generating large precompute table... ")
		go walletapi.Initialize_LookupTable(1, 1<<24)
		//fmt.Printf("Done\n")
	}

	fmt.Printf("Welcome to Engram.\n")
	fmt.Printf("Copyright 2020-2022 DERO Foundation. All rights reserved.\n")
	fmt.Printf("OS:%s ARCH:%s GOMAXPROCS:%d\n\n", runtime.GOOS, runtime.GOARCH, runtime.GOMAXPROCS(0))

	// Map arguments for DERO network
	globals.Arguments = make(map[string]interface{})
	globals.Arguments["--debug"] = false
	globals.Arguments["--testnet"] = false
	globals.Arguments["--rpc-server"] = true
	globals.Arguments["--rpc-bind"] = "127.0.0.1:10103"
	globals.Arguments["--allow-rpc-password-change"] = true
	globals.Arguments["--rpc-login"] = newRPCUsername() + ":" + newRPCPassword()
	globals.Arguments["--offline"] = false
	globals.Arguments["--rpc-bind"] = "127.0.0.1:10102"
	globals.Arguments["--p2p-bind"] = "127.0.0.1:10101"
	globals.Arguments["--getwork-bind"] = "127.0.0.1:10100"
	globals.Arguments["--fastsync"] = true
	//globals.Init_rlog()
	initSettings()
	globals.Initialize()

	go walletapi.Keep_Connectivity()

	if session.DesktopMode {
		go loading()
		session.Window.SetContent(layoutLoading())

		if desktop, ok := appl.(desktop.App); ok {
			menu := fyne.NewMenu("Engram",
				fyne.NewMenuItem("Daemon", func() {
					// show
				}),
				fyne.NewMenuItem("Netrunner", func() {
					if engram.Disk == nil {
						return
					}
					if nr.Mission == 1 || miner.Window != nil {
						miner.Window.Show()
						miner.Window.RequestFocus()
						return
					}
					miner.Window = appl.NewWindow("Netrunner")
					miner.Window.SetPadded(false)
					miner.Window.CenterOnScreen()
					miner.Window.SetIcon(resourceMinerOnPng)
					miner.Window.Resize(fyne.NewSize(MIN_WIDTH, MIN_HEIGHT))
					miner.Window.SetFixedSize(true)
					miner.Window.SetCloseIntercept(func() {
						if nr.Mission == 1 {
							nr.Mission = 0
							nr.Connection.UnderlyingConn().Close()
							nr.Connection.Close()
							fmt.Printf("[Netrunner] Shutdown initiated.\n")
							status.Netrunner.Res = resourceMinerOffPng
							status.Netrunner.Refresh()
						}
						miner.Window.Close()
						miner.Window = nil
					})
					status.Netrunner.Res = resourceMinerOnPng
					status.Netrunner.Refresh()
					miner.Window.SetContent(layoutNetrunner())
					miner.Window.Show()
				}),
				fyne.NewMenuItem("Settings", func() {
					// show
				}),
			)

			desktop.SetSystemTrayMenu(menu)

			_, _ = getGnomon()

		}
	} else {
		session.Domain = "app.main"
		session.Window.SetContent(layoutMain())
	}

	session.Window.ShowAndRun()
}
