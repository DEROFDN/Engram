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
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"github.com/civilware/Gnomon/indexer"
	"github.com/civilware/Gnomon/storage"
	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/globals"
	"github.com/deroproject/derohe/rpc"
	"github.com/deroproject/derohe/transaction"
	"github.com/deroproject/derohe/walletapi"
	"github.com/deroproject/derohe/walletapi/mnemonics"
	"github.com/deroproject/derohe/walletapi/rpcserver"
)

func initSettings() {
	getNetwork()
	getMode()
	getType()
	getDaemon()
	getGnomon()
	getAuthMode()
}

// Get network setting from Graviton
func getNetwork() {
	result, err := GetValue("settings", []byte("network"))
	if err != nil {
		session.Network = true
		globals.Arguments["--testnet"] = false
		setNetwork(true)
	} else {
		if string(result) == "Testnet" {
			session.Network = false
			globals.Arguments["--testnet"] = true
		} else {
			session.Network = true
			globals.Arguments["--testnet"] = false
		}
	}
}

func setNetwork(b bool) (err error) {
	s := ""
	if b {
		s = "Mainnet"
		globals.Arguments["--testnet"] = false
	} else {
		s = "Testnet"
		globals.Arguments["--testnet"] = true
	}

	StoreValue("settings", []byte("network"), []byte(s))

	return
}

// Get daemon endpoint setting from Graviton
func getDaemon() (r string) {
	result, err := GetValue("settings", []byte("endpoint"))
	if err != nil {
		r = DEFAULT_REMOTE_DAEMON
		setDaemon(r)
		session.Daemon = r
		globals.Arguments["--daemon-address"] = r

		return
	}

	r = string(result)
	session.Daemon = r
	globals.Arguments["--daemon-address"] = r

	return
}

func setDaemon(s string) (err error) {
	StoreValue("settings", []byte("endpoint"), []byte(s))
	globals.Arguments["--daemon-address"] = s
	session.Daemon = s
	if gnomon.Index != nil {
		gnomon.Index.Endpoint = s
	}
	return
}

// Get mode (online, offline) setting from Graviton
func getMode() {
	session.Mode = "Online"
	globals.Arguments["--offline"] = false

	/*
		if globals.Arguments["--offline"].(bool) == true {
			session.Mode = "Offline"
			return
		}

		s := "mode"
		t := "settings"
		key := []byte(s)
		result, err := GetValue(t, key)
		if err != nil {
			session.Mode = "Online"
			err := setMode("Online")
			globals.Arguments["--offline"] = false
			if err != nil {
				fmt.Printf("[Engram] Error: %s\n", err)
				return
			}
		} else {
			if result == nil {
				session.Mode = "Online"
				err := setMode("Online")
				globals.Arguments["--offline"] = false
				if err != nil {
					fmt.Printf("[Engram] Error: %s\n", err)
					return
				}
			} else {
				if string(result) == "Offline" {
					globals.Arguments["--offline"] = true
					session.Mode = "Offline"
				} else {
					globals.Arguments["--offline"] = false
					session.Mode = "Online"
				}
			}
		}
	*/
}

func setMode(s string) (err error) {
	err = StoreValue("settings", []byte("mode"), []byte(s))
	if s == "Offline" {
		globals.Arguments["--offline"] = true
	} else {
		globals.Arguments["--offline"] = false
	}
	return
}

// Get connection type (local, remote) setting from Graviton
func getType() (r string) {
	result, err := GetValue("settings", []byte("node_type"))
	if err != nil {
		r = "Remote"
		setType(r)
		session.Type = r
		return
	}

	if result == nil {
		r = "Remote"
		setType(r)
		session.Type = r
	} else {
		r = string(result)
		session.Type = r
	}
	return
}

func setType(s string) (err error) {
	err = StoreValue("settings", []byte("node_type"), []byte(s))
	return
}

func getGnomon() (r string, err error) {
	v, err := GetValue("settings", []byte("gnomon"))
	if err != nil {
		gnomon.Active = 1
		if gnomon.Index != nil {
			gnomon.Index.Endpoint = getDaemon()
		}
		StoreValue("settings", []byte("gnomon"), []byte("1"))
	}

	if string(v) == "1" {
		gnomon.Active = 1
		if gnomon.Index != nil {
			gnomon.Index.Endpoint = getDaemon()
		}
	} else {
		gnomon.Active = 0
	}

	r = string(v)
	return
}

func setGnomon(s string) (err error) {
	if s == "1" {
		err = StoreValue("settings", []byte("gnomon"), []byte("1"))
		gnomon.Active = 1
		if gnomon.Index != nil {
			gnomon.Index.Endpoint = getDaemon()
		}
	} else {
		err = StoreValue("settings", []byte("gnomon"), []byte("0"))
		gnomon.Active = 0
	}
	return
}

func getAuthMode() (result string, err error) {
	r, err := GetValue("settings", []byte("auth_mode"))
	if err != nil {
		StoreValue("settings", []byte("auth_mode"), []byte("true"))
		cyberdeck.mode = 1
		result = "true"
	} else {
		result = string(r)
		if string(result) == "true" {
			cyberdeck.mode = 1
			result = "true"
		} else {
			cyberdeck.mode = 0
			result = "false"
		}
	}
	return
}

func setAuthMode(s string) {
	if s == "true" {
		StoreValue("settings", []byte("auth_mode"), []byte("true"))
	} else {
		StoreValue("settings", []byte("auth_mode"), []byte("false"))
	}
}

// Pulse
func Pulse() {

}

// Set window size
func resizeWindow(width float32, height float32) {
	s := fyne.NewSize(width, height)
	session.Window.Resize(s)
}

func loading() {
	if session.Domain == "app.main.loading" {
		time.Sleep(time.Second * 6)
		session.Domain = "app.main"
		resizeWindow(MIN_WIDTH, MIN_HEIGHT)
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutMain())
		return
	}
}

