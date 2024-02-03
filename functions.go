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
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	x "fyne.io/x/fyne/widget"
	"github.com/civilware/Gnomon/indexer"
	"github.com/civilware/Gnomon/rwc"
	"github.com/civilware/Gnomon/structures"
	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
	"github.com/gorilla/websocket"
	"mvdan.cc/xurls/v2"

	"github.com/civilware/Gnomon/storage"
	"github.com/deroproject/derohe/config"
	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/dvm"
	"github.com/deroproject/derohe/globals"
	"github.com/deroproject/derohe/proof"

	"github.com/deroproject/derohe/rpc"
	"github.com/deroproject/derohe/transaction"

	"github.com/deroproject/derohe/walletapi"
	"github.com/deroproject/derohe/walletapi/mnemonics"
	"github.com/deroproject/derohe/walletapi/rpcserver"
)

// Get the Engram settings from the local Graviton tree
func initSettings() {
	getTestnet()
	getMode()
	getDaemon()
	getGnomon()
}

// Go routine to update the latest information from the connected daemon (Online Mode only)
func StartPulse() {
	if !walletapi.Connected && engram.Disk != nil {

		fmt.Printf(
			attemptConnection,
			walletapi.Daemon_Endpoint,
		)

		err := walletapi.Connect(session.Daemon)
		if err != nil {

			fmt.Printf(
				failedConnection,
				walletapi.Daemon_Endpoint,
			)

			walletapi.Connected = false
			closeWallet()
			session.Window.SetContent(layoutAlert(1))
			removeOverlays()
			return
		} else {
			walletapi.Connected = true
			engram.Disk.SetOnlineMode()
			session.BalanceText = canvas.NewText(string_, colors.Blue)
			session.StatusText = canvas.NewText(string_, colors.Blue)
			status.Connection.FillColor = colors.Gray
			status.Connection.Refresh()
			status.Sync.FillColor = colors.Gray
			status.Sync.Refresh()

			go func() {
				for walletapi.Connected && engram.Disk != nil {
					if walletapi.Get_Daemon_Height() < 1 {

						fmt.Printf(
							attemptConnection,
							walletapi.Daemon_Endpoint,
						)

						err := walletapi.Connect(session.Daemon)
						if err != nil {

							fmt.Printf(
								failedConnection,
								walletapi.Daemon_Endpoint,
							)

							walletapi.Connected = false
							closeWallet()
							session.Window.SetContent(layoutAlert(1))
							removeOverlays()
							break
						}
					}

					if !engram.Disk.IsRegistered() {
						if !walletapi.Connected {

							fmt.Printf(
								failedDaemonConnection,
								engram.Disk.Get_Daemon_TopoHeight(),
							)

							status.Connection.FillColor = colors.Red
							status.Connection.Refresh()
							status.Sync.FillColor = colors.Red
							status.Sync.Refresh()
						}

						time.Sleep(time.Second)
					} else {
						session.Balance, _ = engram.Disk.Get_Balance()
						session.BalanceText.Text = globals.FormatMoney(session.Balance)
						session.BalanceText.Refresh()
						session.WalletHeight = engram.Disk.Get_Height()
						session.DaemonHeight = engram.Disk.Get_Daemon_Height()
						session.StatusText.Text = fmt.Sprintf(
							"%d",
							session.WalletHeight,
						)
						session.StatusText.Refresh()

						if session.LastBalance != session.Balance && session.Balance != 0 {
							go convertBalance()
						}

						session.LastBalance = session.Balance

						if walletapi.IsDaemonOnline() {
							status.Connection.FillColor = colors.Green
							status.Connection.Refresh()
							if session.DaemonHeight > 0 && session.DaemonHeight-session.WalletHeight < 2 {
								status.Connection.FillColor = colors.Green
								status.Connection.Refresh()
								status.Sync.FillColor = colors.Green
								status.Sync.Refresh()
							} else if session.DaemonHeight == 0 {
								status.Sync.FillColor = colors.Red
								status.Sync.Refresh()
							} else {
								status.Sync.FillColor = colors.Yellow
								status.Sync.Refresh()
							}
						} else {
							status.Connection.FillColor = colors.Gray
							status.Connection.Refresh()
							status.Sync.FillColor = colors.Gray
							status.Sync.Refresh()
							status.Cyberdeck.FillColor = colors.Gray
							status.Cyberdeck.Refresh()
							fmt.Printf(
								networkOffline +
									strconv.FormatUint(
										session.WalletHeight,
										base10,
									) +
									singlebackslash +
									strconv.FormatUint(
										session.DaemonHeight,
										base10,
									) +
									singlenewline,
							)
						}

						time.Sleep(time.Second)
					}
				}

				if walletapi.Connected {
					walletapi.Connected = false
				}
			}()
		}
	}
}

// Get Network setting from the local Graviton tree (Ex: Mainnet, Testnet, Simulator)
func getTestnet() bool {
	result, err := GetValue(
		stringsettings,
		bytenetwork,
	)
	if err != nil {
		session.Testnet = false
		globals.Arguments[stringFlagtestnet] = false
		setTestnet(false)
		return false
	} else {
		if string(result) == stringTestnet {
			session.Testnet = true
			globals.Arguments[stringFlagtestnet] = true
			return true
		} else {
			session.Testnet = false
			globals.Arguments[stringFlagtestnet] = false
			return false
		}
	}
}

// Set Network setting to the local Graviton tree (Ex: Mainnet, Testnet, Simulator)
func setTestnet(b bool) (err error) {
	s := string_
	if !b {
		s = stringMainnet
		globals.Arguments[stringFlagtestnet] = false
	} else {
		s = stringTestnet
		globals.Arguments[stringFlagtestnet] = true
	}

	StoreValue(
		stringsettings,
		bytenetwork,
		[]byte(s),
	)

	return
}

// Get daemon endpoint setting from the local Graviton tree
func getDaemon() (r string) {
	result, err := GetValue(
		stringsettings,
		byteendpoint,
	)
	if err != nil {
		r = DEFAULT_REMOTE_DAEMON
		setDaemon(r)
		session.Daemon = r
		globals.Arguments[stringFlagdaemonaddress] = r
		return
	}

	r = string(result)
	session.Daemon = r
	globals.Arguments[stringFlagdaemonaddress] = r
	return
}

// Set the daemon endpoint setting to the local Graviton tree
func setDaemon(s string) (err error) {
	StoreValue(
		stringsettings,
		byteendpoint,
		[]byte(s),
	)
	globals.Arguments[stringFlagdaemonaddress] = s
	session.Daemon = s
	return
}

// Get mode (online, offline) setting from local Graviton tree
func getMode() {

	/*
		if globals.Arguments[stringOffline].(bool) == true {
			session.Mode = stringOffline
			return
		}

		s := "mode"
		t := stringsettings
		key := []byte(s)
		result, err := GetValue(t, key)
		if err != nil {
			session.Mode = "Online"
			err := setMode("Online")
			globals.Arguments[stringOffline] = false
			if err != nil {
				fmt.Printf("[Engram] Error: %s\n", err)
				return
			}
		} else {
			if result == nil {
				session.Mode = "Online"
				err := setMode("Online")
				globals.Arguments[stringOffline] = false
				if err != nil {
					fmt.Printf("[Engram] Error: %s\n", err)
					return
				}
			} else {
				if string(result) == stringOffline {
					globals.Arguments[stringOffline] = true
					session.Mode = stringOffline
				} else {
					globals.Arguments[stringOffline] = false
					session.Mode = "Online"
				}
			}
		}
	*/
}

// Set the default Offline Mode settings to the local Graviton tree
func setMode(s string) (err error) {
	err = StoreValue(
		stringsettings,
		bytemode,
		[]byte(s),
	)
	if s == stringOffline {
		globals.Arguments[stringOffline] = true
	} else {
		globals.Arguments[stringOffline] = false
	}
	return
}