// Close the wallet
func closeWallet() {
	if cyberdeck.mode == 1 && cyberdeck.server != nil {
		cyberdeck.server.RPCServer_Stop()
		cyberdeck.server = nil
	}

	if history.Window != nil {
		history.Window.Close()
		history.Window = nil
	}

	if engram.Disk != nil {
		if nr.Mission == 1 || miner.Window != nil {
			nr.Mission = 0
			if nr.Connection != nil {
				nr.Connection.Close()
			}
			miner.Window.Close()
			nr = Netrunner{}
		}

		engram.Disk.Save_Wallet()

		globals.Exit_In_Progress = true
		engram.Disk.Close_Encrypted_Wallet()
		session.WalletOpen = false
		session.Domain = "app.main"
		engram.Disk = nil
		session.Path = ""
		session.Name = ""
		tx = Transfers{}

		resizeWindow(MIN_WIDTH, MIN_HEIGHT)
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutMain())
		fmt.Printf("[Engram] Wallet closed.\n")
		return
	}
}

func (session *Session) Clear() {
	session = &Session{}
}

func create() (a string, s string, err error) {
	check := findAccount()

	if session.Path == "" {
		session.Error = "Please enter an account name."
	} else if session.Language == -1 {
		session.Error = "Please select a language."
	} else if session.Password == "" {
		session.Error = "Please enter a password."
	} else if session.PasswordConfirm == "" {
		session.Error = "Please confirm your password."
	} else if session.PasswordConfirm != session.Password {
		session.Error = "Passwords do not match."
	} else if check {
		session.Error = "Account name already exists."
	} else {
		engram.Disk, err = walletapi.Create_Encrypted_Wallet_Random(session.Path, session.Password)

		if err != nil {
			session.Language = -1
			session.Name = ""
			session.Path = ""
			session.Password = ""
			session.PasswordConfirm = ""
			session.Error = "Account could not be created."
		} else {
			if !session.Network {
				engram.Disk.SetNetwork(false)
				globals.Arguments["--testnet"] = true
			} else {
				engram.Disk.SetNetwork(true)
				globals.Arguments["--testnet"] = false
			}

			languages := mnemonics.Language_List()

			if session.Language < 0 || session.Language > len(languages)-1 {
				session.Language = 0 // English
			}

			engram.Disk.SetSeedLanguage(languages[session.Language])
			a = engram.Disk.GetAddress().String()
			s = engram.Disk.GetSeed()
			engram.Disk.Close_Encrypted_Wallet()
			engram.Disk = nil
			session.Error = "Account successfully created."
			session.Language = -1
			session.Name = ""
			session.Path = ""
			session.Password = ""
			session.PasswordConfirm = ""
			session.Domain = "app.main"
		}
	}
	return
}

func login() {
	var err error
	var temp *walletapi.Wallet_Disk

	initSettings()

	if engram.Disk == nil {
		temp, err = walletapi.Open_Encrypted_Wallet(session.Path, session.Password)
		if err != nil {
			engram.Disk = nil
			temp = nil
			session.Domain = "app.main"
			session.Error = "Invalid password."
			session.Window.Canvas().Content().Refresh()

			return
		} else {
			engram.Disk = temp
			temp = nil
		}
	}

	session.Domain = "app.wallet"
	session.WalletOpen = true
	session.Password = ""

	if !session.Network {
		engram.Disk.SetNetwork(false)
		globals.Arguments["--testnet"] = true
	} else {
		engram.Disk.SetNetwork(true)
		globals.Arguments["--testnet"] = false
	}

	setRingSize(engram.Disk, 8)
	session.Verified = false

	if session.Mode == "Offline" {
		// Offline mode
		engram.Disk.SetOfflineMode()
		globals.Arguments["--offline"] = true
	} else {
		// Online mode
		globals.Arguments["--offline"] = false
		status.Connection.FillColor = colors.Yellow
		status.Connection.Refresh()
		engram.Disk.SetDaemonAddress(session.Daemon)
		engram.Disk.SetOnlineMode()
		session.Balance = 0

		for !walletapi.IsDaemonOnline() {
			time.Sleep(time.Second / 2)
		}

		if engram.Disk.IsRegistered() {
			resizeWindow(MIN_WIDTH, MIN_HEIGHT)
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutDashboard())
		} else {
			retry := 0
			for !engram.Disk.IsRegistered() {
				if engram.Disk.Get_Registration_TopoHeight() == -1 && retry > 5 {
					registerAccount()
					session.Verified = true
					fmt.Printf("[Registration] Account registration PoW started...\n")
					fmt.Printf("[Registration] Registering your account. This can take up to 120 minutes (one time). Please wait...\n")
					break
				} else {
					time.Sleep(time.Second)
					retry++
				}
			}
		}

		go func() {
			timer := 0

			for engram.Disk != nil && walletapi.IsDaemonOnline() {
				engram.Disk.SetOnlineMode()

				if !engram.Disk.IsRegistered() {
					if engram.Disk.Get_Daemon_TopoHeight() == 0 {
						fmt.Printf("[Network] Could not connect to daemon...%d\n", engram.Disk.Get_Daemon_TopoHeight())
						status.Sync.FillColor = colors.Red
						status.Sync.Refresh()
						time.Sleep(time.Second)
					} else {
						time.Sleep(time.Second)
					}
				} else {
					newBalance, _ := engram.Disk.Get_Balance()

					if timer >= 60 {
						go getPrice()
					}

					session.WalletHeight = engram.Disk.Get_TopoHeight()
					session.DaemonHeight = engram.Disk.Get_Daemon_TopoHeight()

					if session.Mode == "Online" {
						if session.WalletHeight == session.DaemonHeight && session.DaemonHeight > 0 {
							if session.Type != "Local" {
								status.Connection.FillColor = colors.Green
								status.Connection.Refresh()
							}
							status.Sync.FillColor = colors.Green
							status.Sync.Refresh()
							fmt.Printf("[Network] Connected to "+session.Daemon+" › Sync Complete (%s / %s)\n", strconv.FormatInt(session.WalletHeight, 10), strconv.FormatInt(session.DaemonHeight, 10))
						} else if session.DaemonHeight == 0 {
							status.Sync.FillColor = colors.Red
							status.Sync.Refresh()
							fmt.Printf("[Network] Connected to " + session.Daemon + " › Syncing... (" + strconv.FormatInt(session.WalletHeight, 10) + " / " + strconv.FormatInt(session.DaemonHeight, 10) + ")\n")
						} else {
							status.Sync.FillColor = colors.Yellow
							status.Sync.Refresh()
							fmt.Printf("[Network] Connected to " + session.Daemon + " › Syncing... (" + strconv.FormatInt(session.WalletHeight, 10) + " / " + strconv.FormatInt(session.DaemonHeight, 10) + ")\n")
						}
					} else {
						status.Sync.FillColor = colors.Gray
						status.Sync.Refresh()
						status.Cyberdeck.FillColor = colors.Gray
						status.Cyberdeck.Refresh()
						fmt.Printf("[Network] Offline › Last Height: " + strconv.FormatInt(session.WalletHeight, 10) + " / " + strconv.FormatInt(session.DaemonHeight, 10) + "\n")
					}

					session.Balance = newBalance
					session.BalanceText.Text = globals.FormatMoney(session.Balance)
					session.BalanceText.Refresh()

					address := engram.Disk.GetAddress().String()
					shard := fmt.Sprintf("%x", sha1.Sum([]byte(address)))
					session.ID = shard

					if node.Active == 1 {
						status.Daemon.Res = resourceDaemonOnPng
						status.Daemon.Refresh()
					} else {
						status.Daemon.Res = resourceDaemonOffPng
						status.Daemon.Refresh()
					}

					if nr.Mission == 1 {
						status.Netrunner.Res = resourceMinerOnPng
						status.Netrunner.Refresh()
					} else {
						status.Netrunner.Res = resourceMinerOffPng
						status.Netrunner.Refresh()
					}

					time.Sleep(time.Second)
				}
			}
		}()
	}
}