// Get the default Gnomon settings from local Graviton tree
func getGnomon() (r string, err error) {
	v, err := GetValue(
		stringsettings,
		bytegnomon,
	)
	if err != nil {
		gnomon.Active = 1
		if gnomon.Index != nil {
			gnomon.Index.Endpoint = getDaemon()
		}
		StoreValue(
			stringsettings,
			bytegnomon,
			byte1,
		)
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

// Set the default Gnomon settings to the local Graviton tree
func setGnomon(s string) (err error) {
	if s == string1 {
		err = StoreValue(
			stringsettings,
			bytegnomon,
			byte1,
		)
		gnomon.Active = 1
		if gnomon.Index != nil {
			gnomon.Index.Endpoint = getDaemon()
		}
	} else {
		err = StoreValue(
			stringsettings,
			bytegnomon,
			byte0,
		)
		gnomon.Active = 0
	}
	return
}

/*
func getAuthMode() (result string, err error) {
	r, err := GetValue(stringsettings, byteauthmode)
	if err != nil {
		StoreValue(stringsettings, byteauthmode, bytetrue)
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
*/

// Get the auth_mode settings from local Graviton tree
func setAuthMode(s string) {
	if s == stringtrue {
		StoreValue(
			stringsettings,
			byteauthmode,
			bytetrue,
		)
	} else {
		StoreValue(
			stringsettings,
			byteauthmode,
			bytefalse,
		)
	}
}

// Check if a URL exists in the string
func getTextURL(s string) (result []string) {
	return xurls.Relaxed().FindAllString(s, -1)
}

// Set the window size from provided height and width
func resizeWindow(width float32, height float32) {
	s := fyne.NewSize(width, height)
	session.Window.Resize(s)
}

// Close the active wallet
func closeWallet() {
	showLoadingOverlay()

	if gnomon.Index != nil {
		fmt.Printf(gnomonShuttingdown)
		stopGnomon()
	}

	if engram.Disk != nil {
		fmt.Printf(engramShuttingdown)
		engram.Disk.SetOfflineMode()
		engram.Disk.Save_Wallet()

		globals.Exit_In_Progress = true
		engram.Disk.Close_Encrypted_Wallet()
		session.WalletOpen = false
		session.Domain = appMain
		session.BalanceUSD = string_
		session.LastBalance = 0
		engram.Disk = nil
		tx = Transfers{}

		if cyberdeck.server != nil {
			cyberdeck.server.RPCServer_Stop()
			cyberdeck.server = nil
			fmt.Printf(cyberdeckShuttingdown)
		}

		if rpc_client.WS != nil {
			rpc_client.WS.Close()
			rpc_client.WS = nil
			fmt.Printf(websocketShuttingdown)
		}

		if rpc_client.RPC != nil {
			rpc_client.RPC.Close()
			rpc_client.RPC = nil
			fmt.Printf(rpcShuttingdown)
		}

		session.Path = string_
		session.Name = string_

		session.LastDomain = layoutMain()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutMain())
		removeOverlays()
		//session.Window.CenterOnScreen()
		fmt.Printf(shutdownSucces)
		return
	}
}

// Create a new account and wallet file
func create() (address string, seed string, err error) {
	check := findAccount()

	if session.Path == string_ {
		session.Error = enterAccountName
	} else if session.Language == -1 {
		session.Error = enterLanguage
	} else if session.Password == string_ {
		session.Error = enterPasword
	} else if session.PasswordConfirm == string_ {
		session.Error = confirmPass
	} else if session.PasswordConfirm != session.Password {
		session.Error = passMismatch
	} else if check {
		session.Error = accountExists
	} else {
		engram.Disk, err = walletapi.Create_Encrypted_Wallet_Random(
			session.Path,
			session.Password,
		)

		if err != nil {
			session.Language = -1
			session.Name = string_
			session.Path = string_
			session.Password = string_
			session.PasswordConfirm = string_
			session.Error = accountNotCreated
		} else {
			if session.Testnet {
				engram.Disk.SetNetwork(false)
				globals.Arguments[stringFlagtestnet] = true
			} else {
				engram.Disk.SetNetwork(true)
				globals.Arguments[stringFlagtestnet] = false
			}

			languages := mnemonics.Language_List()

			if session.Language < 0 || session.Language > len(languages)-1 {
				session.Language = 0 // English
			}

			engram.Disk.SetSeedLanguage(languages[session.Language])
			address = engram.Disk.GetAddress().String()
			seed = engram.Disk.GetSeed()
			engram.Disk.Close_Encrypted_Wallet()
			engram.Disk = nil
			session.Error = accountCreated
			session.Language = -1
			session.Name = string_
			session.Path = string_
			session.Password = string_
			session.PasswordConfirm = string_
			session.Domain = appMain
		}
	}
	return
}

// The main login routine
func login() {
	var err error
	var temp *walletapi.Wallet_Disk

	showLoadingOverlay()

	if engram.Disk == nil {
		temp, err = walletapi.Open_Encrypted_Wallet(
			session.Path,
			session.Password,
		)
		if err != nil {
			temp = nil
			session.Domain = appMain
			session.Error = err.Error()
			if len(session.Error) > 40 {
				session.Error = fmt.Sprintf(
					"%s...",
					session.Error[0:40],
				)
			}
			session.Window.Canvas().Content().Refresh()
			removeOverlays()
			return
		}

		engram.Disk = temp
		temp = nil
		session.Password = string_
	}

	if session.Testnet {
		engram.Disk.SetNetwork(false)
		globals.Arguments[stringFlagtestnet] = true
	} else {
		engram.Disk.SetNetwork(true)
		globals.Arguments[stringFlagtestnet] = false
	}

	session.WalletOpen = true
	session.BalanceUSD = string_
	session.LastBalance = 0

	if !session.Offline {
		walletapi.SetDaemonAddress(session.Daemon)
		engram.Disk.SetDaemonAddress(session.Daemon)

		if session.TrackRecentBlocks > 0 {
			fmt.Printf(
				scanTracking,
				session.TrackRecentBlocks,
			)
			engram.Disk.SetTrackRecentBlocks(session.TrackRecentBlocks)
		}

		StartPulse()
	} else {
		engram.Disk.SetOfflineMode()
		status.Connection.FillColor = colors.Gray
		status.Connection.Refresh()
		status.Sync.FillColor = colors.Gray
		status.Sync.Refresh()
	}

	setRingSize(engram.Disk, 16)
	session.Verified = false

	if !session.Offline {
		// Online mode
		status.Connection.FillColor = colors.Green
		status.Connection.Refresh()
		session.Balance = 0

		if !walletapi.Connected {
			closeWallet()
			session.Window.SetContent(layoutAlert(1))
			removeOverlays()
			return
		}

		if engram.Disk.Get_Height() < session.DaemonHeight {
			time.Sleep(time.Second * 1)
		}

		for i := 0; i < 10; i++ {
			reg := engram.Disk.Get_Registration_TopoHeight()

			if reg < 1 {
				time.Sleep(time.Second * 1)
			} else {
				break
			}

			if i == 9 {
				registerAccount()
				removeOverlays()
				session.Verified = true
				fmt.Printf(regStartNotice)
				fmt.Printf(regInprogressNotice)
				return
			}
		}

		go startGnomon()
	}

	if a.Driver().Device().IsMobile() {
		session.Domain = appWallet
		resizeWindow(ui.MaxWidth, ui.MaxHeight)
	}

	session.Window.SetContent(layoutDashboard())
	removeOverlays()

	session.Balance, _ = engram.Disk.Get_Balance()
	session.BalanceText.Text = globals.FormatMoney(session.Balance)
	session.BalanceText.Refresh()

	session.WalletHeight = engram.Disk.Wallet_Memory.Get_Height()
	session.DaemonHeight = engram.Disk.Get_Daemon_Height()
	session.StatusText.Text = fmt.Sprintf(
		"%d",
		session.WalletHeight,
	)
	session.StatusText.Refresh()

	if session.WalletHeight == session.DaemonHeight && !session.Offline {
		status.Sync.FillColor = colors.Green
		status.Sync.Refresh()
	}

	address := engram.Disk.GetAddress().String()
	shard := fmt.Sprintf(
		"%x",
		sha1.Sum(
			[]byte(address),
		),
	)
	session.ID = shard
	session.LimitMessages = true
}

// Remove all overlays
func removeOverlays() {
	overlays := session.Window.Canvas().Overlays()
	list := overlays.List()

	for o := range list {
		overlays.Remove(list[o])
	}

	if res.loading != nil {
		res.loading.Hide()
		res.loading.Stop()
		res.loading = nil
	}
}

// Add an overlay with the loading animation
func showLoadingOverlay() {
	frame := &iframe{}

	if res.loading == nil {
		res.loading, _ = x.NewAnimatedGifFromResource(resourceLoadingGif)
		res.loading.SetMinSize(
			fyne.NewSize(
				ui.Width*0.45,
				ui.Width*0.45,
			),
		)
	}

	rect := canvas.NewRectangle(colors.DarkMatter)
	rect.SetMinSize(frame.Size())

	background := container.NewStack(
		rect,
		container.NewCenter(
			res.loading,
		),
	)

	res.loading.Start()

	layout := container.NewStack(
		frame,
		background,
	)

	overlays := session.Window.Canvas().Overlays()
	overlays.Add(layout)
}

// Load embedded resources
func loadResources() {
	res.bg = canvas.NewImageFromResource(resourceBgPng)
	res.bg.FillMode = canvas.ImageFillContain

	res.bg2 = canvas.NewImageFromResource(resourceBg2Png)
	res.bg2.FillMode = canvas.ImageFillContain

	res.icon = canvas.NewImageFromResource(resourceIconPng)
	res.icon.FillMode = canvas.ImageFillContain

	res.header = canvas.NewImageFromResource(resourceBackground1Png)
	res.header.FillMode = canvas.ImageFillContain

	res.load = canvas.NewImageFromResource(resourceLoadPng)
	res.load.FillMode = canvas.ImageFillStretch

	res.dero = canvas.NewImageFromResource(resourceDeroPng)
	res.dero.FillMode = canvas.ImageFillContain

	res.gram = canvas.NewImageFromResource(resourceGramPng)
	res.gram.FillMode = canvas.ImageFillContain

	res.block = canvas.NewImageFromResource(resourceBlockGrayPng)
	res.block.FillMode = canvas.ImageFillContain

	res.red_alert = canvas.NewImageFromResource(resourceRedAlertPng)
	res.red_alert.FillMode = canvas.ImageFillContain

	res.green_alert = canvas.NewImageFromResource(resourceGreenAlertPng)
	res.green_alert.FillMode = canvas.ImageFillContain
}

// Reset UI resources
func resetResources() {
	res = Res{}
	loadResources()
}

// Get SC balance by SCID
func get_balance_SCID(s string) string {
	b := string_

	if s == string_ {
		b = formatBalance + walletapi.FormatMoney(0)
		return b
	}

	id := crypto.HashHexToHash(s)

	balance, _, err := engram.Disk.GetDecryptedBalanceAtTopoHeight(
		id,
		-1,
		engram.Disk.GetAddress().String(),
	)

	if err != nil {
		b = formatBalance + walletapi.FormatMoney(0)
	} else {
		b = formatBalance + walletapi.FormatMoney(balance)
	}

	return b
}

// Check the transaction ring members to see if the provided address exists
func ring_member_exists(txid string, address string) bool {
	if engram.Disk == nil || session.Offline {
		return false
	}

	var err error
	var tx_params rpc.GetTransaction_Params
	var tx_result rpc.GetTransaction_Result

	tx_params.Tx_Hashes = append(tx_params.Tx_Hashes, txid)

	rpc_client.WS, _, err = websocket.DefaultDialer.Dial(
		"ws://"+session.Daemon+"/ws",
		nil,
	)

	input_output := rwc.New(rpc_client.WS)
	rpc_client.RPC = jrpc2.NewClient(
		channel.RawJSON(input_output, input_output),
		nil,
	)

	if err = rpc_client.RPC.CallResult(
		context.Background(),
		DAEMON_GET_TX,
		tx_params,
		&tx_result,
	); err != nil {
		fmt.Printf(
			checkingRings,
			txid,
			err,
		)
		return false
	}

	rpc_client.WS.Close()
	rpc_client.RPC.Close()

	if tx_result.Status != stringOK {
		fmt.Printf(checkingRings, txid, tx_result.Status)
		return false
	}

	if len(tx_result.Txs_as_hex[0]) < 50 {
		return false
	}

	ring := tx_result.Txs[0].Ring

	for i := 0; i < len(ring[0]); i++ {
		if ring[0][i] == address {
			fmt.Printf(checkingRingsVerified, txid)
			return true
		}
	}

	fmt.Printf(checkingRingsUnverified, txid)
	return false
}

// Get the recovery words (seed words) for an account
func display_seed() string {
	seed := engram.Disk.GetSeed()

	return seed
}

// Get the account public/private keys
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
		err = exec.Command(
			"rundll32",
			"url.dll,FileProtocolHandler",
			url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = exec.Command("open", url).Start()
	}
	if err != nil {
		return
	}

}