func loadResources() {
	res.bg = canvas.NewImageFromResource(resourceBgPng)
	res.bg.FillMode = canvas.ImageFillContain

	res.bg2 = canvas.NewImageFromResource(resourceBg2Png)
	res.bg2.FillMode = canvas.ImageFillContain

	res.bg3 = canvas.NewImageFromResource(resourceBg3Png)
	res.bg3.FillMode = canvas.ImageFillContain

	res.icon = canvas.NewImageFromResource(resourceIconPng)
	res.icon.FillMode = canvas.ImageFillContain

	res.header = canvas.NewImageFromResource(resourceBackground1Png)
	res.header.FillMode = canvas.ImageFillContain

	res.load = canvas.NewImageFromResource(resourceLoadPng)
	res.load.FillMode = canvas.ImageFillContain

	res.dero = canvas.NewImageFromResource(resourceDeroPng)
	res.dero.FillMode = canvas.ImageFillContain

	res.enter = canvas.NewImageFromResource(resourceEnterPng)
	res.enter.FillMode = canvas.ImageFillContain

	res.gram = canvas.NewImageFromResource(resourceGramPng)
	res.gram.FillMode = canvas.ImageFillContain

	res.gram_footer = canvas.NewImageFromResource(resourceGramfooterPng)
	res.gram_footer.FillMode = canvas.ImageFillContain

	res.login_footer = canvas.NewImageFromResource(resourceLoginfooterPng)
	res.login_footer.FillMode = canvas.ImageFillContain

	res.rpc_header = canvas.NewImageFromResource(resourceSubHeaderPng)
	res.rpc_header.FillMode = canvas.ImageFillContain

	res.nr_header = canvas.NewImageFromResource(resourceSubHeaderPng)
	res.nr_header.FillMode = canvas.ImageFillContain

	res.rpc_footer = canvas.NewImageFromResource(resourceRpcFooterPng)
	res.rpc_footer.FillMode = canvas.ImageFillContain

	res.nr_footer = canvas.NewImageFromResource(resourceRpcFooterPng)
	res.nr_footer.FillMode = canvas.ImageFillContain

	res.nft_header = canvas.NewImageFromResource(resourceSubHeaderPng)
	res.nft_header.FillMode = canvas.ImageFillContain

	res.nft_footer = canvas.NewImageFromResource(resourceRpcFooterPng)
	res.nft_footer.FillMode = canvas.ImageFillContain

	res.home_header = canvas.NewImageFromResource(resourceSubHeaderPng)
	res.home_header.FillMode = canvas.ImageFillContain

	res.home_footer = canvas.NewImageFromResource(resourceRpcFooterPng)
	res.home_footer.FillMode = canvas.ImageFillContain
}

func newRtnEntry() *returnEntry {
	entry := &returnEntry{}
	entry.ExtendBaseWidget(entry)
	return entry
}

func (e *returnEntry) TypedKey(key *fyne.KeyEvent) {
	switch key.Name {
	case fyne.KeyReturn:
		if session.Domain == "app.main" {
			login()
		} else if session.Domain == "app.create" {
			create()
		}
	default:
		e.Entry.TypedKey(key)
	}
}

func (e *returnEntry) onReturn() {
	login()
}

func newTapRect() *tapRect {
	rect := &tapRect{}
	rect.Rectangle.FillColor = colors.DarkMatter
	return rect
}

/*
func (e *tapRect) CreateRenderer() fyne.WidgetRenderer {
	r := &tapRectRenderer{
		e,
	}
	return r
}
func (r *tapRectRenderer) Destroy()                    {}
func (r *tapRectRenderer) Layout(size fyne.Size) {

}
func (r *tapRectRenderer) MinSize() fyne.Size {
	return r.e.Content.MinSize()
}
*/

func (e *tapRect) Refresh() {
	e.Rectangle.Refresh()
}

func (e *tapRect) SetFillColor(c color.Color) {
	e.Rectangle.FillColor = c
	e.Rectangle.Refresh()
}

func (e *tapRect) Tapped(*fyne.PointEvent) {
	fmt.Printf("tapped")
	/*
		switch key.Name {
		case fyne.KeyReturn:
			if session.Domain == "app.main" {
				login()
			} else if session.Domain == "app.create" {
				create()
			}
		default:
			e.Entry.TypedKey(key)
		}
	*/
}

func (e *tapRect) TappedSecondary(*fyne.PointEvent) {
	fmt.Printf("tapped secondary")
}

// Reset UI resources
func resetResources() {
	res = Res{}
	loadResources()
}

// Get SC balance by SCID
func get_balance_SCID(s string) string {
	b := ""

	if s == "" {
		b = "Balance:  " + walletapi.FormatMoney(0)
		return b
	}

	id := crypto.HashHexToHash(s)

	balance, _, err := engram.Disk.GetDecryptedBalanceAtTopoHeight(id, -1, engram.Disk.GetAddress().String())

	if err != nil {
		b = "Balance:  " + walletapi.FormatMoney(0)
	} else {
		b = "Balance:  " + walletapi.FormatMoney(balance)
	}

	return b
}

// Display seed to the user in preferred language
func display_seed() string {
	seed := engram.Disk.GetSeed()

	return seed
}

// Display keys
func display_spend_key() (secret string, public string) {

	keys := engram.Disk.Get_Keys()
	secret = keys.Secret.Text(16)
	public = keys.Public.StringHex()

	return secret, public
}

// Detect and open URLs in default system browser
func openURL(url string, a *App) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = exec.Command("open", url).Start()
	}
	if err != nil {
		return
	}

}

func checkSeedWord(w string) bool {
	_, _, _, check := mnemonics.Find_indices([]string{w})
	if check {
		return true
	}
	return false
}

func addTransfer(args rpc.Arguments) {
	tx.Pending = append(tx.Pending, rpc.Transfer{Amount: tx.Amount, Destination: tx.Address.String(), Payload_RPC: args})
	fmt.Printf("[Send] Added transfer to the pending list.\n")

	session.Window.SetContent(layoutTransition())
	session.Window.SetContent(layoutTransfers())
}

func sendTransfers() (err error) {
	tx.TX, err = engram.Disk.TransferPayload0(tx.Pending, tx.Ringsize, false, rpc.Arguments{}, 0, false)

	if err != nil {
		fmt.Printf("[Send] Error: %s\n", err)
		return
	}

	if err = engram.Disk.SendTransaction(tx.TX); err != nil {
		fmt.Printf("[Send] Error while dispatching transaction: %s\n", err)
		return
	}

	tx.Fees = tx.TX.Fees()
	tx.TXID = tx.TX.GetHash()

	fmt.Printf("[Send] Dispatched transaction: %s\n", tx.TXID)

	tx = Transfers{}

	return
}

// Routine for account registration
func registerAccount() {
	session.Domain = "app.register"
	if engram.Disk == nil {
		resizeWindow(MIN_WIDTH, MIN_HEIGHT)
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutMain())
		session.Domain = "app.main"
		return
	}

	btnClose := widget.NewButton("Cancel", nil)
	btnClose.OnTapped = func() {
		session.Gif.Stop()
		session.Gif = nil
		closeWallet()
	}

	title := canvas.NewText("R E G I S T R A T I O N", colors.Gray)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 16

	heading := canvas.NewText("Please wait...", colors.Gray)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	sub := canvas.NewText("This one-time process can take a while.", colors.Gray)
	sub.TextSize = 14
	sub.Alignment = fyne.TextAlignCenter
	sub.TextStyle = fyne.TextStyle{Bold: true}

	resizeWindow(MIN_WIDTH, MIN_HEIGHT)
	session.Window.SetContent(layoutTransition())
	session.Window.SetContent(layoutWaiting(title, heading, sub, btnClose))

	go func() {
		var reg_tx *transaction.Transaction
		successful_regs := make(chan *transaction.Transaction)

		counter := 0

		for i := 0; i < runtime.GOMAXPROCS(0)-1; i++ {
			go func() {
				for counter == 0 {
					if engram.Disk == nil {
						break
					} else if engram.Disk.IsRegistered() {
						break
					}

					lreg_tx := engram.Disk.GetRegistrationTX()
					hash := lreg_tx.GetHash()

					if hash[0] == 0 && hash[1] == 0 && hash[2] == 0 {
						successful_regs <- lreg_tx
						counter++
						break
					}
				}

			}()
		}

		if engram.Disk == nil {
			resizeWindow(MIN_WIDTH, MIN_HEIGHT)
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutMain())
			session.Domain = "app.main"
			return
		}

		reg_tx = <-successful_regs

		fmt.Printf("[Registration] Registration TXID: %s\n", reg_tx.GetHash())
		err := engram.Disk.SendTransaction(reg_tx)
		if err != nil {
			fmt.Printf("[Registration] Error: %s\n", err)
			resizeWindow(MIN_WIDTH, MIN_HEIGHT)
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutMain())
			session.Domain = "app.main"
		} else {
			session.Gif.Stop()
			session.Gif = nil
			fmt.Printf("[Registration] Registration transaction dispatched successfully.\n")
			resizeWindow(MIN_WIDTH, MIN_HEIGHT)
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutDashboard())
			session.Domain = "app.wallet"
		}
	}()

	return
}

// Set the ring size for transactions
func setRingSize(wallet *walletapi.Wallet_Disk, s int) bool {
	if wallet == nil {
		fmt.Printf("[error] no wallet found.\n")
		return false
	}

	// Minimum ring size is 2, only accept powers of 2.
	if s < 2 {
		wallet.SetRingSize(2)
		fmt.Printf("[Engram] New transaction ring size: 2\n")
	} else {
		wallet.SetRingSize(s)
		fmt.Printf("[Engram] New transaction ring size: %d\n", s)
	}

	return true
}

// Get the username from the graviton database
// Returns error if no results are found
func getUsername() (result string, err error) {
	username, err := GetValue("usernames", []byte("username"))
	if err != nil {
		return
	}

	result = string(username)

	return
}

func checkUsername(s string) (result bool, address string, err error) {
	payload := DaemonRPC{
		Jsonrpc: "2.0",
		ID:      "1",
		Method:  "DERO.NameToAddress",
	}

	payload.Params.Name = s

	j, err := json.Marshal(payload)
	if err != nil {
		return
	}

	body := bytes.NewReader(j)

	// Connect to daemon
	url := "http://" + session.Daemon + "/json_rpc"
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var resData map[string]interface{}
	responseData, err := ioutil.ReadAll(resp.Body)

	err = json.Unmarshal(responseData, &resData)
	if err != nil {
		return
	}

	if resData["result"] == nil {
		err = errors.New("Username is not registered to this account")
		return
	}

	node := resData["result"].(map[string]interface{})
	address = node["address"].(string)

	result = true

	return
}