// Validate if the provided word is a seed word
func checkSeedWord(w string) (check bool) {
	split := strings.Split(w, " ")

	if len(split) > 1 {
		return
	}
	_, _, _, check = mnemonics.Find_indices([]string{w})

	return
}

// Add a DERO transfer to the batch
func addTransfer() error {
	var arguments = rpc.Arguments{}
	var err error

	fmt.Printf(startTx)
	if tx.Address.IsIntegratedAddress() {
		if tx.Address.Arguments.Validate_Arguments() != nil {
			fmt.Printf(
				integratedAddressCantValidate,
			)
			err = errors.New(
				integratedAddressCantValidate,
			)
			return err
		}

		fmt.Printf(notIntegrated)
		if !tx.Address.Arguments.Has(
			rpc.RPC_DESTINATION_PORT,
			rpc.DataUint64) {

			fmt.Printf(
				noDestPort,
			)
			err = errors.New(
				noDestPort,
			)
			return err
		}

		arguments = append(
			arguments,
			rpc.Argument{
				Name:     rpc.RPC_DESTINATION_PORT,
				DataType: rpc.DataUint64,
				Value: tx.Address.Arguments.Value(
					rpc.RPC_DESTINATION_PORT,
					rpc.DataUint64).(uint64),
			},
		)

		fmt.Printf(addArgs)

		if tx.Address.Arguments.Has(
			rpc.RPC_EXPIRY,
			rpc.DataTime,
		) {

			if tx.Address.Arguments.Value(
				rpc.RPC_EXPIRY,
				rpc.DataTime).(time.Time).Before(time.Now().UTC()) {

				fmt.Printf(
					addressExpired,
					expirytime,
					tx.Address.Arguments.Value(
						rpc.RPC_EXPIRY,
						rpc.DataTime,
					),
				)
				err = errors.New(addressExpired)
				return err

			} else {

				fmt.Printf(
					addressWillExpire,
					expirytime,
					tx.Address.Arguments.Value(
						rpc.RPC_EXPIRY,
						rpc.DataTime,
					),
				)
				err = errors.New(addressExpired)
				return err

			}
		}

		fmt.Printf(
			destIntegrated,
			dstport,
			tx.Address.Arguments.Value(
				rpc.RPC_DESTINATION_PORT,
				rpc.DataUint64).(uint64),
		)

		if tx.Address.Arguments.Has(
			rpc.RPC_COMMENT,
			rpc.DataString,
		) {

			fmt.Printf(
				integratedMessage,
				stringcomment,
				tx.Address.Arguments.Value(
					rpc.RPC_COMMENT,
					rpc.DataString),
			)

			arguments = append(
				arguments,
				rpc.Argument{
					Name:     rpc.RPC_COMMENT,
					DataType: rpc.DataString,
					Value: tx.Address.Arguments.Value(
						rpc.RPC_COMMENT,
						rpc.DataString),
				},
			)
		}
	}

	fmt.Printf(checkingArgs)

	for _, arg := range tx.Address.Arguments {
		if !(arg.Name == rpc.RPC_COMMENT ||
			arg.Name == rpc.RPC_EXPIRY ||
			arg.Name == rpc.RPC_DESTINATION_PORT ||
			arg.Name == rpc.RPC_SOURCE_PORT ||
			arg.Name == rpc.RPC_VALUE_TRANSFER ||
			arg.Name == rpc.RPC_NEEDS_REPLYBACK_ADDRESS) {

			switch arg.DataType {
			case rpc.DataString:
				arguments = append(arguments,
					rpc.Argument{
						Name:     arg.Name,
						DataType: arg.DataType,
						Value:    arg.Value.(string),
					},
				)

			case rpc.DataInt64:
				arguments = append(arguments,
					rpc.Argument{
						Name:     arg.Name,
						DataType: arg.DataType,
						Value:    arg.Value.(string),
					},
				)

			case rpc.DataUint64:
				arguments = append(arguments,
					rpc.Argument{
						Name:     arg.Name,
						DataType: arg.DataType,
						Value:    arg.Value.(string),
					},
				)

			case rpc.DataFloat64:
				arguments = append(arguments,
					rpc.Argument{
						Name:     arg.Name,
						DataType: arg.DataType,
						Value:    arg.Value.(string),
					},
				)

			case rpc.DataTime:
				fmt.Errorf(timeUnsupported)
			}
		}
	}

	fmt.Printf(checkingAmount)

	if tx.Address.Arguments.Has(
		rpc.RPC_VALUE_TRANSFER,
		rpc.DataUint64) {

		fmt.Printf(
			txAmount,
			globals.FormatMoney(
				tx.Address.Arguments.Value(
					rpc.RPC_VALUE_TRANSFER,
					rpc.DataUint64).(uint64),
			),
		)
		tx.Amount = tx.Address.Arguments.Value(
			rpc.RPC_VALUE_TRANSFER,
			rpc.DataUint64).(uint64)

	} else {

		balance, _ := engram.Disk.Get_Balance()
		fmt.Printf(sendBalance, balance)
		fmt.Printf(sentAmount, tx.Amount)

		if tx.Amount > balance {
			fmt.Printf(sentInsufficentFunds)
			err = errors.New(sentInsufficentFunds)
			return err
		} else if tx.Amount == balance {
			tx.SendAll = true
		} else {
			tx.SendAll = false
		}
	}

	fmt.Printf(checkingServices)

	if tx.Address.Arguments.Has(
		rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
		rpc.DataUint64) {
		fmt.Printf(
			replyRequired,
			engram.Disk.GetAddress().String(),
		)
		arguments = append(
			arguments,
			rpc.Argument{
				Name:     rpc.RPC_REPLYBACK_ADDRESS,
				DataType: rpc.DataAddress,
				Value:    engram.Disk.GetAddress(),
			},
		)
	}

	fmt.Printf(checkingPayid)

	if len(arguments) == 0 {
		arguments = append(
			arguments,
			rpc.Argument{
				Name:     rpc.RPC_DESTINATION_PORT,
				DataType: rpc.DataUint64,
				Value:    tx.PaymentID,
			},
		)
		arguments = append(
			arguments,
			rpc.Argument{
				Name:     rpc.RPC_COMMENT,
				DataType: rpc.DataString,
				Value:    tx.Comment,
			},
		)
	}

	fmt.Printf(checkPack)

	if _, err := arguments.CheckPack(transaction.PAYLOAD0_LIMIT); err != nil {
		fmt.Printf(argPackErr, err)
		return err
	}

	if tx.Ringsize == 0 {
		tx.Ringsize = DEFAULT_SC_RINGSIZE
	} else if tx.Ringsize > DEFAULT_MAX_RINGSIZE {
		tx.Ringsize = DEFAULT_MAX_RINGSIZE
	} else if !crypto.IsPowerOf2(int(tx.Ringsize)) {
		tx.Ringsize = DEFAULT_SC_RINGSIZE
		fmt.Printf(
			invalidRings,
			tx.Ringsize,
		)
		err = errors.New(invalidRingsize)
		return err
	}

	tx.Status = stringUnsent

	fmt.Printf(ringSize, tx.Ringsize)

	tx.Pending = append(
		tx.Pending,
		rpc.Transfer{
			Amount:      tx.Amount,
			Destination: tx.Address.String(),
			Payload_RPC: arguments,
		},
	)
	fmt.Printf(addedTransfer)

	return nil
}

// Send all batched transfers (TODO: export offline transactions to file in Offline mode)
func sendAllTransfers() (txid crypto.Hash, err error) {
	if session.Offline {
		return
	}

	tx.TX, err = engram.Disk.TransferPayload0(
		tx.Pending,
		tx.Ringsize,
		false,
		rpc.Arguments{},
		0,
		false,
	)

	if err != nil {
		fmt.Printf(sendError, err)
		return
	}

	if err = engram.Disk.SendTransaction(tx.TX); err != nil {
		fmt.Printf(dispatchErr, err)
		return
	}

	tx.Fees = tx.TX.Fees()
	tx.TXID = tx.TX.GetHash()

	fmt.Printf(dispatchTx, tx.TXID)

	txid = tx.TX.GetHash()

	tx = Transfers{}

	return
}