// Register a new DERO username
func registerUsername(s string) (err error) {
	// Check first if the name is taken
	check, _, err := checkUsername(s)
	if err == nil || check {
		fmt.Printf("[username] error: skipping registration - username exists.\n")
		err = errors.New("Username already exists")
		return
	}

	var args = rpc.Arguments{}
	args = append(args, rpc.Argument{string("entrypoint"), rpc.DataString, string("Register")})

	var cp = rpc.SC_Invoke_Params{
		SC_ID:            string("0000000000000000000000000000000000000000000000000000000000000001"), // string    `json:"scid"`
		SC_RPC:           args,                                                                       // arguments `json:"sc_rpc"`
		SC_DERO_Deposit:  uint64(0),                                                                  // uint64    `json:"sc_dero_deposit"`
		SC_TOKEN_Deposit: uint64(0),                                                                  // uint64    `json:"sc_token_deposit"`
	}

	var p rpc.Transfer_Params
	var dest string

	if session.Network {
		dest = "dero1qykyta6ntpd27nl0yq4xtzaf4ls6p5e9pqu0k2x4x3pqq5xavjsdxqgny8270"
	} else {
		dest = "deto1qy0ehnqjpr0wxqnknyc66du2fsxyktppkr8m8e6jvplp954klfjz2qqdzcd8p"
	}
	p.Transfers = append(p.Transfers, rpc.Transfer{
		Destination: dest,
		Amount:      0,
		Burn:        0,
	})

	p.SC_RPC = cp.SC_RPC
	p.SC_ID = cp.SC_ID

	cp.SC_RPC = append(cp.SC_RPC, rpc.Argument{rpc.SCACTION, rpc.DataUint64, uint64(0)})
	cp.SC_RPC = append(cp.SC_RPC, rpc.Argument{rpc.SCID, rpc.DataHash, crypto.HashHexToHash(cp.SC_ID)})
	cp.SC_RPC = append(cp.SC_RPC, rpc.Argument{string("name"), rpc.DataString, string(s)})

	var tx *transaction.Transaction
	tx, err = engram.Disk.TransferPayload0(p.Transfers, 2, false, cp.SC_RPC, 0, false)
	if err != nil {
		fmt.Printf("[Username] Error: %s\n", err)
		return
	}

	err = engram.Disk.SendTransaction(tx)
	if err != nil {
		fmt.Printf("[Username] Error: %s", err)
		return
	}

	go func() {
		c := 0
		for i := 0; i < 61; i++ {
			_, _, err = checkUsername(s)
			if err == nil {
				fmt.Printf("[Username] Successfully registered username: %s\n", s)
				fmt.Printf("[Username] Username Registration TXID:  %s\n", tx.GetHash().String())
				break
			} else {
				c++
				time.Sleep(1 * time.Second)
			}
		}

		_ = tx
		session.NewUser = ""

		if c >= 60 {
			fmt.Printf("[Username] error: timed out when registering username: %s\n", s)
		}
	}()

	return
}

func sendMessage(m string, s string, r string) (err error) {
	if m == "" {
		return
	}

	mapAddress := ""
	a, err := globals.ParseValidateAddress(r)
	if err != nil {
		mapAddress, err = engram.Disk.NameToAddress(r)
		if err != nil {
			return
		}
		a, err = globals.ParseValidateAddress(mapAddress)
		if err != nil {
			return
		}
	}

	if s == "" {
		s = engram.Disk.GetAddress().String()
	}

	amount, err := globals.ParseAmount("0.00002")
	if err != nil {
		//logger.Error(err, "Err parsing amount")
		return
	}
	//var transfer []rpc.Transfer

	//integrated := engram.Disk.GetRandomIAddress8()

	var arguments = rpc.Arguments{
		{rpc.RPC_DESTINATION_PORT, rpc.DataUint64, uint64(1337)},
		{rpc.RPC_VALUE_TRANSFER, rpc.DataUint64, amount},
		// { rpc.RPC_EXPIRY , rpc.DataTime, time.Now().Add(time.Hour).UTC()},
		{rpc.RPC_COMMENT, rpc.DataString, m},
		{rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString, s},
	}

	if a.IsIntegratedAddress() { // read everything from the address

		if a.Arguments.Validate_Arguments() != nil {
			//fmt.Printf(err, "Integrated Address  arguments could not be validated.\n")
			return
		}

		if !a.Arguments.Has(rpc.RPC_DESTINATION_PORT, rpc.DataUint64) { // but only it is present
			//fmt.Printf(fmt.Errorf("Integrated Address does not contain destination port.\n"), "")
			return
		}

		arguments = append(arguments, rpc.Argument{Name: rpc.RPC_DESTINATION_PORT, DataType: rpc.DataUint64, Value: a.Arguments.Value(rpc.RPC_DESTINATION_PORT, rpc.DataUint64).(uint64)})

		if a.Arguments.Has(rpc.RPC_EXPIRY, rpc.DataTime) { // but only it is present

			if a.Arguments.Value(rpc.RPC_EXPIRY, rpc.DataTime).(time.Time).Before(time.Now().UTC()) {
				//fmt.Printf(nil, "This address has expired.", "expiry time", a.Arguments.Value(rpc.RPC_EXPIRY, rpc.DataTime))
				return
			} else {
				//fmt.Printf("This address will expire ", "expiry time", a.Arguments.Value(rpc.RPC_EXPIRY, rpc.DataTime))
			}
		}

		fmt.Printf("Destination port is integrated in address. %s\n", a.Arguments.Value(rpc.RPC_DESTINATION_PORT, rpc.DataUint64).(uint64))

		if a.Arguments.Has(rpc.RPC_COMMENT, rpc.DataString) { // but only it is present
			fmt.Printf("Integrated Message: %s\n", a.Arguments.Value(rpc.RPC_COMMENT, rpc.DataString))
			arguments = append(arguments, rpc.Argument{rpc.RPC_COMMENT, rpc.DataString, a.Arguments.Value(rpc.RPC_COMMENT, rpc.DataString)})
		}
	}

	for _, arg := range arguments {
		if !(arg.Name == rpc.RPC_COMMENT || arg.Name == rpc.RPC_EXPIRY || arg.Name == rpc.RPC_DESTINATION_PORT || arg.Name == rpc.RPC_SOURCE_PORT || arg.Name == rpc.RPC_VALUE_TRANSFER || arg.Name == rpc.RPC_NEEDS_REPLYBACK_ADDRESS) {
			switch arg.DataType {
			case rpc.DataString:
				arguments = append(arguments, rpc.Argument{Name: arg.Name, DataType: arg.DataType, Value: arg.Value.(string)})
			case rpc.DataInt64:
				arguments = append(arguments, rpc.Argument{Name: arg.Name, DataType: arg.DataType, Value: arg.Value.(string)})
			case rpc.DataUint64:
				arguments = append(arguments, rpc.Argument{Name: arg.Name, DataType: arg.DataType, Value: arg.Value.(string)})
			case rpc.DataFloat64:
				arguments = append(arguments, rpc.Argument{Name: arg.Name, DataType: arg.DataType, Value: arg.Value.(string)})
			case rpc.DataTime:
				fmt.Errorf("[Service] Time currently not supported.\n")
			}
		}
	}

	if a.Arguments.Has(rpc.RPC_VALUE_TRANSFER, rpc.DataUint64) { // but only it is present
		//logger.Info("Transaction", "Value", globals.FormatMoney(a.Arguments.Value(rpc.RPC_VALUE_TRANSFER, rpc.DataUint64).(uint64)))
		amount = a.Arguments.Value(rpc.RPC_VALUE_TRANSFER, rpc.DataUint64).(uint64)
	} else {
		amount, err = globals.ParseAmount("0.00002")
		if err != nil {
			//logger.Error(err, "Err parsing amount")
			return
		}
	}

	if a.Arguments.Has(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString) {
		arguments = append(arguments, rpc.Argument{Name: rpc.RPC_NEEDS_REPLYBACK_ADDRESS, DataType: rpc.DataString, Value: s})
	}

	// if no arguments, use space by embedding a small comment
	if len(arguments) == 0 { // allow user to enter Comment
		arguments = append(arguments, rpc.Argument{Name: rpc.RPC_DESTINATION_PORT, DataType: rpc.DataUint64, Value: uint64(1337)})
		arguments = append(arguments, rpc.Argument{Name: rpc.RPC_COMMENT, DataType: rpc.DataString, Value: m})
	}

	if _, err = arguments.CheckPack(transaction.PAYLOAD0_LIMIT); err != nil {
		fmt.Printf("[Message] Arguments packing err: %s\n", err)
		return
	}

	//transfer = append(transfer, rpc.Transfer{Amount: uint64(0), Destination: a.String(), Payload_RPC: arguments})

	tx, err := engram.Disk.TransferPayload0([]rpc.Transfer{rpc.Transfer{Amount: amount, Destination: a.String(), Payload_RPC: arguments}}, 0, false, rpc.Arguments{}, 0, false)
	//tx, err := engram.Disk.TransferPayload0(transfer, uint64(2), false, rpc.Arguments{}, uint64(0), false)
	if err != nil {
		return
	}

	if err = engram.Disk.SendTransaction(tx); err != nil {
		fmt.Printf("[Send] Error while dispatching transaction: %s\n", err)
		return
	}

	//fees := tx.Fees()
	txid := tx.GetHash()

	fmt.Printf("[Send] Dispatched transaction: %s\n", txid)

	return
}