// Go Routine for account registration
func registerAccount() {
	session.Domain = appRegister
	if engram.Disk == nil {
		resizeWindow(ui.MaxWidth, ui.MaxHeight)
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutMain())
		session.Domain = appMain
		return
	}

	link := widget.NewHyperlinkWithStyle(
		stringCancel,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)

	link.OnTapped = func() {
		session.Gif.Stop()
		session.Gif = nil
		closeWallet()
	}

	title := canvas.NewText(regBanner, colors.Green)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 16

	heading := canvas.NewText(plsWaitMsg, colors.Gray)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	sub := canvas.NewText(oneTimeMsg, colors.Gray)
	sub.TextSize = 14
	sub.Alignment = fyne.TextAlignCenter
	sub.TextStyle = fyne.TextStyle{Bold: true}

	resizeWindow(ui.MaxWidth, ui.MaxHeight)
	session.Window.SetContent(layoutTransition())
	session.Window.SetContent(
		layoutWaiting(
			title,
			heading,
			sub,
			link,
		),
	)

	// Registration PoW
	go func() {
		var reg_tx *transaction.Transaction
		successful_regs := make(chan *transaction.Transaction)
		counter := 0
		session.RegHashes = 0

		for i := 0; i < runtime.GOMAXPROCS(0)/2; i++ {
			go func() {
				for counter == 0 {
					if engram.Disk == nil {
						break
					} else if engram.Disk.IsRegistered() {
						break
					}

					lreg_tx := engram.Disk.GetRegistrationTX()
					hash := lreg_tx.GetHash()
					session.RegHashes++

					if hash[0] == 0 && hash[1] == 0 && hash[2] == 0 {
						successful_regs <- lreg_tx
						counter++
						break
					}
				}
			}()
		}

		if engram.Disk == nil {
			session.Gif.Stop()
			session.Gif = nil
			resizeWindow(ui.MaxWidth, ui.MaxHeight)
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutMain())
			session.Domain = appMain
			return
		}

		reg_tx = <-successful_regs

		fmt.Printf(
			userRegTxID,
			reg_tx.GetHash(),
		)
		err := engram.Disk.SendTransaction(reg_tx)
		if err != nil {
			session.Gif.Stop()
			session.Gif = nil
			fmt.Printf(regErr, err)
			resizeWindow(ui.MaxWidth, ui.MaxHeight)
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutMain())
			session.Domain = appMain
		} else {
			session.Gif.Stop()
			session.Gif = nil
			fmt.Printf(
				regSuccess,
			)
			resizeWindow(ui.MaxWidth, ui.MaxHeight)
			session.Domain = appWallet
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutDashboard())
		}
	}()

	return
}

// Set the ring size for transactions
func setRingSize(wallet *walletapi.Wallet_Disk, s int) bool {
	if wallet == nil {
		fmt.Printf(noWallet)
		return false
	}

	// Minimum ring size is 2, only accept powers of 2.
	if s < 2 {
		wallet.SetRingSize(2)
		fmt.Printf(setMinRingSize, 2)
	} else {
		wallet.SetRingSize(s)
		fmt.Printf(setMinRingSize, s)
	}

	return true
}

// Get the username from the graviton database
// Returns error if no results are found
func getUsername() (result string, err error) {
	username, err := GetValue(stringUsernames, byteusername)
	if err != nil {
		return
	}

	result = string(username)

	return
}

// Check if a username exists, return the registered address if so
func checkUsername(s string, h int64) (valid bool, address string, err error) {
	if session.Offline {
		valid = false
		return
	}
	var params rpc.NameToAddress_Params
	var response *jrpc2.Response
	var result rpc.NameToAddress_Result

	rpc_client.WS, _, err = websocket.DefaultDialer.Dial(
		"ws://"+session.Daemon+"/ws",
		nil,
	)

	input_output := rwc.New(rpc_client.WS)
	rpc_client.RPC = jrpc2.NewClient(
		channel.RawJSON(input_output, input_output),
		nil,
	)

	if rpc_client.RPC != nil {
		params.Name = s
		params.TopoHeight = -1

		valid = false
		address = string_
		response, err = rpc_client.RPC.Call(
			context.Background(),
			DAEMON_NAME_TO_ADDRESS,
			params,
		)

		rpc_client.WS.Close()
		rpc_client.RPC.Close()

		if err != nil {
			return
		}

		err = response.UnmarshalResult(&result)
		if err != nil {
			return
		}

		if result.Status != stringOK {
			err = errors.New(userDoesntExist)
			return
		}

		valid = true
		address = result.Address
	}

	return
}

// Get the transaction fees to be paid
func getGasEstimate(gp rpc.GasEstimate_Params) (gas uint64, err error) {
	var result rpc.GasEstimate_Result

	rpc_client.WS, _, err = websocket.DefaultDialer.Dial(
		"ws://"+session.Daemon+"/ws",
		nil,
	)
	if err != nil {
		return
	}

	input_output := rwc.New(rpc_client.WS)
	rpc_client.RPC = jrpc2.NewClient(
		channel.RawJSON(input_output, input_output),
		nil,
	)

	if err = rpc_client.RPC.CallResult(
		context.Background(),
		DAEMON_GET_GAS_ESTIMATE,
		gp,
		&result,
	); err != nil {
		return
	}

	if result.Status != stringOK {
		return
	}

	gas = result.GasStorage

	return
}

// Register a new DERO username
func registerUsername(s string) (err error) {
	// Check first if the name is taken
	valid, _, _ := checkUsername(s, -1)
	if valid {
		fmt.Printf(
			userErr,
		)
		err = errors.New(userExists)
		return
	}

	var scid crypto.Hash
	scid = crypto.HashHexToHash(DERO_NAME_SERVICE_SCID)

	var args = rpc.Arguments{}
	args = append(
		args,
		rpc.Argument{
			Name:     DEFAULT_SC_ENTRYPOINT,
			DataType: rpc.DataString,
			Value:    REGISTER_STRING,
		},
	)
	args = append(
		args,
		rpc.Argument{
			Name:     rpc.SCID,
			DataType: rpc.DataHash,
			Value:    scid,
		},
	)
	args = append(
		args,
		rpc.Argument{
			Name:     rpc.SCACTION,
			DataType: rpc.DataUint64,
			Value:    uint64(rpc.SC_CALL),
		},
	)
	args = append(
		args,
		rpc.Argument{
			Name:     stringname,
			DataType: rpc.DataString,
			Value:    s,
		},
	)

	var p rpc.Transfer_Params
	var dest string

	if !session.Testnet {
		dest = DERO_DEVELOPER_MAINNET_ADDRESS
	} else {
		dest = DERO_DEVELOPER_TESTNET_ADDRESS
	}
	p.Transfers = append(
		p.Transfers,
		rpc.Transfer{
			Destination: dest,
			Amount:      0,
			Burn:        0,
		},
	)

	gp := rpc.GasEstimate_Params{
		SC_RPC:    args,
		Ringsize:  2,
		Signer:    engram.Disk.GetAddress().String(),
		Transfers: p.Transfers,
	}

	storage, err := getGasEstimate(gp)
	if err != nil {
		fmt.Printf(usrError, err)
		return
	}

	tx, err := engram.Disk.TransferPayload0(
		p.Transfers,
		2,
		false,
		args,
		storage,
		false,
	)

	if err != nil {
		fmt.Printf(usrError, err)
		return
	}

	err = engram.Disk.SendTransaction(tx)
	if err != nil {
		fmt.Printf(usrError, err)
		return
	}

	fmt.Printf(
		userRegTxID,
		tx.GetHash().String(),
	)

	return
}