func getMessagesFromUser(s string, h uint64) (result []rpc.Entry) {
	var zeroscid crypto.Hash
	if s == "" {
		return
	}

	messages := engram.Disk.Get_Payments_DestinationPort(zeroscid, uint64(1337), h)

	for m := range messages {
		var username bool
		var username2 bool
		txid := messages[m].TXID
		tx := engram.Disk.Get_Payments_TXID(txid)

		check, err := engram.Disk.NameToAddress(s)
		if err != nil {
			username = false
		} else {
			username = true
		}

		if tx.Incoming {
			if tx.Payload_RPC.HasValue(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString) {
				check2, err := engram.Disk.NameToAddress(tx.Payload_RPC.Value(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString).(string))
				if err != nil {
					username2 = false
				} else {
					username2 = true
				}

				if username && username2 {
					if check == check2 {
						result = append(result, messages[m])
					}
				} else if !username && !username2 {
					if s == tx.Payload_RPC.Value(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString).(string) {
						result = append(result, messages[m])
					}
				} else if check == tx.Payload_RPC.Value(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString).(string) {
					result = append(result, messages[m])
				} else if s == check2 {
					result = append(result, messages[m])
				}
			}
		} else {
			addr, err := engram.Disk.NameToAddress(s)
			if err != nil {
				if tx.Destination == s {
					result = append(result, messages[m])
				}
			} else {
				if tx.Destination == addr {
					result = append(result, messages[m])
				}
			}
		}
	}

	return
}

func getMessages(h uint64) (result []string) {
	var zeroscid crypto.Hash
	messages := engram.Disk.Get_Payments_DestinationPort(zeroscid, uint64(1337), h)

	for m := range messages {
		if messages[m].Incoming {
			if messages[m].Payload_RPC.HasValue(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString) {
				if messages[m].Payload_RPC.Value(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString).(string) == "" {

				} else {
					sender, err := engram.Disk.NameToAddress(messages[m].Payload_RPC.Value(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString).(string))
					if err != nil {
						addr, err := globals.ParseValidateAddress(messages[m].Payload_RPC.Value(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString).(string))
						if err != nil {

						} else {
							sender = addr.String()
							c := 0
							for r := range result {
								if strings.Contains(result[r], sender+":") {
									c += 1
								}
							}
							if c == 0 {
								result = append(result, sender+":")
							}
						}
					} else {
						c := 0
						for r := range result {
							if strings.Contains(result[r], sender+":"+messages[m].Payload_RPC.Value(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString).(string)) {
								c += 1
							}
						}
						if c == 0 {
							result = append(result, sender+":"+messages[m].Payload_RPC.Value(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString).(string))
						}
					}
				}
			}
		} else {
			if messages[m].Payload_RPC.HasValue(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString) {
				c := 0
				for r := range result {
					if strings.Contains(result[r], messages[m].Destination) {
						c += 1
					}
				}
				if c == 0 {
					result = append(result, messages[m].Destination+":")
				}
			}
		}
	}
	return
}