// Check to make sure the message transaction meets criteria
func checkMessagePack(m string, s string, r string) (err error) {
	if m == string_ {
		return
	}

	mapAddress := string_
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

	if s == string_ {
		s = engram.Disk.GetAddress().String()
	}

	amount, err := globals.ParseAmount(twoderi)
	if err != nil {
		//logger.Error(err, "Err parsing amount")
		return
	}

	var arguments = rpc.Arguments{
		{
			Name:     rpc.RPC_DESTINATION_PORT,
			DataType: rpc.DataUint64,
			Value:    destPort,
		},
		{
			Name:     rpc.RPC_VALUE_TRANSFER,
			DataType: rpc.DataUint64,
			Value:    amount,
		},
		{
			Name:     rpc.RPC_COMMENT,
			DataType: rpc.DataString,
			Value:    m,
		},
		{
			Name:     rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
			DataType: rpc.DataString,
			Value:    s,
		},
	}

	if a.IsIntegratedAddress() { // read everything from the address

		if a.Arguments.Validate_Arguments() != nil {
			//fmt.Printf(err, "Integrated Address  arguments could not be validated.\n")
			return
		}

		if !a.Arguments.Has(rpc.RPC_DESTINATION_PORT, rpc.DataUint64) { // but only it is present
			//fmt.Printf(fmt.Errorf("Integrated Address does not contain destination port.\n"), string_)
			return
		}

		arguments = append(
			arguments,
			rpc.Argument{
				Name:     rpc.RPC_DESTINATION_PORT,
				DataType: rpc.DataUint64,
				Value: a.Arguments.Value(
					rpc.RPC_DESTINATION_PORT,
					rpc.DataUint64,
				).(uint64),
			},
		)

		if a.Arguments.Has(rpc.RPC_EXPIRY, rpc.DataTime) { // but only it is present

			if a.Arguments.Value(
				rpc.RPC_EXPIRY,
				rpc.DataTime).(time.Time).
				Before(
					time.
						Now().
						UTC(),
				) {
				//fmt.Printf(nil, "This address has expired.", "expiry time", a.Arguments.Value(rpc.RPC_EXPIRY, rpc.DataTime))
				return
			} else {
				//fmt.Printf("This address will expire ", "expiry time", a.Arguments.Value(rpc.RPC_EXPIRY, rpc.DataTime))
			}
		}

		fmt.Printf(
			destinationIntegrated,
			a.Arguments.Value(
				rpc.RPC_DESTINATION_PORT,
				rpc.DataUint64).(uint64),
		)

		if a.Arguments.Has(rpc.RPC_COMMENT, rpc.DataString) { // but only it is present
			fmt.Printf(
				integratedMsg,
				a.Arguments.Value(
					rpc.RPC_COMMENT,
					rpc.DataString,
				),
			)

			arguments = append(
				arguments,
				rpc.Argument{
					Name:     rpc.RPC_COMMENT,
					DataType: rpc.DataString,
					Value: a.Arguments.Value(
						rpc.RPC_COMMENT,
						rpc.DataString,
					),
				},
			)
		}
	}

	for _, arg := range arguments {
		if !(arg.Name == rpc.RPC_COMMENT ||
			arg.Name == rpc.RPC_EXPIRY ||
			arg.Name == rpc.RPC_DESTINATION_PORT ||
			arg.Name == rpc.RPC_SOURCE_PORT ||
			arg.Name == rpc.RPC_VALUE_TRANSFER ||
			arg.Name == rpc.RPC_NEEDS_REPLYBACK_ADDRESS) {
			switch arg.DataType {
			case rpc.DataString:
				arguments = append(
					arguments,
					rpc.Argument{
						Name:     arg.Name,
						DataType: arg.DataType,
						Value:    arg.Value.(string),
					},
				)
			case rpc.DataInt64:
				arguments = append(
					arguments,
					rpc.Argument{
						Name:     arg.Name,
						DataType: arg.DataType,
						Value:    arg.Value.(string),
					},
				)
			case rpc.DataUint64:
				arguments = append(
					arguments,
					rpc.Argument{
						Name:     arg.Name,
						DataType: arg.DataType,
						Value:    arg.Value.(string),
					},
				)
			case rpc.DataFloat64:
				arguments = append(
					arguments,
					rpc.Argument{
						Name:     arg.Name,
						DataType: arg.DataType,
						Value:    arg.Value.(string),
					},
				)
			case rpc.DataTime:
				fmt.Errorf(timeUnsupported)
			}
		}
	}

	if a.Arguments.Has(
		rpc.RPC_VALUE_TRANSFER,
		rpc.DataUint64,
	) { // but only it is present
		//logger.Info("Transaction", "Value", globals.FormatMoney(a.Arguments.Value(rpc.RPC_VALUE_TRANSFER, rpc.DataUint64).(uint64)))
		amount = a.Arguments.Value(
			rpc.RPC_VALUE_TRANSFER,
			rpc.DataUint64,
		).(uint64)
	} else {
		amount, err = globals.ParseAmount(onederi)
		if err != nil {
			//logger.Error(err, "Err parsing amount")
			return
		}
	}

	if a.Arguments.Has(
		rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
		rpc.DataString,
	) {
		arguments = append(
			arguments,
			rpc.Argument{
				Name:     rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
				DataType: rpc.DataString,
				Value:    s,
			},
		)
	}

	// if no arguments, use space by embedding a small comment
	if len(arguments) == 0 { // allow user to enter Comment
		arguments = append(
			arguments,
			rpc.Argument{
				Name:     rpc.RPC_DESTINATION_PORT,
				DataType: rpc.DataUint64,
				Value:    destPort,
			},
		)
		arguments = append(
			arguments,
			rpc.Argument{
				Name:     rpc.RPC_COMMENT,
				DataType: rpc.DataString,
				Value:    m,
			},
		)
	}

	if _, err = arguments.CheckPack(transaction.PAYLOAD0_LIMIT); err != nil {
		fmt.Printf(packErr, err)
		return
	}

	return
}

// Send a private message to another account
func sendMessage(m string, s string, r string) (txid crypto.Hash, err error) {
	if m == string_ {
		return
	}

	mapAddress := string_
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

	if s == string_ {
		s = engram.Disk.GetAddress().String()
	}

	amount, err := globals.ParseAmount(onederi)
	if err != nil {
		//logger.Error(err, "Err parsing amount")
		return
	}

	var arguments = rpc.Arguments{
		{
			Name:     rpc.RPC_DESTINATION_PORT,
			DataType: rpc.DataUint64,
			Value:    destPort,
		},
		{
			Name:     rpc.RPC_VALUE_TRANSFER,
			DataType: rpc.DataUint64,
			Value:    amount,
		},
		{
			Name:     rpc.RPC_EXPIRY,
			DataType: rpc.DataTime,
			Value:    time.Now().Add(time.Hour).UTC(),
		},
		{
			Name:     rpc.RPC_COMMENT,
			DataType: rpc.DataString,
			Value:    m,
		},
		{
			Name:     rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
			DataType: rpc.DataString,
			Value:    s,
		},
	}

	if a.IsIntegratedAddress() {
		if a.Arguments.Validate_Arguments() != nil {
			return
		}

		if !a.Arguments.Has(
			rpc.RPC_DESTINATION_PORT,
			rpc.DataUint64,
		) {
			fmt.Printf(
				integratedAddresWithoutDest,
			)
			return
		}

		arguments = append(
			arguments,
			rpc.Argument{
				Name:     rpc.RPC_DESTINATION_PORT,
				DataType: rpc.DataUint64,
				Value: a.Arguments.Value(
					rpc.RPC_DESTINATION_PORT,
					rpc.DataUint64,
				).(uint64),
			},
		)

		if a.Arguments.Has(
			rpc.RPC_EXPIRY,
			rpc.DataTime,
		) {
			if a.Arguments.Value(
				rpc.RPC_EXPIRY,
				rpc.DataTime,
			).(time.Time).
				Before(
					time.
						Now().
						UTC(),
				) {
				fmt.Printf(
					addressExpiredOn,
					a.Arguments.Value(
						rpc.RPC_EXPIRY,
						rpc.DataTime,
					),
				)

				return
			} else {
				fmt.Printf(
					addressDoesExpireOn,
					a.Arguments.Value(
						rpc.RPC_EXPIRY,
						rpc.DataTime,
					),
				)
			}
		}

		fmt.Printf(
			integratedAddresWithDest,
			a.Arguments.Value(
				rpc.RPC_DESTINATION_PORT,
				rpc.DataUint64).(uint64),
		)

		if a.Arguments.Has(
			rpc.RPC_COMMENT,
			rpc.DataString,
		) {
			fmt.Printf(
				integratedMessageIs,
				a.Arguments.Value(
					rpc.RPC_COMMENT,
					rpc.DataString,
				),
			)
			arguments = append(
				arguments,
				rpc.Argument{
					Name:     rpc.RPC_COMMENT,
					DataType: rpc.DataString,
					Value: a.Arguments.Value(
						rpc.RPC_COMMENT,
						rpc.DataString,
					),
				},
			)
		}
	}

	for _, arg := range arguments {
		if !(arg.Name == rpc.RPC_COMMENT ||
			arg.Name == rpc.RPC_EXPIRY ||
			arg.Name == rpc.RPC_DESTINATION_PORT ||
			arg.Name == rpc.RPC_SOURCE_PORT ||
			arg.Name == rpc.RPC_VALUE_TRANSFER ||
			arg.Name == rpc.RPC_NEEDS_REPLYBACK_ADDRESS) {
			switch arg.DataType {
			case rpc.DataString:
				arguments = append(
					arguments,
					rpc.Argument{
						Name:     arg.Name,
						DataType: arg.DataType,
						Value:    arg.Value.(string),
					},
				)
			case rpc.DataInt64:
				arguments = append(
					arguments,
					rpc.Argument{
						Name:     arg.Name,
						DataType: arg.DataType,
						Value:    arg.Value.(string),
					},
				)
			case rpc.DataUint64:
				arguments = append(
					arguments,
					rpc.Argument{
						Name:     arg.Name,
						DataType: arg.DataType,
						Value:    arg.Value.(string),
					},
				)
			case rpc.DataFloat64:
				arguments = append(
					arguments,
					rpc.Argument{
						Name:     arg.Name,
						DataType: arg.DataType,
						Value:    arg.Value.(string),
					},
				)
			case rpc.DataTime:
				fmt.Printf(timeUnsupported)
			}
		}
	}

	if a.Arguments.Has(
		rpc.RPC_VALUE_TRANSFER,
		rpc.DataUint64,
	) {
		amount = a.Arguments.Value(
			rpc.RPC_VALUE_TRANSFER,
			rpc.DataUint64,
		).(uint64)
	} else {
		amount, err = globals.ParseAmount(onederi)
		if err != nil {
			fmt.Printf(sendErr, err)
			return
		}
	}

	if a.Arguments.Has(
		rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
		rpc.DataString,
	) {
		arguments = append(
			arguments,
			rpc.Argument{
				Name:     rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
				DataType: rpc.DataString,
				Value:    s,
			},
		)
	}

	if len(arguments) == 0 {
		arguments = append(
			arguments,
			rpc.Argument{
				Name:     rpc.RPC_DESTINATION_PORT,
				DataType: rpc.DataUint64,
				Value:    destPort,
			},
		)
		arguments = append(
			arguments,
			rpc.Argument{
				Name:     rpc.RPC_COMMENT,
				DataType: rpc.DataString,
				Value:    m,
			},
		)
	}

	if _, err = arguments.CheckPack(transaction.PAYLOAD0_LIMIT); err != nil {
		fmt.Printf(
			packErr,
			err,
		)
		return
	}

	fees := ((uint64(engram.Disk.GetRingSize()) + 1) * config.FEE_PER_KB) / 4

	fmt.Printf(calcFees, fees)

	tx, err := engram.Disk.TransferPayload0(
		[]rpc.Transfer{
			{
				Amount:      amount,
				Destination: a.String(),
				Payload_RPC: arguments,
			},
		},
		0,
		false,
		rpc.Arguments{},
		fees,
		false,
	)
	if err != nil {

		fmt.Printf(
			errBuilding,
			err,
		)

		return
	}

	if err = engram.Disk.SendTransaction(tx); err != nil {

		fmt.Printf(
			errDispatching,
			err,
		)

		return
	}

	txid = tx.GetHash()

	fmt.Printf(
		dispatchedTx,
		txid,
	)

	return
}

// Get a list of message transactions from an address
func getMessagesFromUser(s string, h uint64) (result []rpc.Entry) {
	var zeroscid crypto.Hash
	if s == string_ {
		return
	}

	messages := engram.Disk.Get_Payments_DestinationPort(
		zeroscid,
		destPort,
		h,
	)

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
			if tx.Payload_RPC.HasValue(
				rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
				rpc.DataString,
			) {
				height := int64(tx.Height)
				_, check2, err := checkUsername(
					tx.Payload_RPC.Value(
						rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
						rpc.DataString).(string),
					height,
				)
				if err != nil {
					username2 = false
					addr, err := globals.ParseValidateAddress(
						tx.Payload_RPC.Value(
							rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
							rpc.DataString).(string),
					)
					if err != nil {
						check2 = string_
					} else {
						check2 = addr.String()
					}
				} else {
					username2 = true
				}

				// Check for spoofing
				//if ring_member_exists(txid, check2) {

				if username && username2 {
					if check == check2 {
						result = append(
							result,
							messages[m],
						)
					}
				} else if !username && !username2 {
					if s == tx.Payload_RPC.Value(
						rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
						rpc.DataString).(string) {
						result = append(
							result,
							messages[m],
						)
					}
				} else if check == tx.Payload_RPC.Value(
					rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
					rpc.DataString).(string) {
					result = append(
						result,
						messages[m],
					)
				} else if s == check2 {
					result = append(
						result,
						messages[m],
					)
				}
				//}
			}
		} else {
			addr, err := engram.Disk.NameToAddress(s)
			if err != nil {
				if tx.Destination == s {
					result = append(
						result,
						messages[m],
					)
				}
			} else {
				if tx.Destination == addr {
					result = append(
						result,
						messages[m],
					)
				}
			}
		}
	}

	return
}

// Get a list of all message transactions and sort them by address
func getMessages(h uint64) (result []string) {
	var zeroscid crypto.Hash
	messages := engram.Disk.Get_Payments_DestinationPort(
		zeroscid,
		destPort,
		h,
	)

	for m := range messages {
		if messages[m].Incoming {
			if messages[m].Payload_RPC.HasValue(
				rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
				rpc.DataString,
			) {
				if messages[m].Payload_RPC.Value(
					rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
					rpc.DataString).(string) == string_ {

				} else {
					height := int64(messages[m].Height)
					valid, sender, _ := checkUsername(
						messages[m].Payload_RPC.Value(
							rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
							rpc.DataString).(string),
						height,
					)
					if !valid {
						addr, err := globals.ParseValidateAddress(
							messages[m].Payload_RPC.Value(
								rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
								rpc.DataString).(string),
						)
						if err != nil {

						} else {
							sender = addr.String()
							for r := range result {
								if r > -1 && r < len(result) {
									if strings.Contains(
										result[r],
										sender+threetilde,
									) {
										copy(
											result[r:],
											result[r+1:],
										)
										result[len(result)-1] = string_
										result = result[:len(result)-1]
									}
								}
							}
							result = append(
								result,
								sender+threetilde,
							)
						}
					} else {
						// Check for spoofing
						//if ring_member_exists(messages[m].TXID, sender) {
						for r := range result {
							if r > -1 && r < len(result) {
								//if strings.Contains(result[r], sender+"```"+messages[m].Payload_RPC.Value(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString).(string)) {
								if strings.Contains(
									result[r],
									sender+threetilde,
								) {
									copy(
										result[r:],
										result[r+1:],
									)
									result[len(result)-1] = string_
									result = result[:len(result)-1]
								}
							}
						}
						result = append(
							result,
							sender+threetilde+messages[m].Payload_RPC.Value(
								rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
								rpc.DataString).(string),
						)
						//} else {
						// TODO: Add spoofing address to the ban list?
						//}
					}
				}
			}
		} else {
			if messages[m].Payload_RPC.HasValue(
				rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
				rpc.DataString,
			) {
				uname := string_
				for r := range result {
					if r > -1 && r < len(result) {
						if strings.Contains(
							result[r],
							messages[m].Destination+threetilde,
						) {
							split := strings.Split(
								result[r],
								threetilde,
							)
							uname = split[1]
							copy(
								result[r:],
								result[r+1:],
							)
							result[len(result)-1] = string_
							result = result[:len(result)-1]
						}
					}
				}
				result = append(
					result,
					messages[m].Destination+threetilde+uname,
				)
			}
		}
	}

	sort.Sort(
		sort.Reverse(
			sort.StringSlice(
				result,
			),
		),
	)
	return
}

// Returns a list of registered usernames from Gnomon
func queryUsernames() (result []string, err error) {
	if gnomon.Index != nil && engram.Disk != nil {
		result, _ = gnomon.Graviton.GetSCIDKeysByValue(
			DERO_NAME_SERVICE_SCID,
			engram.Disk.GetAddress().String(),
			engram.Disk.Get_Daemon_TopoHeight(),
			false,
		)
		if len(result) <= 0 {
			result, _, err = gnomon.Index.GetSCIDKeysByValue(
				nil, DERO_NAME_SERVICE_SCID,
				engram.Disk.GetAddress().String(),
				engram.Disk.Get_Daemon_TopoHeight(),
			)
			if err != nil {
				fmt.Printf(
					gnomonFailure,
					err,
				)
				return
			}
		}

		sort.Sort(
			sort.StringSlice(
				result,
			),
		)
	}

	return
}

// Get the local list of registered usernames saved from previous Gnomon scans
func getUsernames() (result []string, err error) {
	usernames, err := GetEncryptedValue(
		stringUsernames,
		byteusernames,
	)
	if err != nil {
		return
	}

	result = strings.Split(
		string(usernames),
		",",
	)
	return
}

// Set the Primary Username saved to a wallet's datashard
func setPrimaryUsername(s string) (err error) {
	err = StoreEncryptedValue(
		stringsettings,
		byteusername,
		[]byte(s),
	)
	return
}

// Get the Primary Username saved to a wallet's datashard
func getPrimaryUsername() (err error) {
	u, err := GetEncryptedValue(
		stringsettings,
		byteusername,
	)
	if err != nil {
		session.Username = string_
		return
	}
	session.Username = string(u)
	return
}

// Returns a list of SCIDs that a wallet interacted with from Gnomon
func queryAssets() (result []string, err error) {
	if gnomon.Active == 1 && engram.Disk != nil {
		gnomon.BBolt.DBPath = filepath.Join(
			AppPath(),
			stringdatashards,
			stringgnomon,
		)
		if session.Testnet {
			gnomon.BBolt.DBPath = filepath.Join(
				AppPath(),
				stringdatashards,
				stringgnomonTestnet,
			)
		}
		result = gnomon.BBolt.GetSCIDInteractionByAddr(
			engram.Disk.GetAddress().String(),
		)
	}

	return
}

// Get the local path to a smart contract file (Ex: contract.bas)
func prepareSC(path string) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Printf("%s", err)
		return
	}

	defer file.Close()
}

// Start the Gnomon indexer
func startGnomon() {
	if walletapi.Connected {
		if gnomon.Index == nil && gnomon.Active == 1 {
			path := filepath.Join(
				AppPath(),
				stringdatashards,
				stringgnomon,
			)
			if session.Testnet {
				path = filepath.Join(
					AppPath(),
					stringdatashards,
					stringgnomonTestnet,
				)
			}
			gnomon.BBolt, _ = storage.NewBBoltDB(
				path,
				stringgnomon,
			)
			gnomon.Graviton, _ = storage.NewGravDB(
				path,
				"25ms",
			)
			term := []string(nil)
			term = append(
				term,
				functionInit,
			)
			height, err := gnomon.Graviton.GetLastIndexHeight()
			if err != nil {
				height = 0
			}

			// Fastsync Config
			config := &structures.FastSyncConfig{
				Enabled:           true,
				SkipFSRecheck:     true,
				ForceFastSync:     true,
				ForceFastSyncDiff: 20,
				NoCode:            true,
			}

			// exclude the Gnomon SC, etc. to keep faster sync times
			var exclusions []string

			gnomon.Index = indexer.NewIndexer(
				gnomon.Graviton,
				gnomon.BBolt,
				stringgravdb,
				term,
				height,
				session.Daemon,
				stringdaemon,
				false,
				true,
				config,
				exclusions,
			)
			indexer.InitLog(
				globals.Arguments,
				os.Stdout,
			)

			// We can allow parallel processing of x blocks at a time
			go gnomon.Index.StartDaemonMode(1)

			fmt.Printf(
				gnomonScan,
				height,
				gnomon.Index.LastIndexedHeight,
			)
		}
	}
}