// Returns a list of registered usernames
func queryUsernames() (result []string, err error) {
	if gnomon.Active == 1 && engram.Disk != nil {
		result, _ = gnomon.Index.GetSCIDKeysByValue(nil, "0000000000000000000000000000000000000000000000000000000000000001", engram.Disk.GetAddress().String(), gnomon.Index.ChainHeight)
		//fmt.Printf("[queryUsernames-Live] result: %v", result)
		if len(result) <= 0 {
			result, _ = gnomon.DB.GetSCIDKeysByValue("0000000000000000000000000000000000000000000000000000000000000001", engram.Disk.GetAddress().String(), gnomon.Index.ChainHeight, true)
			//fmt.Printf("[queryUsernames-DB] result: %v", result)
		}
		sort.Sort(sort.StringSlice(result))
	}

	return
}

func setPrimaryUsername(s string) (err error) {
	err = StoreEncryptedValue("settings", []byte("username"), []byte(s))
	return
}

func getPrimaryUsername() (err error) {
	u, err := GetEncryptedValue("settings", []byte("username"))
	if err != nil {
		session.Username = ""
		return
	}
	session.Username = string(u)
	return
}

// Returns a list of SCIDs that a wallet interacted with
func queryAssets() (result []string, err error) {
	if gnomon.Active == 1 && engram.Disk != nil {
		gnomon.DB.DBFolder = "datashards/gnomon"
		result = gnomon.DB.GetSCIDInteractionByAddr(engram.Disk.GetAddress().String())
	}

	return
}

func prepareSC(path string) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Printf("%s", err)
		return
	}

	defer file.Close()
}

func startGnomon() {
	if gnomon.Index != nil {
		gnomon.Index.Endpoint = getDaemon()
	} else if gnomon.Index == nil && gnomon.Active == 1 {
		folder := "datashards/gnomon"
		if !session.Network {
			folder = "datashards/gnomon_testnet"
		}
		gnomon.DB = storage.NewGravDB(folder, "25ms")
		gnomon.Index = indexer.NewIndexer(gnomon.DB, "Function Initialize", gnomon.DB.GetLastIndexHeight(), session.Daemon, "Daemon", false, false, true)
		gnomon.Index.StartDaemonMode()

		fmt.Printf("[Gnomon] Scan Status: [%d / %d]\n", gnomon.DB.GetLastIndexHeight(), gnomon.Index.LastIndexedHeight)
	}
}

func stopGnomon() {
	if gnomon.Index != nil {
		gnomon.Index.Closing = true
		gnomon.Index.Close()
		gnomon.Index = nil
		fmt.Printf("[Gnomon] Closed all indexers.\n")
	}
}

func installSC(path string) (result string, err error) {
	data := &InstallContract{}
	file, err := os.Open(path)
	if err != nil {
		fmt.Printf("%s", err)
		return
	}

	defer file.Close()

	url := fmt.Sprintf("http://127.0.0.1:%d/install_sc", DEFAULT_WALLET_PORT)
	req, err := http.NewRequest("POST", url, file)
	if err != nil {
		fmt.Printf("%s", err)
		return
	}

	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("%s", err)
		return
	}

	responseData, err := ioutil.ReadAll(resp.Body)

	err = json.Unmarshal([]byte(responseData), &data)
	if err != nil {
		fmt.Printf("%s", err)
		return
	}

	defer resp.Body.Close()

	result = data.TXID

	// Store SCID in local Graviton database
	//err = StoreValue("Creations", []byte(result), []byte(session.Wallet))

	return
}

func newRPCPassword() (s string) {
	r := make([]byte, 20)
	_, err := rand.Read(r)
	if err != nil {
		panic(err)
	}

	s = base64.URLEncoding.EncodeToString(r)
	cyberdeck.pass = s
	return
}

func newRPCUsername() (s string) {
	r, _ := rand.Int(rand.Reader, big.NewInt(1600))
	w := mnemonics.Key_To_Words(r, "english")
	l := strings.Split(string(w), " ")
	s = l[len(l)-2]
	cyberdeck.user = s
	return
}

func cyberdeckUpdate() {
	var timer int
	cyberdeck.interval = 60
	status.Authenticator.SetValue(60)

	for cyberdeck.active == 1 {
		if cyberdeck.server != nil {
			cyberdeck.status.Text = "Allowed"
			cyberdeck.status.Color = colors.Green
			cyberdeck.status.Refresh()
			status.Cyberdeck.FillColor = colors.Gray
			status.Cyberdeck.Refresh()
			status.Cyberdeck.FillColor = colors.Green
			status.Cyberdeck.Refresh()
			cyberdeck.toggle.SetText("Turn Off")
			cyberdeck.userText.Disable()
			cyberdeck.passText.Disable()
			cyberdeck.checkbox.Disable()

			if timer >= cyberdeck.interval {
				timer = 0
				if cyberdeck.mode == 1 {
					setRPCLogin()
					fmt.Printf("[Cyberdeck] Authentication Credentials Updated\n")
					return
				}
			} else {
				if cyberdeck.mode == 1 {
					status.Authenticator.SetValue(float64(60 - timer))
					status.Authenticator.Refresh()
					cyberdeck.userText.Text = cyberdeck.user
					cyberdeck.passText.Text = cyberdeck.pass
					cyberdeck.userText.Refresh()
					cyberdeck.passText.Refresh()
					cyberdeck.checkbox.Checked = true
					cyberdeck.checkbox.Refresh()
				} else {
					cyberdeck.userText.Text = cyberdeck.user
					cyberdeck.passText.Text = cyberdeck.pass
					cyberdeck.userText.Refresh()
					cyberdeck.passText.Refresh()
					cyberdeck.checkbox.Checked = false
					cyberdeck.checkbox.Refresh()
				}
			}
		} else {
			cyberdeck.status.Text = "Blocked"
			cyberdeck.status.Color = colors.Gray
			cyberdeck.status.Refresh()
			status.Cyberdeck.FillColor = colors.Gray
			status.Cyberdeck.Refresh()
			cyberdeck.toggle.SetText("Turn On")
		}

		timer++
		time.Sleep(time.Second)
	}
}

// TODO: Rework this completely
func setRPCLogin() {
	var err error
	if engram.Disk == nil {
		return
	}

	if cyberdeck.active == 1 && cyberdeck.server != nil && cyberdeck.mode == 1 {
		cyberdeck.active = 0
		cyberdeck.server.RPCServer_Stop()
	}

	if cyberdeck.mode == 1 {
		cyberdeck.user = newRPCUsername()
		cyberdeck.pass = newRPCPassword()
		globals.Arguments["--rpc-login"] = cyberdeck.user + ":" + cyberdeck.pass
		cyberdeck.userText.Text = cyberdeck.user
		cyberdeck.passText.Text = cyberdeck.pass
		cyberdeck.userText.Refresh()
		cyberdeck.passText.Refresh()
	}

	if !session.Network {
		globals.Arguments["--rpc-bind"] = fmt.Sprintf("127.0.0.1:%d", DEFAULT_TESTNET_WALLET_PORT)
	} else {
		globals.Arguments["--rpc-bind"] = fmt.Sprintf("127.0.0.1:%d", DEFAULT_WALLET_PORT)
	}

	globals.Arguments["--rpc-login"] = cyberdeck.user + ":" + cyberdeck.pass

	cyberdeck.server, err = rpcserver.RPCServer_Start(engram.Disk, "Cyberdeck")
	if err != nil {
		cyberdeck.server = nil
		cyberdeck.status.Text = "Blocked"
		cyberdeck.status.Color = colors.Gray
		cyberdeck.status.Refresh()
		status.Cyberdeck.FillColor = colors.Red
		status.Cyberdeck.StrokeColor = colors.Red
		status.Cyberdeck.Refresh()
		status.Authenticator.Value = 0
		status.Authenticator.Refresh()
		return
	}

	cyberdeck.status.Text = "Allowed"
	cyberdeck.status.Color = colors.Green
	cyberdeck.status.Refresh()
	status.Cyberdeck.FillColor = colors.Green
	status.Cyberdeck.StrokeColor = colors.Green
	status.Cyberdeck.Refresh()
	if cyberdeck.mode == 0 {
		cyberdeck.userText.Enable()
		cyberdeck.passText.Enable()
	} else {
		cyberdeck.userText.Disable()
		cyberdeck.passText.Disable()
	}
	cyberdeck.active = 1
	go cyberdeckUpdate()

	return
}

func getPrice() {
	check, _ := engram.Disk.Get_Balance()
	if check == 0 {
		session.BalanceUSDText.Text = "USD  0.00"
		session.BalanceUSDText.Refresh()
		return
	}

	url := "https://api.coingecko.com/api/v3/simple/price?ids=dero&vs_currencies=usd"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("%s", err)
		session.BalanceUSDText.Text = ""
		session.BalanceUSDText.Refresh()
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("[USD] %s\n", err)
		session.BalanceUSDText.Text = ""
		session.BalanceUSDText.Refresh()
		return
	}
	defer resp.Body.Close()

	var resData map[string]interface{}
	responseData, err := ioutil.ReadAll(resp.Body)

	err = json.Unmarshal(responseData, &resData)
	if err != nil {
		fmt.Printf("[USD] %s\n", err)
		session.BalanceUSDText.Text = ""
		session.BalanceUSDText.Refresh()
		return
	}

	defer resp.Body.Close()

	if resData["dero"] == nil {
		err = errors.New("error: could not query price from coingecko")
		fmt.Printf("[USD] %s\n", err)
		fmt.Printf("[USD] %s\n", resp.Body)
		fmt.Printf("[USD] %s\n", resData)
		session.BalanceUSDText.Text = ""
		session.BalanceUSDText.Refresh()
		return
	}

	node := resData["dero"].(map[string]interface{})
	result := fmt.Sprintf("%.2f", node["usd"])
	f, _ := strconv.ParseFloat(result, 5)
	tmp, _ := engram.Disk.Get_Balance()
	bal := fmt.Sprintf("%d", tmp)
	b, _ := strconv.ParseFloat(bal, 5)
	usd := (f / 100000) * b
	formatted := fmt.Sprintf("%.2f", usd)
	session.BalanceUSD = formatted
	session.BalanceUSDText.Text = "USD  " + formatted
	session.BalanceUSDText.Refresh()
	fmt.Printf("[USD] Value conversion updated.\n")
}

/*
func invokeSC(f string, v string, u string, p string) (r *jsonrpc.RPCResponse) {
	rpcClient := jsonrpc.NewClientWithOpts("http://127.0.0.1:40403/rpc", &jsonrpc.RPCClientOpts{
		CustomHeaders: map[string]string{
			"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(u+":"+p)),
		},
	})
	r, _ = rpcClient.Call(f, v) // send with Authorization-Header

	return
}
*/

func checkAccount(s string) (err error) {
	var zero crypto.Hash
	payload := DaemonCheckRPC{
		Jsonrpc: "2.0",
		ID:      "1",
		Method:  "DERO.GetEncryptedBalance",
	}

	payload.Params.Address = s
	payload.Params.SCID = zero
	payload.Params.Merkle_Balance_TreeHash = ""
	payload.Params.TopoHeight = engram.Disk.Get_Daemon_TopoHeight()

	j, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("[Engram] Marshall: %s", err)
		return
	}

	body := bytes.NewReader(j)

	// Connect to daemon
	url := "http://" + session.Daemon + "/json_rpc"
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		fmt.Printf("[Engram] Connect: %s", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("[Engram] Response: %s", err)
		return
	}
	defer resp.Body.Close()

	var resData map[string]interface{}
	responseData, err := ioutil.ReadAll(resp.Body)

	err = json.Unmarshal(responseData, &resData)
	if err != nil {
		fmt.Printf("[Engram] Unmarshall: %s\n", err)
		return
	}

	fmt.Printf("[Check] Data: %s\n", resData)

	if resData["result"] == nil {
		err = errors.New("account is not registered")
		fmt.Printf("[Engram] %s\n", err)
		return
	}

	node := resData["result"].(map[string]interface{})
	fmt.Printf("[Engram] Dump: %s", node)
	reg := node["registration"].(string)

	//result = true

	fmt.Printf("[Check] %s is a registered\n", reg)

	return
}