// Stop all indexers and close Gnomon
func stopGnomon() {
	if gnomon.Index != nil {
		gnomon.Index.Close()
		gnomon.Index = nil
		fmt.Printf(gnomonClosed)
	}
}

// Get the current state of all variables in a smart contract
func getContractVars(scid string) (vars map[string]interface{}, err error) {
	var params = rpc.GetSC_Params{
		SCID:      scid,
		Variables: true,
		Code:      false,
	}
	var result rpc.GetSC_Result

	rpc_client.WS, _, err = websocket.DefaultDialer.Dial(
		"ws://"+session.Daemon+"/ws",
		nil,
	)
	if err != nil {
		return
	}

	input_output := rwc.New(rpc_client.WS)
	rpc_client.RPC = jrpc2.NewClient(
		channel.RawJSON(input_output, input_output),
		nil,
	)

	err = rpc_client.RPC.CallResult(context.Background(), DAEMON_GET_SC, params, &result)
	if err != nil {
		fmt.Printf(
			errSCVars,
			err,
		)
		return
	}

	vars = result.VariableStringKeys

	return
}

// Install a new smart contract
func installSC(path string) (result string, err error) {
	data := &InstallContract{}
	file, err := os.Open(path)
	if err != nil {
		fmt.Printf("%s", err)
		return
	}

	defer file.Close()

	url := fmt.Sprintf(
		endpointInstallSC,
		DEFAULT_LOCAL_IP_ADDRESS,
		DEFAULT_WALLET_PORT,
	)
	req, err := http.NewRequest(POST_STRING, url, file)
	if err != nil {
		fmt.Printf("%s", err)
		return
	}

	req.Header.Set(CONTENT_TYPE_STRING, APP_OCTET_STREAM_STRING)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("%s", err)
		return
	}

	responseData, err := io.ReadAll(resp.Body)

	err = json.Unmarshal([]byte(responseData), &data)
	if err != nil {
		fmt.Printf("%s", err)
		return
	}

	defer resp.Body.Close()

	result = data.TXID

	return
}

// Set the Cyberdeck password
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

// Set the Cyberdeck username
func newRPCUsername() (s string) {
	r, _ := rand.Int(
		rand.Reader,
		big.NewInt(1600),
	)
	w := mnemonics.Key_To_Words(
		r,
		stringenglish,
	)
	l := strings.Split(
		string(w),
		" ",
	)
	s = l[len(l)-2]
	cyberdeck.user = s
	return
}

// Start an RPC server to allow decentralized application communication (TODO: Replace with or add permissioned websockets?)
func toggleCyberdeck() {
	var err error
	if engram.Disk == nil {
		return
	}

	if cyberdeck.server != nil {
		cyberdeck.server.RPCServer_Stop()
		cyberdeck.server = nil
		cyberdeck.status.Text = stringBlocked
		cyberdeck.status.Color = colors.Gray
		cyberdeck.status.Refresh()
		cyberdeck.toggle.Text = turnOn
		cyberdeck.toggle.Refresh()
		status.Cyberdeck.FillColor = colors.Gray
		status.Cyberdeck.StrokeColor = colors.Gray
		status.Cyberdeck.Refresh()
		cyberdeck.userText.Text = cyberdeck.user
		cyberdeck.passText.Text = cyberdeck.pass
		cyberdeck.userText.Enable()
		cyberdeck.passText.Enable()
	} else {
		if session.Testnet {
			globals.Arguments[stringFlagrpcbind] = DEFAULT_LOCAL_TESTNET_WALLET_RPC
		} else {
			globals.Arguments[stringFlagrpcbind] = DEFAULT_LOCAL_WALLET_RPC
		}

		if cyberdeck.user == string_ {
			cyberdeck.user = newRPCUsername()
		}

		if cyberdeck.pass == string_ {
			cyberdeck.pass = newRPCPassword()
		}

		globals.Arguments[stringFlagrpclogin] = cyberdeck.user + singlecolon + cyberdeck.pass

		cyberdeck.server, err = rpcserver.RPCServer_Start(engram.Disk, stringCyberdeck)
		if err != nil {
			cyberdeck.server = nil
			cyberdeck.status.Text = stringBlocked
			cyberdeck.status.Color = colors.Gray
			cyberdeck.status.Refresh()
			cyberdeck.toggle.Text = turnOn
			cyberdeck.toggle.Refresh()
			status.Cyberdeck.FillColor = colors.Gray
			status.Cyberdeck.StrokeColor = colors.Gray
			status.Cyberdeck.Refresh()
			cyberdeck.userText.Text = cyberdeck.user
			cyberdeck.passText.Text = cyberdeck.pass
			cyberdeck.userText.Enable()
			cyberdeck.passText.Enable()
		} else {
			cyberdeck.status.Text = stringAllowed
			cyberdeck.status.Color = colors.Green
			cyberdeck.status.Refresh()
			cyberdeck.toggle.Text = turnOff
			cyberdeck.toggle.Refresh()
			status.Cyberdeck.FillColor = colors.Green
			status.Cyberdeck.StrokeColor = colors.Green
			status.Cyberdeck.Refresh()
			cyberdeck.userText.Text = cyberdeck.user
			cyberdeck.passText.Text = cyberdeck.pass
			cyberdeck.userText.Disable()
			cyberdeck.passText.Disable()
		}
	}
}

// Convert DERO value to USD (TODO: rework this and support other currencies)
func convertBalance() {
	if !session.Offline {
		if session.BalanceUSDText == nil {
			session.BalanceUSDText = canvas.NewText(string_, colors.Gray)
		}

		check, _ := engram.Disk.Get_Balance()
		if check == 0 {
			session.BalanceUSDText.Text = usdZero
			session.BalanceUSDText.Refresh()
			return
		}

		url := priceURL
		req, err := http.NewRequest(GET_STRING, url, nil)
		if err != nil {
			fmt.Printf("%s", err)
			session.BalanceUSDText.Text = string_
			session.BalanceUSDText.Refresh()
			return
		}
		req.Header.Set(CONTENT_TYPE_STRING, APP_JSON_STRING)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf(engramMsg, err)
			session.BalanceUSDText.Text = string_
			session.BalanceUSDText.Refresh()
			return
		}
		defer resp.Body.Close()

		var resData map[string]interface{}
		responseData, err := io.ReadAll(resp.Body)

		err = json.Unmarshal(responseData, &resData)
		if err != nil {
			fmt.Printf(engramMsg, err)
			session.BalanceUSDText.Text = string_
			session.BalanceUSDText.Refresh()
			return
		}

		defer resp.Body.Close()

		if resData["dero"] == nil {
			err = errors.New(errPriceQuote)
			session.BalanceUSDText.Text = string_
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
		session.BalanceUSDText.Text = usdWithPad + formatted
		session.BalanceUSDText.Refresh()
		fmt.Printf(valueUpdated)
	} else {
		session.BalanceUSD = valueBlank
		session.BalanceUSDText.Text = usdWithPad + valueBlank
		session.BalanceUSDText.Refresh()
	}
}

// Get the latest smart contract header data (must follow the standard here: https://github.com/civilware/artificer-nfa-standard/blob/main/Headers/README.md)
func getContractHeader(scid crypto.Hash) (name string, desc string, icon string, owner string, code string) {
	var headerData []*structures.SCIDVariable
	var found bool

	headerData = gnomon.Index.GravDBBackend.GetAllSCIDVariableDetails(scid.String())

	if headerData == nil {
		addIndex := make(map[string]*structures.FastSyncImport)
		addIndex[scid.String()] = &structures.FastSyncImport{}
		gnomon.Index.AddSCIDToIndex(addIndex, false, true)
		headerData = gnomon.Index.GravDBBackend.GetAllSCIDVariableDetails(scid.String())
	}

	for _, h := range headerData {
		switch key := h.Key.(type) {
		case string:
			if key == "nameHdr" {
				found = true
				name = h.Value.(string)
			}

			if key == "descrHdr" {
				found = true
				desc = h.Value.(string)
			}

			if key == "iconURLHdr" {
				found = true
				icon = h.Value.(string)
			}

			if key == "owner" {
				owner = h.Value.(string)
			}

			if key == "C" {
				code = h.Value.(string)
			}
		}
	}

	// Secondary check for headers in Gnomon SC
	if !found {
		headerData = gnomon.Index.GravDBBackend.GetAllSCIDVariableDetails(structures.MAINNET_GNOMON_SCID)
		if headerData == nil {
			addIndex := make(map[string]*structures.FastSyncImport)
			addIndex[structures.MAINNET_GNOMON_SCID] = &structures.FastSyncImport{}
			gnomon.Index.AddSCIDToIndex(addIndex, false, true)
			headerData = gnomon.Index.GravDBBackend.GetAllSCIDVariableDetails(structures.MAINNET_GNOMON_SCID)
		}

		for _, h := range headerData {
			if strings.Contains(h.Key.(string), scid.String()) {
				switch key := h.Key.(type) {
				case string:
					if key == scid.String() {
						query := h.Value.(string)
						header := strings.Split(query, singlecolon)

						if len(header) > 2 {
							name = header[0]
							desc = header[1]
							icon = header[2]
						}
					}

					if key == scid.String()+OWNER_STRING {
						owner = h.Value.(string)
					}
				}
			}
		}
	}

	return
}

// Send an asset from one account to another
func transferAsset(scid crypto.Hash, address string, amount string) (txid crypto.Hash, err error) {
	var amount_to_transfer uint64

	if amount == string_ {
		amount = onederi
	}

	amount_to_transfer, err = globals.ParseAmount(amount)
	if err != nil {
		fmt.Printf(failedParse, err)
		return
	}

	tx, err := engram.Disk.TransferPayload0(
		[]rpc.Transfer{
			{
				SCID:        scid,
				Amount:      amount_to_transfer,
				Destination: address,
			},
		},
		0,
		false,
		rpc.Arguments{},
		0,
		false,
	)
	if err != nil {
		fmt.Printf(
			failedBuild,
			err,
		)
		return
	}

	if err = engram.Disk.SendTransaction(tx); err != nil {
		fmt.Printf(
			failedSend,
			scid,
			err,
		)
		return
	}

	txid = tx.GetHash()

	fmt.Printf(
		successSend,
		scid,
		tx.GetHash().String(),
	)
	return
}

// Transfer a username to another account
func transferUsername(username string, address string) (err error) {
	var args = rpc.Arguments{}
	var dest string

	scid := crypto.HashHexToHash(DERO_NAME_SERVICE_SCID)

	args = append(
		args,
		rpc.Argument{
			Name:     DEFAULT_SC_ENTRYPOINT,
			DataType: rpc.DataString,
			Value:    TRANSFER_OWNERSHIP_STRING,
		},
	)
	args = append(
		args,
		rpc.Argument{
			Name:     rpc.SCID,
			DataType: rpc.DataHash,
			Value:    scid,
		},
	)
	args = append(
		args,
		rpc.Argument{
			Name:     rpc.SCACTION,
			DataType: rpc.DataUint64,
			Value:    uint64(rpc.SC_CALL),
		},
	)
	args = append(
		args,
		rpc.Argument{
			Name:     NAME_STRING,
			DataType: rpc.DataString,
			Value:    username,
		},
	)
	args = append(
		args,
		rpc.Argument{
			Name:     NEWOWNER_STRING,
			DataType: rpc.DataString,
			Value:    address,
		},
	)

	if !session.Testnet {
		dest = DERO_DEVELOPER_MAINNET_ADDRESS
	} else {
		if globals.IsSimulator() {
			dest = DERO_DEVELOPER_SIMULATOR_ADDRESS
		} else {
			dest = DERO_DEVELOPER_TESTNET_ADDRESS

		}
	}

	transfer := rpc.Transfer{
		Destination: dest,
		Amount:      0,
		Burn:        0,
	}

	gasParams := rpc.GasEstimate_Params{
		SC_RPC:    args,
		SC_Value:  0,
		Ringsize:  DEFAULT_SC_RINGSIZE,
		Signer:    engram.Disk.GetAddress().String(),
		Transfers: []rpc.Transfer{transfer},
	}

	storage, err := getGasEstimate(gasParams)
	if err != nil {
		fmt.Printf(
			errGasEstimate,
			TRANSFER_OWNERSHIP_STRING,
			err,
		)
		return
	}

	tx, err := engram.Disk.TransferPayload0(
		[]rpc.Transfer{transfer},
		2,
		false,
		args,
		storage,
		false,
	)
	if err != nil {
		fmt.Printf(
			errBuildTx,
			TRANSFER_OWNERSHIP_STRING,
			err,
		)
		return
	}

	txid := tx.GetHash().String()

	err = engram.Disk.SendTransaction(tx)
	if err != nil {
		fmt.Printf(
			errSendTx,
			TRANSFER_OWNERSHIP_STRING,
			err,
		)
		return
	}

	walletapi.WaitNewHeightBlock()
	fmt.Printf(
		successSendTx,
		TRANSFER_OWNERSHIP_STRING,
		txid,
	)
	_ = tx

	return
}

// Execute arbitrary exportable smart contract functions
func executeContractFunction(
	scid crypto.Hash,
	dero_amount uint64,
	asset_amount uint64,
	funcName string,
	funcType rpc.DataType,
	params []dvm.Variable,
) (err error) {

	var args = rpc.Arguments{}
	var zero uint64
	var dest string

	args = append(
		args,
		rpc.Argument{
			Name:     DEFAULT_SC_ENTRYPOINT,
			DataType: rpc.DataString,
			Value:    funcName,
		},
	)
	args = append(
		args,
		rpc.Argument{
			Name:     rpc.SCID,
			DataType: rpc.DataHash,
			Value:    scid,
		},
	)
	args = append(
		args,
		rpc.Argument{
			Name:     rpc.SCACTION,
			DataType: rpc.DataUint64,
			Value:    uint64(rpc.SC_CALL),
		},
	)

	for p := range params {
		if params[p].Type == 0x4 {
			args = append(
				args,
				rpc.Argument{
					Name:     params[p].Name,
					DataType: rpc.DataUint64,
					Value:    params[p].ValueUint64,
				},
			)
		} else {
			args = append(
				args,
				rpc.Argument{
					Name:     params[p].Name,
					DataType: rpc.DataString,
					Value:    params[p].ValueString,
				},
			)
		}
	}

	if !session.Testnet {
		dest = DERO_DEVELOPER_MAINNET_ADDRESS
	} else {
		if globals.IsSimulator() {
			dest = DERO_DEVELOPER_SIMULATOR_ADDRESS
		} else {
			dest = DERO_DEVELOPER_TESTNET_ADDRESS
		}
	}

	var transfers []rpc.Transfer

	if dero_amount != zero {
		burn := dero_amount

		transfer := rpc.Transfer{
			Destination: dest,
			Amount:      0,
			Burn:        burn,
		}

		transfers = append(transfers, transfer)
	}

	if asset_amount != zero {
		burn := asset_amount

		transfer := rpc.Transfer{
			SCID:        scid,
			Destination: dest,
			Amount:      0,
			Burn:        burn,
		}

		transfers = append(transfers, transfer)
	}

	if len(transfers) < 1 {
		transfer := rpc.Transfer{
			Destination: dest,
			Amount:      0,
			Burn:        0,
		}

		transfers = append(transfers, transfer)
	}

	gasParams := rpc.GasEstimate_Params{
		SC_RPC:    args,
		SC_Value:  0,
		Ringsize:  DEFAULT_SC_RINGSIZE,
		Signer:    engram.Disk.GetAddress().String(),
		Transfers: transfers,
	}

	storage, err := getGasEstimate(gasParams)
	if err != nil {
		fmt.Printf(
			errGasEstimate,
			funcName,
			err,
		)
		return
	}

	tx, err := engram.Disk.TransferPayload0(
		transfers,
		DEFAULT_SC_RINGSIZE,
		false,
		args,
		storage,
		false,
	)

	if err != nil {
		fmt.Printf(
			errBuildTx,
			funcName,
			err,
		)
		return
	}

	err = engram.Disk.SendTransaction(tx)
	if err != nil {
		fmt.Printf(
			errSendTx,
			funcName,
			err,
		)
		return
	}

	walletapi.WaitNewHeightBlock()
	fmt.Printf(
		successFuncTx,
		funcName,
		tx.GetHash().String(),
	)
	_ = tx

	return
}

// Delete the Gnomon directory
func cleanGnomonData() error {
	dir, err := os.ReadDir(
		filepath.Join(
			AppPath(),
			stringdatashards,
			stringgnomon,
		),
	)
	if err != nil {
		fmt.Printf(
			errGnomonPurge,
			err,
		)
		return err
	}

	for _, d := range dir {
		os.RemoveAll(
			filepath.Join(
				[]string{
					AppPath(),
					stringdatashards,
					stringgnomon,
					d.Name(),
				}...,
			),
		)
		fmt.Printf(
			successGnomonPurge,
		)
	}

	return nil
}

// Delete the datashard directory for the active wallet
func cleanWalletData() (err error) {
	path, err := GetShard()
	if err != nil {
		return
	}

	dir, err := os.ReadDir(path)
	if err != nil {
		fmt.Printf(
			errPurgeData, err)
		return err
	}

	for _, d := range dir {
		os.RemoveAll(
			filepath.Join(
				[]string{
					path,
					d.Name(),
				}...,
			),
		)
		fmt.Printf(
			successPurgeData,
		)
	}

	return nil
}

// Get transaction data for any TXID from the daemon
func getTxData(txid string) (result rpc.GetTransaction_Result, err error) {
	if engram.Disk == nil || session.Offline {
		return
	}

	var params rpc.GetTransaction_Params

	params.Tx_Hashes = append(params.Tx_Hashes, txid)

	rpc_client.WS, _, err = websocket.DefaultDialer.Dial(
		"ws://"+session.Daemon+"/ws",
		nil,
	)

	if err != nil {
		return
	}

	input_output := rwc.New(rpc_client.WS)
	rpc_client.RPC = jrpc2.NewClient(
		channel.RawJSON(input_output, input_output),
		nil,
	)

	if err = rpc_client.RPC.CallResult(
		context.Background(),
		DAEMON_GET_TX,
		params,
		&result,
	); err != nil {
		fmt.Printf(
			failTxData,
			txid,
			err,
		)
		return
	}

	rpc_client.WS.Close()
	rpc_client.RPC.Close()

	if result.Status != stringOK {
		fmt.Printf(
			failTxData,
			txid,
			result.Status,
		)
		return
	}

	if len(result.Txs_as_hex[0]) < 50 {
		return
	}

	return
}

// Use a transaction proof to decode and return the payload
func proveGetTxData(txid string, proof_string string) (result ProofData, err error) {
	data, err := getTxData(txid)
	if err != nil {
		return
	}

	ring := data.Txs[0].Ring

	result.Receivers, result.Amounts, _, result.Payloads, err = proof.Prove(
		proof_string,
		data.Txs[0].As_Hex,
		ring,
		engram.Disk.GetNetwork(),
	)

	return
}
