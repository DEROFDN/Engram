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
	"crypto/sha1"
	"errors"
	"fmt"
	"image/color"
	"net/url"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/globals"
	"github.com/deroproject/derohe/rpc"
	"github.com/deroproject/derohe/transaction"
	"github.com/deroproject/derohe/walletapi"
	"github.com/deroproject/derohe/walletapi/mnemonics"
)

func layoutMain() fyne.CanvasObject {
	// Reset UI resources
	resetResources()
	initSettings()
	session.Domain = "app.main"
	session.Path = ""
	session.Password = ""

	// Define objects

	errorText := canvas.NewText(" ", colors.Green)
	errorText.Alignment = fyne.TextAlignCenter
	errorText.TextSize = 12

	if session.Error != "" {
		errorText.Text = session.Error
		errorText.Refresh()
		session.Error = ""
	}

	btnLogin := widget.NewButtonWithIcon(" Enter", resourceEnterPng, func() {
		if session.Path == "" {
			errorText.Text = "No account selected."
			errorText.Refresh()
		} else if session.Password == "" {
			errorText.Text = "Invalid password."
			errorText.Refresh()
		} else {
			errorText.Text = ""
			errorText.Refresh()
			login()
			errorText.Text = session.Error
			errorText.Refresh()
			session.Error = ""
		}
	})

	btnLogin.Disable()

	session.Window.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
		if session.Domain == "app.main" || session.Domain == "app.register" {
			if k.Name == fyne.KeyReturn {
				if session.Path == "" {
					errorText.Text = "No account selected."
					errorText.Refresh()
				} else if session.Password == "" {
					errorText.Text = "Invalid password."
					errorText.Refresh()
				} else {
					errorText.Text = ""
					errorText.Refresh()
					login()
					errorText.Text = session.Error
					errorText.Refresh()
					session.Error = ""
				}
			}
		} else {
			return
		}
	})

	btnCreate := widget.NewButton("New Account", func() {
		session.Domain = "app.create"
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutNewAccount())
	})

	btnRestore := widget.NewButton("Account Recovery", func() {
		session.Domain = "app.restore"
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutRestore())
	})

	btnSettings := widget.NewButton("Settings", func() {
		session.Domain = "app.settings"
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutSettings())
	})

	wPassword := newRtnEntry()
	// Set password masking
	wPassword.Password = true
	// OnChange assign password to session variable
	wPassword.OnChanged = func(s string) {
		session.Error = ""
		errorText.Text = ""
		errorText.Refresh()
		session.Password = s

		if len(s) < 1 {
			btnLogin.Disable()
		} else if session.Path == "" {
			btnLogin.Disable()
		} else {
			btnLogin.Enable()
		}

		btnLogin.Refresh()
	}
	wPassword.SetPlaceHolder("Password")

	// Get account databases in app directory
	list := GetAccounts()

	// Populate the accounts in dropdown menu
	wAccount := widget.NewSelect(list, nil)
	wAccount.PlaceHolder = "(Select Account)"
	wAccount.OnChanged = func(s string) {
		session.Error = ""
		errorText.Text = ""
		errorText.Refresh()

		// OnChange set wallet path
		if !session.Network {
			session.Path = AppPath() + string(filepath.Separator) + "testnet" + string(filepath.Separator) + s
		} else {
			session.Path = AppPath() + string(filepath.Separator) + "mainnet" + string(filepath.Separator) + s
		}

		if session.Password != "" {
			btnLogin.Enable()
		} else {
			btnLogin.Disable()
		}

		session.Window.Canvas().Focus(wPassword)

		btnLogin.Refresh()
	}

	if len(list) < 1 {
		wAccount.Disable()
		wPassword.Disable()
	} else {
		wAccount.Enable()
	}

	wSpacer := widget.NewLabel(" ")
	heading := canvas.NewText("Sign In", colors.Green)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	rectStatus := canvas.NewRectangle(color.Transparent)
	rectStatus.SetMinSize(fyne.NewSize(10, 10))

	res.bg2.SetMinSize(fyne.NewSize(300, 200))
	res.bg2.Refresh()
	res.login_footer.SetMinSize(fyne.NewSize(300, 80))
	res.login_footer.Refresh()

	frame := canvas.NewRectangle(color.Transparent)
	frame.SetMinSize(fyne.NewSize(MIN_WIDTH, MIN_HEIGHT))

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(300, 5))

	status.Connection.FillColor = colors.Gray
	status.Cyberdeck.FillColor = colors.Gray
	status.Sync.FillColor = colors.Gray

	if node.Active == 0 && session.Type == "Local" {
		go startDaemon()
	} else if node.Active == 1 && session.Type == "Remote" {
		stopDaemon()
		fmt.Print("[Engram] Daemon closed.\n")
	}

	if gnomon.Active == 1 && gnomon.Index == nil {
		go startGnomon()
	} else if gnomon.Active == 0 && gnomon.Index != nil {
		stopGnomon()
	}

	// Create a new form for account/password inputs
	form := container.NewVBox(
		rectSpacer,
		rectSpacer,
		rectSpacer,
		res.bg2,
		heading,
		wSpacer,
		wAccount,
		wPassword,
		rectSpacer,
		errorText,
		rectSpacer,
		btnLogin,
		wSpacer,
		btnCreate,
		btnRestore,
		btnSettings,
		wSpacer,
		res.login_footer,
	)

	layout := container.NewMax(
		frame,
		container.NewBorder(
			container.NewVBox(
				container.NewCenter(
					form,
				),
			),
			container.NewVBox(
				container.NewHBox(
					layout.NewSpacer(),
					container.NewMax(
						rectStatus,
						status.Connection,
					),
					widget.NewLabel(""),
					container.NewMax(
						rectStatus,
						status.Sync,
					),
					widget.NewLabel(""),
					container.NewMax(
						rectStatus,
						status.Cyberdeck,
					),
					layout.NewSpacer(),
				),
				rectSpacer,
			),
			nil,
			nil,
		),
	)

	return layout
}

func layoutDashboard() fyne.CanvasObject {
	// Reset UI resources
	resetResources()

	session.Dashboard = "main"
	session.Domain = "app.wallet"

	address := engram.Disk.GetAddress().String()
	short := address[len(address)-10:]
	shard := fmt.Sprintf("%x", sha1.Sum([]byte(address)))
	shard = shard[len(shard)-10:]
	wSpacer := widget.NewLabel(" ")

	session.Balance, _ = engram.Disk.Get_Balance()
	session.BalanceText = canvas.NewText(walletapi.FormatMoney(session.Balance), colors.Green)
	session.BalanceText.TextSize = 28
	session.BalanceText.TextStyle = fyne.TextStyle{Bold: true}

	session.BalanceUSDText = canvas.NewText("", colors.Gray)
	session.BalanceUSDText.TextSize = 14
	session.BalanceUSDText.TextStyle = fyne.TextStyle{Bold: true}

	modeColor := colors.Cold
	if !session.Network {
		session.ModeText = canvas.NewText("T  E  S  T  N  E  T", modeColor)
	} else {
		session.ModeText = canvas.NewText("M  A  I  N  N  E  T", modeColor)
	}
	session.ModeText.TextSize = 13
	session.ModeText.TextStyle = fyne.TextStyle{Bold: true}

	shortShard := canvas.NewText("PRIMARY  USERNAME", colors.Gray)
	shortShard.TextStyle = fyne.TextStyle{Bold: true}
	shortShard.TextSize = 12

	linkColor := colors.Gray

	if cyberdeck.server == nil {
		session.Link = "Blocked"
		linkColor = colors.Gray
	}

	cyberdeck.status = canvas.NewText(session.Link, linkColor)
	cyberdeck.status.TextSize = 22
	cyberdeck.status.TextStyle = fyne.TextStyle{Bold: true}

	rectStatus := canvas.NewRectangle(color.Transparent)
	rectStatus.SetMinSize(fyne.NewSize(10, 10))
	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(fyne.NewSize(MIN_WIDTH, 20))
	frame := canvas.NewRectangle(color.Transparent)
	frame.SetMinSize(fyne.NewSize(MIN_WIDTH, MIN_HEIGHT))

	// Containers
	balanceCenter := container.NewCenter(
		container.NewVBox(
			container.NewCenter(
				session.BalanceText,
			),
			container.NewCenter(
				session.BalanceUSDText,
			),
		),
	)

	modeCenter := container.NewCenter(
		session.ModeText,
	)

	idCenter := container.NewCenter(
		shortShard,
	)

	linkCenter := container.NewCenter(
		cyberdeck.status,
	)

	vLabel := canvas.NewText(" - ", color.RGBA{R: 255, G: 255, B: 255, A: 255})
	vLabel.SetMinSize(fyne.NewSize(300, 300))

	cyberdeck.toggle = widget.NewButton("Turn On", nil)
	cyberdeck.toggle.OnTapped = func() {
		if cyberdeck.active == 0 {
			setRPCLogin()
			cyberdeck.status.Text = "Allowed"
			cyberdeck.status.Color = colors.Green
			cyberdeck.status.Refresh()
			status.Cyberdeck.FillColor = colors.Green
			status.Cyberdeck.StrokeColor = colors.Green
			status.Cyberdeck.Refresh()
			cyberdeck.toggle.Text = "Turn Off"
			status.Authenticator.Value = 60
			status.Authenticator.Refresh()
			linkCenter.Refresh()
			cyberdeck.checkbox.Disable()
			cyberdeck.active = 1
			go cyberdeckUpdate()
		} else {
			if cyberdeck.active == 1 && cyberdeck.server != nil {
				cyberdeck.active = 0
				cyberdeck.server.RPCServer_Stop()
				cyberdeck.server = nil
				cyberdeck.status.Text = "Blocked"
				cyberdeck.status.Color = colors.Gray
				cyberdeck.status.Refresh()
				status.Cyberdeck.FillColor = colors.Gray
				status.Cyberdeck.Refresh()
				linkCenter.Refresh()
				status.Authenticator.Value = 0
				status.Authenticator.Refresh()
				cyberdeck.toggle.Text = "Turn On"
				cyberdeck.checkbox.Enable()
			}
		}
	}

	btnSend := widget.NewButton("Save", nil)
	//btnSendNow := widget.NewButton(" Send ", nil)

	btnCopyLogin := widget.NewButton("Copy", func() {
		session.Window.Clipboard().SetContent(cyberdeck.user + ":" + cyberdeck.pass)
	})

	wReceiver := widget.NewEntry()
	wReceiver.Validator = func(s string) error {
		address, err := globals.ParseValidateAddress(s)
		if err != nil {
			tx.Address = nil
			go func() {
				exists, addr, err := checkUsername(s)
				if err != nil && !exists {
					btnSend.Disable()
					//btnSendNow.Disable()
					wReceiver.SetValidationError(errors.New("invalid username or address"))
					tx.Address = nil
				} else {
					wReceiver.SetValidationError(nil)
					tx.Address, _ = globals.ParseValidateAddress(addr)
					if tx.Amount != 0 {
						balance, _ := engram.Disk.Get_Balance()
						if tx.Amount <= balance {
							btnSend.Enable()
						}
					}
				}
			}()
		} else {
			tx.Address = address
			wReceiver.SetValidationError(nil)
			if tx.Amount != 0 {
				balance, _ := engram.Disk.Get_Balance()
				if tx.Amount <= balance {
					btnSend.Enable()
				}
			}
		}
		return nil
	}

	wReceiver.SetPlaceHolder("Receiver username or address")
	wReceiver.OnChanged = func(s string) {
		address, err := globals.ParseValidateAddress(s)
		if err != nil {
			wReceiver.SetValidationError(errors.New("invalid username or address"))
			return
		} else {
			tx.Address = address
		}
	}
	wReceiver.SetValidationError(nil)

	wAmount := widget.NewEntry()
	wAmount.SetPlaceHolder("Amount")

	wMessage := widget.NewEntry()
	wMessage.SetPlaceHolder("Message")
	wMessage.OnChanged = func(s string) {
		bytes := []byte(s)
		if len(bytes) <= 130 {
			tx.Comment = s
		}
		wMessage.SetText(tx.Comment)
	}

	wAll := widget.NewCheck(" All", func(b bool) {
		if b {
			tx.Amount = engram.Disk.GetAccount().Balance_Mature
			wAmount.SetText(walletapi.FormatMoney(tx.Amount))
		} else {
			tx.Amount = 0
			wAmount.SetText("")
		}
	})

	wAmount.Validator = func(s string) error {
		if s == "" {
			tx.Amount = 0
			wAmount.SetValidationError(errors.New("Invalid transaction amount"))
			btnSend.Disable()
			//btnSendNow.Disable()
		} else {
			balance, _ := engram.Disk.Get_Balance()
			entry, err := globals.ParseAmount(s)
			if err != nil {
				tx.Amount = 0
				wAmount.SetValidationError(errors.New("Invalid transaction amount"))
				btnSend.Disable()
				//btnSendNow.Disable()
				return errors.New("Invalid transaction amount")
			}
			if entry <= balance {
				tx.Amount = entry
				wAmount.SetValidationError(nil)
				if wReceiver.Validate() == nil {
					btnSend.Enable()
					//btnSendNow.Enable()
				}
			} else {
				tx.Amount = 0
				btnSend.Disable()
				//btnSendNow.Disable()
				wAmount.SetValidationError(errors.New("Insufficient funds"))
			}
			return nil
		}
		return errors.New("Invalid transaction amount")
	}

	wAmount.SetValidationError(nil)

	wPaymentID := widget.NewEntry()
	wPaymentID.OnChanged = func(s string) {
		tx.PaymentID = 0
	}
	wPaymentID.SetPlaceHolder("Payment ID / Service Port")

	options := []string{"2", "4", "8", "16", "32", "64", "128", "256"}
	wRings := widget.NewSelect(options, nil)
	wRings.PlaceHolder = "(Select Anonymity Set)"
	wRings.OnChanged = func(s string) {
		tx.Ringsize, _ = strconv.ParseUint(s, 10, 64)
		session.Window.Canvas().Focus(wReceiver)
	}

	rect.SetMinSize(fyne.NewSize(300, 30))
	rectSynapse := canvas.NewRectangle(color.Transparent)
	rectSynapse.SetMinSize(fyne.NewSize(75, 75))
	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(10, 5))
	rectCenter := canvas.NewRectangle(colors.DarkMatter)
	rectCenter.FillColor = colors.Green
	rect.SetMinSize(fyne.NewSize(10, 10))
	rectUp := canvas.NewRectangle(colors.DarkMatter)
	rect.SetMinSize(fyne.NewSize(10, 10))
	rectDown := canvas.NewRectangle(colors.DarkMatter)
	rect.SetMinSize(fyne.NewSize(10, 10))
	rectLeft := canvas.NewRectangle(colors.DarkMatter)
	rect.SetMinSize(fyne.NewSize(10, 10))
	rectRight := canvas.NewRectangle(colors.DarkMatter)
	rectRight.SetMinSize(fyne.NewSize(10, 10))
	rectEmpty := canvas.NewRectangle(color.Transparent)
	rectEmpty.SetMinSize(fyne.NewSize(10, 10))

	btnSend.OnTapped = func() {
		var arguments = rpc.Arguments{}

		fmt.Printf("[Send] Starting tx...\n")
		if tx.Address.IsIntegratedAddress() {
			if tx.Address.Arguments.Validate_Arguments() != nil {
				fmt.Printf("[Service] Integrated Address arguments could not be validated.")
				return
			}

			fmt.Printf("[Send] Not Integrated..\n")
			if !tx.Address.Arguments.Has(rpc.RPC_DESTINATION_PORT, rpc.DataUint64) {
				fmt.Printf("[Service] Integrated Address does not contain destination port.")
				return
			}

			arguments = append(arguments, rpc.Argument{Name: rpc.RPC_DESTINATION_PORT, DataType: rpc.DataUint64, Value: tx.Address.Arguments.Value(rpc.RPC_DESTINATION_PORT, rpc.DataUint64).(uint64)})
			fmt.Printf("[Send] Added arguments..\n")

			if tx.Address.Arguments.Has(rpc.RPC_EXPIRY, rpc.DataTime) {

				if tx.Address.Arguments.Value(rpc.RPC_EXPIRY, rpc.DataTime).(time.Time).Before(time.Now().UTC()) {
					fmt.Printf("[Service] This address has expired.", "expiry time", tx.Address.Arguments.Value(rpc.RPC_EXPIRY, rpc.DataTime))
					return
				} else {
					fmt.Printf("[Service] This address will expire ", "expiry time", tx.Address.Arguments.Value(rpc.RPC_EXPIRY, rpc.DataTime))
					return
				}
			}

			fmt.Printf("[Service] Destination port is integrated in address.", "dst port", tx.Address.Arguments.Value(rpc.RPC_DESTINATION_PORT, rpc.DataUint64).(uint64))

			if tx.Address.Arguments.Has(rpc.RPC_COMMENT, rpc.DataString) {
				fmt.Printf("[Service] Integrated Message", "comment", tx.Address.Arguments.Value(rpc.RPC_COMMENT, rpc.DataString))
				arguments = append(arguments, rpc.Argument{rpc.RPC_COMMENT, rpc.DataString, tx.Address.Arguments.Value(rpc.RPC_COMMENT, rpc.DataString)})
			}
		}

		fmt.Printf("[Send] Checking arguments..\n")

		for _, arg := range tx.Address.Arguments {
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

		fmt.Printf("[Send] Checking Amount..\n")

		if tx.Address.Arguments.Has(rpc.RPC_VALUE_TRANSFER, rpc.DataUint64) {
			fmt.Printf("[Service] Transaction amount: %x", globals.FormatMoney(tx.Address.Arguments.Value(rpc.RPC_VALUE_TRANSFER, rpc.DataUint64).(uint64)))
			tx.Amount = tx.Address.Arguments.Value(rpc.RPC_VALUE_TRANSFER, rpc.DataUint64).(uint64)
		} else {
			balance, _ := engram.Disk.Get_Balance()
			fmt.Printf("[Send] Balance: %d\n", balance)
			fmt.Printf("[Send] Amount: %d\n", tx.Amount)

			if tx.Amount > balance {
				fmt.Printf("[Send] Error: Insufficient funds")
				return
			} else if tx.Amount == balance {
				tx.SendAll = true
			} else {
				tx.SendAll = false
			}
		}

		fmt.Printf("[Send] Checking services..\n")

		if tx.Address.Arguments.Has(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataUint64) {
			arguments = append(arguments, rpc.Argument{Name: rpc.RPC_REPLYBACK_ADDRESS, DataType: rpc.DataAddress, Value: engram.Disk.GetAddress()})
		}

		fmt.Printf("[Send] Checking payment ID/destination port..\n")

		if len(arguments) == 0 {
			arguments = append(arguments, rpc.Argument{Name: rpc.RPC_DESTINATION_PORT, DataType: rpc.DataUint64, Value: tx.PaymentID})
			arguments = append(arguments, rpc.Argument{Name: rpc.RPC_COMMENT, DataType: rpc.DataString, Value: tx.Comment})
		}

		fmt.Printf("[Send] Checking Pack..\n")

		if _, err := arguments.CheckPack(transaction.PAYLOAD0_LIMIT); err != nil {
			fmt.Printf("[Send] Arguments packing err: %s\n", err)
			return
		}

		if tx.Ringsize == 0 {
			tx.Ringsize = 2
		} else if tx.Ringsize > 256 {
			tx.Ringsize = 256
		} else if !crypto.IsPowerOf2(int(tx.Ringsize)) {
			tx.Ringsize = 2
			fmt.Printf("[Send] Error: Invalid ringsize - %d is not a power of 2\n", tx.Ringsize)
			return
		}

		//tx.Fees = tx.TX.Fees()
		tx.Status = "Unsent"

		fmt.Printf("[Send] Ringsize: %d\n", tx.Ringsize)

		addTransfer(arguments)
	}

	go getPrice()

	shardTitle := canvas.NewText("D A T A S H A R D", colors.Gray)
	shardTitle.TextStyle = fyne.TextStyle{Bold: true}
	shardTitle.TextSize = 16

	deckTitle := canvas.NewText("C Y B E R D E C K", colors.Gray)
	deckTitle.TextStyle = fyne.TextStyle{Bold: true}
	deckTitle.TextSize = 16

	sendTitle := canvas.NewText("T R A N S F E R", colors.Gray)
	sendTitle.TextStyle = fyne.TextStyle{Bold: true}
	sendTitle.TextSize = 16

	nodeTitle := canvas.NewText("H O M E", colors.Gray)
	nodeTitle.TextStyle = fyne.TextStyle{Bold: true}
	nodeTitle.TextSize = 16

	path := strings.Split(session.Path, string(filepath.Separator))
	accountName := canvas.NewText(path[len(path)-1], colors.Green)
	accountName.TextStyle = fyne.TextStyle{Bold: true}
	accountName.TextSize = 18

	shardText := canvas.NewText(session.Username, colors.Green)
	shardText.TextStyle = fyne.TextStyle{Bold: true}
	shardText.TextSize = 22

	shortAddress := canvas.NewText("····"+short, colors.Gray)
	shortAddress.TextStyle = fyne.TextStyle{Bold: true}
	shortAddress.TextSize = 22

	cyberdeck.checkbox = widget.NewCheck("Use Authenticator", nil)
	cyberdeck.checkbox.OnChanged = func(b bool) {
		if b {
			cyberdeck.mode = 1
			setAuthMode("true")
			cyberdeck.userText.Disable()
			cyberdeck.passText.Disable()
		} else {
			cyberdeck.mode = 0
			setAuthMode("false")
			cyberdeck.userText.Enable()
			cyberdeck.passText.Enable()
		}
	}

	cyberdeck.userText = widget.NewEntry()
	cyberdeck.userText.PlaceHolder = "Username"
	cyberdeck.userText.OnChanged = func(s string) {
		if len(s) > 1 {
			cyberdeck.user = s
		}
	}

	cyberdeck.passText = widget.NewEntry()
	cyberdeck.passText.PlaceHolder = "Password"
	cyberdeck.passText.OnChanged = func(s string) {
		if len(s) > 1 {
			cyberdeck.pass = s
		}
	}

	status.Authenticator = widget.NewProgressBar()
	status.Authenticator.Max = 60
	status.Authenticator.Min = 0
	status.Authenticator.Value = 60
	status.Authenticator.TextFormatter = func() string {
		return ""
	}

	if cyberdeck.mode == 1 {
		cyberdeck.checkbox.Checked = true
		cyberdeck.userText.Disable()
		cyberdeck.passText.Disable()
	} else {
		cyberdeck.checkbox.Checked = false
		cyberdeck.userText.Enable()
		cyberdeck.passText.Enable()
	}

	authRect := canvas.NewRectangle(color.Transparent)
	authRect.SetMinSize(fyne.NewSize(300, 3))

	gramSend := widget.NewButton(" Send ", nil)
	gramManage := widget.NewButton(" My Account ", nil)
	gramManage.OnTapped = func() {

		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutAccount())
	}

	entryReg := widget.NewEntry()

	btnReg := widget.NewButton(" Register ", nil)
	btnReg.Disable()
	btnReg.OnTapped = func() {
		if len(session.NewUser) > 5 {
			_, _, err := checkUsername(session.NewUser)
			if err != nil {
				err = registerUsername(session.NewUser)
				if err != nil {
					btnReg.Enable()
				} else {
					btnReg.Disable()
					session.NewUser = ""
					entryReg.Text = ""
					entryReg.Refresh()
				}
			}
		}
	}

	entryReg.PlaceHolder = "New Username"
	entryReg.OnChanged = func(s string) {
		session.NewUser = s
		if len(s) > 5 {
			_, _, err := checkUsername(s)
			if err != nil {
				btnReg.Enable()
			} else {
				btnReg.Disable()
			}
		} else {
			btnReg.Disable()
		}
	}

	btnClose := widget.NewButton(" Log Out ", func() {
		closeWallet()
	})

	btnCancel := widget.NewButton(" Cancel ", nil)

	heading := canvas.NewText("Balance", colors.Gray)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	sendDesc := canvas.NewText("Add Transfer Details", colors.Gray)
	sendDesc.TextSize = 18
	sendDesc.Alignment = fyne.TextAlignCenter
	sendDesc.TextStyle = fyne.TextStyle{Bold: true}

	sendHeading := canvas.NewText("Send Money", colors.Green)
	sendHeading.TextSize = 22
	sendHeading.Alignment = fyne.TextAlignCenter
	sendHeading.TextStyle = fyne.TextStyle{Bold: true}

	serverStatus := canvas.NewText("APPLICATION  CONNECTIONS", colors.Gray)
	serverStatus.TextSize = 12
	serverStatus.Alignment = fyne.TextAlignCenter
	serverStatus.TextStyle = fyne.TextStyle{Bold: true}

	res.gram.SetMinSize(fyne.NewSize(300, 185))
	res.gram_footer.SetMinSize(fyne.NewSize(300, 80))
	res.rpc_header.SetMinSize(fyne.NewSize(300, 80))
	res.rpc_footer.SetMinSize(fyne.NewSize(300, 20))
	res.home_footer.SetMinSize(fyne.NewSize(300, 20))
	res.nft_footer.SetMinSize(fyne.NewSize(300, 20))

	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(fyne.NewSize(300, 200))

	userData, err := queryUsernames()
	if err != nil {

	}

	userList := binding.BindStringList(&userData)

	userBox := widget.NewListWithData(userList,
		func() fyne.CanvasObject {
			c := container.NewVBox(
				widget.NewLabel(""),
			)
			return c
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			dat := di.(binding.String)
			str, err := dat.Get()
			if err != nil {
				return
			}

			co.(*fyne.Container).Objects[0].(*widget.Label).SetText(str)
			co.(*fyne.Container).Objects[0].(*widget.Label).Wrapping = fyne.TextWrapWord
			co.(*fyne.Container).Objects[0].(*widget.Label).TextStyle.Bold = false
			co.(*fyne.Container).Objects[0].(*widget.Label).Alignment = fyne.TextAlignLeading
		})

	userBox.OnSelected = func(id widget.ListItemID) {
		setPrimaryUsername(userData[id])
		session.Username = userData[id]
		shardText.Text = userData[id]
		shardText.Refresh()
	}

	getPrimaryUsername()

	if session.Username == "" {
		shardText.Text = "---"
		shardText.Refresh()
	} else {
		for u := range userData {
			if userData[u] == session.Username {
				userBox.Select(u)
				userBox.ScrollTo(u)
			}
		}
	}

	synapse := container.NewMax(
		rectSynapse,
		container.NewGridWithColumns(3,
			rectEmpty,
			rectUp,
			rectEmpty,
			rectLeft,
			rectCenter,
			rectRight,
			rectEmpty,
			rectDown,
			rectEmpty,
		),
	)

	deroForm := container.NewVBox(
		wSpacer,
		res.gram,
		heading,
		rectSpacer,
		balanceCenter,
		wSpacer,
		gramSend,
		gramManage,
		btnClose,
		wSpacer,
		container.NewCenter(
			res.gram_footer,
			container.NewVBox(
				rectSpacer,
				modeCenter,
			),
		),
	)

	deckForm := container.NewVBox(
		wSpacer,
		container.NewCenter(
			res.rpc_header,
			container.NewVBox(
				deckTitle,
				rectSpacer,
			),
		),
		rectSpacer,
		linkCenter,
		rectSpacer,
		serverStatus,
		wSpacer,
		cyberdeck.toggle,
		wSpacer,
		cyberdeck.userText,
		rectSpacer,
		cyberdeck.passText,
		rectSpacer,
		cyberdeck.checkbox,
		wSpacer,
		rectSpacer,
		container.NewMax(
			status.Authenticator,
			container.NewMax(
				btnCopyLogin,
			),
		),
		wSpacer,
		res.rpc_footer,
	)

	sendForm := container.NewVBox(
		wSpacer,
		container.NewCenter(res.rpc_header, container.NewVBox(sendTitle, rectSpacer)),
		rectSpacer,
		sendHeading,
		//rectSpacer,
		//sendDesc,
		wSpacer,
		wRings,
		wReceiver,
		wAmount,
		//wAll,
		wPaymentID,
		wMessage,
		wSpacer,
		//btnSendNow,
		wSpacer,
		btnSend,
		btnCancel,
		wSpacer,
		res.rpc_footer,
	)

	shardForm := container.NewVBox(
		wSpacer,
		container.NewCenter(res.rpc_header, container.NewVBox(shardTitle, rectSpacer)),
		rectSpacer,
		container.NewMax(container.NewCenter(shardText)),
		rectSpacer,
		idCenter,
		wSpacer,
		container.NewMax(
			rectList,
			userBox,
		),
		rectSpacer,
		widget.NewSeparator(),
		rectSpacer,
		entryReg,
		rectSpacer,
		btnReg,
		wSpacer,
		res.rpc_footer,
	)

	gridItem2 := container.NewCenter(
		deroForm,
	)

	gridItem3 := container.NewCenter(
		deckForm,
	)

	gridItem1 := container.NewCenter(
		shardForm,
	)

	gridItem4 := container.NewCenter(
		sendForm,
	)

	gridItem1.Hidden = true
	gridItem2.Hidden = false
	gridItem3.Hidden = true
	gridItem4.Hidden = true

	gramSend.OnTapped = func() {
		rectCenter.FillColor = colors.Blue
		rectCenter.Refresh()
		gridItem1.Hidden = true
		gridItem2.Hidden = true
		gridItem3.Hidden = true
		gridItem4.Hidden = false
	}

	btnCancel.OnTapped = func() {
		//tx = Transfers{}
		rectCenter.FillColor = colors.Green
		rectCenter.Refresh()
		gridItem1.Hidden = true
		gridItem2.Hidden = false
		gridItem3.Hidden = true
		gridItem4.Hidden = true
		wReceiver.SetText("")
		wPaymentID.SetText("")
		wAmount.SetText("")
		wAll.SetChecked(false)
		wMessage.SetText("")
		wRings.ClearSelected()
		sendForm.Refresh()
	}

	status.Daemon = newImageButton(resourceDaemonOffPng, nil)
	status.Daemon.OnTapped = func() {
		// Maybe GUI for daemon?
	}

	if node.Active == 0 {
		status.Daemon.Image.Resource = resourceDaemonOffPng
		status.Daemon.Refresh()
	} else {
		status.Daemon.Image.Resource = resourceDaemonOnPng
		status.Daemon.Refresh()
	}

	status.Netrunner = newImageButton(resourceMinerOffPng, nil)
	status.Netrunner.OnTapped = func() {
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
	}

	session.Window.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
		if session.Domain != "app.wallet" {
			return
		}

		if k.Name == fyne.KeyRight {
			if session.Dashboard == "node" {
				session.Dashboard = "main"
				gridItem1.Hidden = true
				gridItem2.Hidden = false
				gridItem3.Hidden = true
				gridItem4.Hidden = true
				rectCenter.FillColor = colors.Green
				rectCenter.Refresh()
				rectUp.FillColor = colors.DarkMatter
				rectUp.Refresh()
				rectDown.FillColor = colors.DarkMatter
				rectDown.Refresh()
				rectLeft.FillColor = colors.DarkMatter
				rectLeft.Refresh()
				rectRight.FillColor = colors.DarkMatter
				rectRight.Refresh()
			} else if session.Dashboard == "main" {
				session.Dashboard = "deck"
				gridItem1.Hidden = true
				gridItem2.Hidden = true
				gridItem3.Hidden = false
				gridItem4.Hidden = true
				rectCenter.FillColor = colors.DarkMatter
				rectCenter.Refresh()
				rectUp.FillColor = colors.DarkMatter
				rectUp.Refresh()
				rectDown.FillColor = colors.DarkMatter
				rectDown.Refresh()
				rectLeft.FillColor = colors.DarkMatter
				rectLeft.Refresh()
				rectRight.FillColor = colors.Green
				rectRight.Refresh()
			} else {
				return
			}

			gridItem1.Refresh()
			gridItem2.Refresh()
			gridItem3.Refresh()
			session.Window.Canvas().Content().Refresh()
		} else if k.Name == fyne.KeyLeft {
			if session.Dashboard == "deck" {
				session.Dashboard = "main"
				gridItem1.Hidden = true
				gridItem2.Hidden = false
				gridItem3.Hidden = true
				gridItem4.Hidden = true
				rectCenter.FillColor = colors.Green
				rectCenter.Refresh()
				rectUp.FillColor = colors.DarkMatter
				rectUp.Refresh()
				rectDown.FillColor = colors.DarkMatter
				rectDown.Refresh()
				rectLeft.FillColor = colors.DarkMatter
				rectLeft.Refresh()
				rectRight.FillColor = colors.DarkMatter
				rectRight.Refresh()
			} else if session.Dashboard == "main" {
				session.Dashboard = "node"
				gridItem1.Hidden = false
				gridItem2.Hidden = true
				gridItem3.Hidden = true
				gridItem4.Hidden = true
				rectCenter.FillColor = colors.DarkMatter
				rectCenter.Refresh()
				rectUp.FillColor = colors.DarkMatter
				rectUp.Refresh()
				rectDown.FillColor = colors.DarkMatter
				rectDown.Refresh()
				rectLeft.FillColor = colors.Green
				rectLeft.Refresh()
				rectRight.FillColor = colors.DarkMatter
				rectRight.Refresh()
			} else {
				return
			}

			gridItem1.Refresh()
			gridItem2.Refresh()
			gridItem3.Refresh()
			session.Window.Canvas().Content().Refresh()
		} else if k.Name == fyne.KeyUp {
			if session.Dashboard == "main" {
				session.Dashboard = "transfers"

				session.Window.SetContent(layoutTransition())
				session.Window.SetContent(layoutTransfers())
			} else {
				return
			}
		} else if k.Name == fyne.KeyDown {
			if session.Dashboard == "main" {
				session.Dashboard = "contacts"

				session.Window.SetContent(layoutTransition())
				session.Window.SetContent(layoutMessages())
			} else {
				return
			}
		}

	})

	features := container.NewCenter(
		layout.NewSpacer(),
		gridItem1,
		layout.NewSpacer(),
		gridItem2,
		layout.NewSpacer(),
		gridItem3,
		layout.NewSpacer(),
		gridItem4,
		layout.NewSpacer(),
	)

	subContainer := container.NewMax(
		container.NewVBox(
			container.NewHBox(
				layout.NewSpacer(),
				container.NewVBox(
					rectSpacer,
					status.Daemon,
				),
				layout.NewSpacer(),
				container.NewCenter(
					rectSynapse,
					synapse,
				),
				layout.NewSpacer(),
				container.NewVBox(
					rectSpacer,
					status.Netrunner,
				),
				layout.NewSpacer(),
			),
			rectSpacer,
			container.NewHBox(
				layout.NewSpacer(),
				container.NewMax(
					rectStatus,
					status.Connection,
				),
				widget.NewLabel(""),
				container.NewMax(
					rectStatus,
					status.Sync,
				),
				widget.NewLabel(""),
				container.NewMax(
					rectStatus,
					status.Cyberdeck,
				),
				layout.NewSpacer(),
			),
		),
		rectSpacer,
		rectSpacer,
	)

	c := container.NewBorder(
		features,
		container.NewVBox(
			subContainer,
			rectSpacer,
		),
		nil,
		nil,
	)

	layout := container.NewMax(
		frame,
		c,
	)

	return layout
}

func layoutLoading() fyne.CanvasObject {
	res.load.FillMode = canvas.ImageFillContain
	layout := container.NewMax(
		res.load,
	)

	return layout
}

func layoutNewAccount() fyne.CanvasObject {
	// Reset UI resources
	resetResources()

	// Define
	session.Domain = "app.register"
	session.Language = -1
	session.Error = ""
	session.Name = ""
	session.Password = ""
	session.PasswordConfirm = ""

	languages := mnemonics.Language_List()

	errorText := canvas.NewText(" ", colors.Green)
	errorText.TextSize = 12
	errorText.Alignment = fyne.TextAlignCenter

	btnCreate := widget.NewButton("Create", nil)
	btnCreate.Disable()

	btnCancel := widget.NewButton("Cancel", func() {
		session.Domain = "app.main"
		session.Error = ""
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutMain())
	})

	btnClose := widget.NewButton("Close", func() {
		session.Domain = "app.main"
		session.Error = ""
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutMain())
	})

	btnCopySeed := widget.NewButton("Copy Recovery Words", nil)
	btnCopyAddress := widget.NewButton("Copy Address", nil)

	session.Window.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
		if session.Domain != "app.register" {
			return
		}

		if k.Name == fyne.KeyReturn {
			errorText.Text = ""
			errorText.Refresh()
			create()
			errorText.Text = session.Error
			errorText.Refresh()
		}
	})

	wPassword := newRtnEntry()
	// Set password masking
	wPassword.Password = true
	// OnChange assign password to session variable
	// Remember to clear it after opening the wallet
	wPassword.OnChanged = func(s string) {
		session.Error = ""
		errorText.Text = ""
		errorText.Refresh()
		session.Password = s

		if len(session.Password) > 0 && session.Password == session.PasswordConfirm && !findAccount() && session.Language != -1 {
			btnCreate.Enable()
			btnCreate.Refresh()
		} else {
			btnCreate.Disable()
			btnCreate.Refresh()
		}
	}
	wPassword.SetPlaceHolder("Password")
	wPassword.Wrapping = fyne.TextTruncate

	wPasswordConfirm := newRtnEntry()
	// Set password masking
	wPasswordConfirm.Password = true
	// OnChange assign password to session variable
	// Remember to clear it after opening the wallet
	wPasswordConfirm.OnChanged = func(s string) {
		session.Error = ""
		errorText.Text = ""
		errorText.Refresh()
		session.PasswordConfirm = s

		if len(session.Password) > 0 && session.Password == session.PasswordConfirm && !findAccount() && session.Language != -1 {
			btnCreate.Enable()
			btnCreate.Refresh()
		} else {
			btnCreate.Disable()
			btnCreate.Refresh()
		}
	}
	wPasswordConfirm.SetPlaceHolder("Confirm Password")
	wPasswordConfirm.Wrapping = fyne.TextTruncate

	wAccount := widget.NewEntry()
	wAccount.SetPlaceHolder("Account Name")
	wAccount.Validator = func(s string) (err error) {
		session.Error = ""
		errorText.Text = ""
		errorText.Refresh()

		if len(s) > 30 {
			err = errors.New("Account name is too long.")
			wAccount.SetText(session.Name)
			wAccount.Refresh()
			return
		}

		checkDir()
		getNetwork()
		if !session.Network {
			session.Path = AppPath() + string(filepath.Separator) + "testnet" + string(filepath.Separator) + s + ".db"
		} else {
			session.Path = AppPath() + string(filepath.Separator) + "mainnet" + string(filepath.Separator) + s + ".db"
		}
		session.Name = s

		if findAccount() {
			err = errors.New("Account name already exists.")
			errorText.Text = err.Error()
			errorText.Color = colors.Red
			errorText.Refresh()
			return
		} else {
			errorText.Text = ""
			errorText.Refresh()
		}

		if len(session.Password) > 0 && session.Password == session.PasswordConfirm && !findAccount() && session.Language != -1 {
			btnCreate.Enable()
			btnCreate.Refresh()
		} else {
			btnCreate.Disable()
			btnCreate.Refresh()
		}
		return nil
	}

	wAccount.OnChanged = func(s string) {
		wAccount.Validate()
	}

	wLanguage := widget.NewSelect(languages, nil)
	wLanguage.OnChanged = func(s string) {
		index := wLanguage.SelectedIndex()
		session.Language = index
		session.Window.Canvas().Focus(wAccount)

		if len(session.Password) > 0 && session.Password == session.PasswordConfirm && !findAccount() && session.Language != -1 {
			btnCreate.Enable()
			btnCreate.Refresh()
		} else {
			btnCreate.Disable()
			btnCreate.Refresh()
		}
	}
	wLanguage.PlaceHolder = "(Select Language)"

	wSpacer := widget.NewLabel(" ")
	heading := canvas.NewText("New Account", colors.Green)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	heading2 := canvas.NewText("Recovery", colors.Green)
	heading2.TextSize = 22
	heading2.Alignment = fyne.TextAlignCenter
	heading2.TextStyle = fyne.TextStyle{Bold: true}

	rectStatus := canvas.NewRectangle(color.Transparent)
	rectStatus.SetMinSize(fyne.NewSize(10, 10))

	title := canvas.NewText("R E G I S T R A T I O N", colors.Gray)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 16

	res.rpc_header.SetMinSize(fyne.NewSize(300, 80))
	res.rpc_footer.SetMinSize(fyne.NewSize(300, 80))

	frame := canvas.NewRectangle(color.Transparent)
	frame.SetMinSize(fyne.NewSize(MIN_WIDTH, MIN_HEIGHT))

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(300, 5))

	status.Connection.FillColor = colors.Gray
	status.Cyberdeck.FillColor = colors.Gray
	status.Sync.FillColor = colors.Gray

	grid := container.NewAdaptiveGrid(1)
	grid.Objects = nil

	// Create a new form for account/password inputs
	form := container.NewVBox(
		wSpacer,
		container.NewCenter(
			res.rpc_header,
			container.NewVBox(
				container.NewCenter(title),
				rectSpacer,
			),
		),
		rectSpacer,
		heading,
		wSpacer,
		wLanguage,
		wAccount,
		wPassword,
		wPasswordConfirm,
		rectSpacer,
		errorText,
		rectSpacer,
		wSpacer,
		wSpacer,
		wSpacer,
		wSpacer,
		rectSpacer,
		rectSpacer,
		btnCreate,
		btnCancel,
		rectSpacer,
		res.rpc_footer,
	)

	rectForm := canvas.NewRectangle(color.Transparent)
	rectForm.SetMinSize(fyne.NewSize(300, 750))

	body := widget.NewLabel("Please save the following 25 recovery words in a safe place. These are the keys to your account, so never share them with anyone.")
	body.Wrapping = fyne.TextWrapWord
	body.Alignment = fyne.TextAlignCenter
	body.TextStyle = fyne.TextStyle{Bold: true}

	formSuccess := container.NewVBox(
		wSpacer,
		container.NewCenter(
			res.rpc_header,
			container.NewVBox(
				container.NewCenter(title),
				rectSpacer,
			),
		),
		rectSpacer,
		heading2,
		rectSpacer,
		body,
		wSpacer,
		container.NewCenter(grid),
		rectSpacer,
		errorText,
		rectSpacer,
		btnCopyAddress,
		btnCopySeed,
		btnClose,
		rectSpacer,
		res.rpc_footer,
	)

	formSuccess.Hide()

	scrollBox := container.NewVScroll(
		container.NewHBox(
			layout.NewSpacer(),
			container.NewMax(
				formSuccess,
				form,
			),
			layout.NewSpacer(),
		),
	)

	scrollBox.SetMinSize(fyne.NewSize(300, 750))

	btnCreate.OnTapped = func() {
		if findAccount() {
			errorText.Text = "Account name already exists."
			errorText.Color = colors.Red
			errorText.Refresh()
			return
		} else {
			errorText.Text = ""
			errorText.Refresh()
		}

		address, seed, err := create()
		if err != nil {
			errorText.Text = session.Error
			errorText.Refresh()
			return
		}

		formatted := strings.Split(seed, " ")

		rect := canvas.NewRectangle(color.RGBA{19, 25, 34, 255})
		rect.SetMinSize(fyne.NewSize(300, 25))

		for i := 0; i < len(formatted); i++ {
			pos := fmt.Sprintf("%d", i+1)
			word := strings.ReplaceAll(formatted[i], " ", "")
			grid.Add(container.NewMax(
				rect,
				container.NewHBox(
					widget.NewLabel(" "),
					widget.NewLabelWithStyle(pos, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
					layout.NewSpacer(),
					widget.NewLabelWithStyle(word, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
					widget.NewLabel(" "),
				),
			),
			)
		}

		btnCopySeed.OnTapped = func() {
			session.Window.Clipboard().SetContent(seed)
		}

		btnCopyAddress.OnTapped = func() {
			session.Window.Clipboard().SetContent(address)
		}

		form.Hide()
		form.Refresh()
		formSuccess.Show()
		formSuccess.Refresh()
		grid.Refresh()
		scrollBox.Refresh()
		session.Window.Canvas().Content().Refresh()
		session.Window.Canvas().Refresh(session.Window.Content())
	}

	layout := container.NewMax(
		frame,
		container.NewVBox(
			scrollBox,
			layout.NewSpacer(),
			container.NewVBox(
				container.NewHBox(
					layout.NewSpacer(),
					container.NewMax(
						rectStatus,
						status.Connection,
					),
					widget.NewLabel(""),
					container.NewMax(
						rectStatus,
						status.Sync,
					),
					widget.NewLabel(""),
					container.NewMax(
						rectStatus,
						status.Cyberdeck,
					),
					layout.NewSpacer(),
				),
				rectSpacer,
			),
		),
	)
	return layout
}

func layoutRestore() fyne.CanvasObject {
	var seed [25]string
	// Reset UI resources
	resetResources()

	// Define
	session.Domain = "app.restore"
	session.Language = -1
	session.Error = ""
	session.Name = ""
	session.Password = ""
	session.PasswordConfirm = ""

	//languages := mnemonics.Language_List()

	errorText := canvas.NewText(" ", colors.Green)
	errorText.TextSize = 12
	errorText.Alignment = fyne.TextAlignCenter

	btnCreate := widget.NewButton("Recover", nil)
	btnCreate.Disable()

	btnCancel := widget.NewButton("Cancel", func() {
		session.Domain = "app.main"
		session.Error = ""
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutMain())
	})

	btnClose := widget.NewButton("Close", func() {
		session.Domain = "app.main"
		session.Error = ""
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutMain())
	})

	btnCopyAddress := widget.NewButton("Copy Address", nil)

	wPassword := newRtnEntry()
	// Set password masking
	wPassword.Password = true
	// OnChange assign password to session variable
	// Remember to clear it after opening the wallet
	wPassword.OnChanged = func(s string) {
		session.Error = ""
		errorText.Text = ""
		errorText.Refresh()
		session.Password = s

		if len(session.Password) > 0 && session.Password == session.PasswordConfirm && session.Name != "" {
			btnCreate.Enable()
			btnCreate.Refresh()
		} else {
			btnCreate.Disable()
			btnCreate.Refresh()
		}
	}
	wPassword.SetPlaceHolder("Password")
	wPassword.Wrapping = fyne.TextTruncate

	wPasswordConfirm := newRtnEntry()
	// Set password masking
	wPasswordConfirm.Password = true
	// OnChange assign password to session variable
	// Remember to clear it after opening the wallet
	wPasswordConfirm.OnChanged = func(s string) {
		session.Error = ""
		errorText.Text = ""
		errorText.Refresh()
		session.PasswordConfirm = s

		if len(session.Password) > 0 && session.Password == session.PasswordConfirm && session.Name != "" {
			btnCreate.Enable()
			btnCreate.Refresh()
		} else {
			btnCreate.Disable()
			btnCreate.Refresh()
		}
	}
	wPasswordConfirm.SetPlaceHolder("Confirm Password")
	wPasswordConfirm.Wrapping = fyne.TextTruncate

	wAccount := widget.NewEntry()

	/*
		wLanguage := widget.NewSelect(languages, nil)
		wLanguage.OnChanged = func(s string) {
			index := wLanguage.SelectedIndex()
			session.Language = index
			session.Window.Canvas().Focus(wAccount)
		}
		wLanguage.PlaceHolder = "(Select Language)"
	*/

	wAccount.SetPlaceHolder("Account Name")
	wAccount.Wrapping = fyne.TextTruncate
	wAccount.Validator = func(s string) (err error) {
		session.Error = ""
		errorText.Text = ""
		errorText.Refresh()

		if len(s) > 30 {
			err = errors.New("Account name is too long (max 30 characters).")
			wAccount.SetText(err.Error())
			wAccount.Refresh()
			return
		}

		checkDir()
		getNetwork()
		if !session.Network {
			session.Path = AppPath() + string(filepath.Separator) + "testnet" + string(filepath.Separator) + s + ".db"
		} else {
			session.Path = AppPath() + string(filepath.Separator) + "mainnet" + string(filepath.Separator) + s + ".db"
		}
		session.Name = s

		if findAccount() {
			err = errors.New("Account name already exists.")
			errorText.Text = err.Error()
			errorText.Refresh()
			return
		}

		if len(session.Password) > 0 && session.Password == session.PasswordConfirm && session.Name != "" {

		} else {
			btnCreate.Disable()
			btnCreate.Refresh()
		}
		return nil
	}
	wAccount.OnChanged = func(s string) {
		wAccount.Validate()
	}

	wSpacer := widget.NewLabel(" ")
	heading := canvas.NewText("Recover Account", colors.Green)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	heading2 := canvas.NewText("Success", colors.Green)
	heading2.TextSize = 22
	heading2.Alignment = fyne.TextAlignCenter
	heading2.TextStyle = fyne.TextStyle{Bold: true}

	rectStatus := canvas.NewRectangle(color.Transparent)
	rectStatus.SetMinSize(fyne.NewSize(10, 10))

	title := canvas.NewText("R E C O V E R", colors.Gray)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 16

	res.rpc_header.SetMinSize(fyne.NewSize(300, 80))
	res.rpc_footer.SetMinSize(fyne.NewSize(300, 80))

	frame := canvas.NewRectangle(color.Transparent)
	frame.SetMinSize(fyne.NewSize(MIN_WIDTH, MIN_HEIGHT))

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(300, 5))

	status.Connection.FillColor = colors.Gray
	status.Cyberdeck.FillColor = colors.Gray
	status.Sync.FillColor = colors.Gray

	grid := container.NewAdaptiveGrid(1)
	grid.Objects = nil

	word1 := widget.NewEntry()
	word1.PlaceHolder = "Seed Word 1"
	word1.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("Invalid seed word")
		}
		seed[0] = s
		return nil
	}
	word1.OnChanged = func(s string) {
		word1.Validate()
	}

	word2 := widget.NewEntry()
	word2.PlaceHolder = "Seed Word 2"
	word2.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("Invalid seed word")
		}
		seed[1] = s

		var list []string
		for s := range seed {
			if seed[s] != "" {
				list = append(list, seed[s])
			}
		}

		if len(list) == 25 {
			btnCreate.Enable()
		}

		return nil
	}
	word2.OnChanged = func(s string) {
		word2.Validate()
	}

	word3 := widget.NewEntry()
	word3.PlaceHolder = "Seed Word 3"
	word3.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("Invalid seed word")
		}
		seed[2] = s

		var list []string
		for s := range seed {
			if seed[s] != "" {
				list = append(list, seed[s])
			}
		}

		if len(list) == 25 {
			btnCreate.Enable()
		}

		return nil
	}
	word3.OnChanged = func(s string) {
		word3.Validate()
	}

	word4 := widget.NewEntry()
	word4.PlaceHolder = "Seed Word 4"
	word4.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("Invalid seed word")
		}
		seed[3] = s

		var list []string
		for s := range seed {
			if seed[s] != "" {
				list = append(list, seed[s])
			}
		}

		if len(list) == 25 {
			btnCreate.Enable()
		}

		return nil
	}
	word4.OnChanged = func(s string) {
		word4.Validate()
	}

	word5 := widget.NewEntry()
	word5.PlaceHolder = "Seed Word 5"
	word5.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("Invalid seed word")
		}
		seed[4] = s

		var list []string
		for s := range seed {
			if seed[s] != "" {
				list = append(list, seed[s])
			}
		}

		if len(list) == 25 {
			btnCreate.Enable()
		}

		return nil
	}
	word5.OnChanged = func(s string) {
		word5.Validate()
	}

	word6 := widget.NewEntry()
	word6.PlaceHolder = "Seed Word 6"
	word6.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("Invalid seed word")
		}
		seed[5] = s

		var list []string
		for s := range seed {
			if seed[s] != "" {
				list = append(list, seed[s])
			}
		}

		if len(list) == 25 {
			btnCreate.Enable()
		}

		return nil
	}
	word6.OnChanged = func(s string) {
		word6.Validate()
	}

	word7 := widget.NewEntry()
	word7.PlaceHolder = "Seed Word 7"
	word7.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("Invalid seed word")
		}
		seed[6] = s

		var list []string
		for s := range seed {
			if seed[s] != "" {
				list = append(list, seed[s])
			}
		}

		if len(list) == 25 {
			btnCreate.Enable()
		}

		return nil
	}
	word7.OnChanged = func(s string) {
		word7.Validate()
	}

	word8 := widget.NewEntry()
	word8.PlaceHolder = "Seed Word 8"
	word8.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("Invalid seed word")
		}
		seed[7] = s

		var list []string
		for s := range seed {
			if seed[s] != "" {
				list = append(list, seed[s])
			}
		}

		if len(list) == 25 {
			btnCreate.Enable()
		}

		return nil
	}
	word8.OnChanged = func(s string) {
		word8.Validate()
	}

	word9 := widget.NewEntry()
	word9.PlaceHolder = "Seed Word 9"
	word9.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("Invalid seed word")
		}
		seed[8] = s

		var list []string
		for s := range seed {
			if seed[s] != "" {
				list = append(list, seed[s])
			}
		}

		if len(list) == 25 {
			btnCreate.Enable()
		}

		return nil
	}
	word9.OnChanged = func(s string) {
		word9.Validate()
	}

	word10 := widget.NewEntry()
	word10.PlaceHolder = "Seed Word 10"
	word10.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("Invalid seed word")
		}
		seed[9] = s

		var list []string
		for s := range seed {
			if seed[s] != "" {
				list = append(list, seed[s])
			}
		}

		if len(list) == 25 {
			btnCreate.Enable()
		}

		return nil
	}
	word10.OnChanged = func(s string) {
		word10.Validate()
	}

	word11 := widget.NewEntry()
	word11.PlaceHolder = "Seed Word 11"
	word11.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("Invalid seed word")
		}
		seed[10] = s

		var list []string
		for s := range seed {
			if seed[s] != "" {
				list = append(list, seed[s])
			}
		}

		if len(list) == 25 {
			btnCreate.Enable()
		}

		return nil
	}
	word11.OnChanged = func(s string) {
		word11.Validate()
	}

	word12 := widget.NewEntry()
	word12.PlaceHolder = "Seed Word 12"
	word12.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("Invalid seed word")
		}
		seed[11] = s

		var list []string
		for s := range seed {
			if seed[s] != "" {
				list = append(list, seed[s])
			}
		}

		if len(list) == 25 {
			btnCreate.Enable()
		}

		return nil
	}
	word12.OnChanged = func(s string) {
		word12.Validate()
	}

	word13 := widget.NewEntry()
	word13.PlaceHolder = "Seed Word 13"
	word13.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("Invalid seed word")
		}
		seed[12] = s

		var list []string
		for s := range seed {
			if seed[s] != "" {
				list = append(list, seed[s])
			}
		}

		if len(list) == 25 {
			btnCreate.Enable()
		}

		return nil
	}
	word13.OnChanged = func(s string) {
		word13.Validate()
	}

	word14 := widget.NewEntry()
	word14.PlaceHolder = "Seed Word 14"
	word14.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("Invalid seed word")
		}
		seed[13] = s

		var list []string
		for s := range seed {
			if seed[s] != "" {
				list = append(list, seed[s])
			}
		}

		if len(list) == 25 {
			btnCreate.Enable()
		}

		return nil
	}
	word14.OnChanged = func(s string) {
		word14.Validate()
	}

	word15 := widget.NewEntry()
	word15.PlaceHolder = "Seed Word 15"
	word15.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("Invalid seed word")
		}
		seed[14] = s

		var list []string
		for s := range seed {
			if seed[s] != "" {
				list = append(list, seed[s])
			}
		}

		if len(list) == 25 {
			btnCreate.Enable()
		}

		return nil
	}
	word15.OnChanged = func(s string) {
		word15.Validate()
	}

	word16 := widget.NewEntry()
	word16.PlaceHolder = "Seed Word 16"
	word16.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("Invalid seed word")
		}
		seed[15] = s

		var list []string
		for s := range seed {
			if seed[s] != "" {
				list = append(list, seed[s])
			}
		}

		if len(list) == 25 {
			btnCreate.Enable()
		}

		return nil
	}
	word16.OnChanged = func(s string) {
		word16.Validate()
	}

	word17 := widget.NewEntry()
	word17.PlaceHolder = "Seed Word 17"
	word17.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("Invalid seed word")
		}
		seed[16] = s

		var list []string
		for s := range seed {
			if seed[s] != "" {
				list = append(list, seed[s])
			}
		}

		if len(list) == 25 {
			btnCreate.Enable()
		}

		return nil
	}
	word17.OnChanged = func(s string) {
		word17.Validate()
	}

	word18 := widget.NewEntry()
	word18.PlaceHolder = "Seed Word 18"
	word18.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("Invalid seed word")
		}
		seed[17] = s

		var list []string
		for s := range seed {
			if seed[s] != "" {
				list = append(list, seed[s])
			}
		}

		if len(list) == 25 {
			btnCreate.Enable()
		}

		return nil
	}
	word18.OnChanged = func(s string) {
		word18.Validate()
	}

	word19 := widget.NewEntry()
	word19.PlaceHolder = "Seed Word 19"
	word19.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("Invalid seed word")
		}
		seed[18] = s

		var list []string
		for s := range seed {
			if seed[s] != "" {
				list = append(list, seed[s])
			}
		}

		if len(list) == 25 {
			btnCreate.Enable()
		}

		return nil
	}
	word19.OnChanged = func(s string) {
		word19.Validate()
	}

	word20 := widget.NewEntry()
	word20.PlaceHolder = "Seed Word 20"
	word20.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("Invalid seed word")
		}
		seed[19] = s
		return nil
	}
	word20.OnChanged = func(s string) {
		word20.Validate()
	}

	word21 := widget.NewEntry()
	word21.PlaceHolder = "Seed Word 21"
	word21.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("Invalid seed word")
		}
		seed[20] = s

		var list []string
		for s := range seed {
			if seed[s] != "" {
				list = append(list, seed[s])
			}
		}

		if len(list) == 25 {
			btnCreate.Enable()
		}

		return nil
	}
	word21.OnChanged = func(s string) {
		word21.Validate()
	}

	word22 := widget.NewEntry()
	word22.PlaceHolder = "Seed Word 22"
	word22.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("Invalid seed word")
		}
		seed[21] = s

		var list []string
		for s := range seed {
			if seed[s] != "" {
				list = append(list, seed[s])
			}
		}

		if len(list) == 25 {
			btnCreate.Enable()
		}

		return nil
	}
	word22.OnChanged = func(s string) {
		word22.Validate()
	}

	word23 := widget.NewEntry()
	word23.PlaceHolder = "Seed Word 23"
	word23.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("Invalid seed word")
		}
		seed[22] = s

		var list []string
		for s := range seed {
			if seed[s] != "" {
				list = append(list, seed[s])
			}
		}

		if len(list) == 25 {
			btnCreate.Enable()
		}

		return nil
	}
	word23.OnChanged = func(s string) {
		word23.Validate()
	}

	word24 := widget.NewEntry()
	word24.PlaceHolder = "Seed Word 24"
	word24.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("Invalid seed word")
		}
		seed[23] = s

		var list []string
		for s := range seed {
			if seed[s] != "" {
				list = append(list, seed[s])
			}
		}

		if len(list) == 25 {
			btnCreate.Enable()
		}

		return nil
	}
	word24.OnChanged = func(s string) {
		word24.Validate()
	}

	word25 := widget.NewEntry()
	word25.PlaceHolder = "Seed Word 25"
	word25.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("Invalid seed word")
		}
		seed[24] = s

		var list []string
		for s := range seed {
			if seed[s] != "" {
				list = append(list, seed[s])
			}
		}

		if len(list) == 25 {
			btnCreate.Enable()
		}

		return nil
	}
	word25.OnChanged = func(s string) {
		word25.Validate()
	}

	// Create a new form for account/password inputs
	form := container.NewVBox(
		wSpacer,
		container.NewCenter(
			res.rpc_header,
			container.NewVBox(
				container.NewCenter(title),
				rectSpacer,
			),
		),
		rectSpacer,
		heading,
		wSpacer,
		//wLanguage,
		wAccount,
		wPassword,
		wPasswordConfirm,
		rectSpacer,
		rectSpacer,
		word1,
		rectSpacer,
		word2,
		rectSpacer,
		word3,
		rectSpacer,
		word4,
		rectSpacer,
		word5,
		rectSpacer,
		word6,
		rectSpacer,
		word7,
		rectSpacer,
		word8,
		rectSpacer,
		word9,
		rectSpacer,
		word10,
		rectSpacer,
		word11,
		rectSpacer,
		word12,
		rectSpacer,
		word13,
		rectSpacer,
		word14,
		rectSpacer,
		word15,
		rectSpacer,
		word16,
		rectSpacer,
		word17,
		rectSpacer,
		word18,
		rectSpacer,
		word19,
		rectSpacer,
		word20,
		rectSpacer,
		word21,
		rectSpacer,
		word22,
		rectSpacer,
		word23,
		rectSpacer,
		word24,
		rectSpacer,
		word25,
		rectSpacer,
		errorText,
		wSpacer,
		btnCreate,
		btnCancel,
		rectSpacer,
		res.rpc_footer,
	)

	rectForm := canvas.NewRectangle(color.Transparent)
	rectForm.SetMinSize(fyne.NewSize(300, 750))

	body := widget.NewLabel("Your account has been successfully recovered. ")
	body.Wrapping = fyne.TextWrapWord
	body.Alignment = fyne.TextAlignCenter
	body.TextStyle = fyne.TextStyle{Bold: true}

	formSuccess := container.NewVBox(
		wSpacer,
		container.NewCenter(
			res.rpc_header,
			container.NewVBox(
				container.NewCenter(title),
				rectSpacer,
			),
		),
		rectSpacer,
		heading2,
		rectSpacer,
		body,
		wSpacer,
		container.NewCenter(grid),
		rectSpacer,
		errorText,
		rectSpacer,
		btnCopyAddress,
		btnClose,
		rectSpacer,
		res.rpc_footer,
	)

	formSuccess.Hide()

	scrollBox := container.NewVScroll(
		container.NewHBox(
			layout.NewSpacer(),
			container.NewMax(
				formSuccess,
				form,
			),
			layout.NewSpacer(),
		),
	)

	scrollBox.SetMinSize(fyne.NewSize(300, 740))

	btnCreate.OnTapped = func() {
		if engram.Disk != nil {
			closeWallet()
		}
		/*
			if wLanguage.SelectedIndex() == -1 {
				errorText.Text = "Please select a language and try again."
				errorText.Refresh()
				return
			}
		*/

		var err error

		if findAccount() {
			err = errors.New("Account name already exists.")
			errorText.Text = err.Error()
			errorText.Color = colors.Red
			errorText.Refresh()
			return
		} else {
			errorText.Text = ""
			errorText.Refresh()
		}

		getNetwork()

		var format []string
		var words string

		for w := range seed {
			format = append(format, seed[w])
			words += seed[w] + " "
		}

		language, _, err := mnemonics.Words_To_Key(words)

		/*
			prefix := mnemonics.Languages[session.Language].Unique_Prefix_Length
			check := mnemonics.Verify_Checksum(format, prefix)

			if !check {
				errorText.Text = "Check your recovery words and try again."
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}
		*/

		account, err := walletapi.Generate_Account_From_Recovery_Words(words)
		if err != nil {
			logger.Error(err, "Error while recovering seed.")
			return
		}

		//engram.Disk, err = walletapi.Create_Encrypted_Wallet_From_Recovery_Words(session.Path, session.Password, words)
		temp, err := walletapi.Create_Encrypted_Wallet(session.Path, session.Password, account.Keys.Secret)
		if err != nil {
			errorText.Text = err.Error()
			errorText.Color = colors.Red
			errorText.Refresh()
			return
		}

		engram.Disk = temp
		temp = nil

		if !session.Network {
			engram.Disk.SetNetwork(false)
		} else {
			engram.Disk.SetNetwork(true)
		}

		engram.Disk.SetSeedLanguage(language)

		address := engram.Disk.GetAddress().String()

		btnCopyAddress.OnTapped = func() {
			session.Window.Clipboard().SetContent(address)
		}

		engram.Disk.SetDaemonAddress(getDaemon())
		engram.Disk.SetOnlineMode()
		engram.Disk.Get_Balance_Rescan()
		engram.Disk.Close_Encrypted_Wallet()

		session.WalletOpen = false
		engram.Disk = nil
		session.Path = ""
		session.Name = ""
		tx = Transfers{}

		form.Hide()
		form.Refresh()
		formSuccess.Show()
		formSuccess.Refresh()
		grid.Refresh()
		scrollBox.Refresh()
		session.Window.Canvas().Content().Refresh()
		session.Window.Canvas().Refresh(session.Window.Content())
	}

	layout := container.NewMax(
		frame,
		container.NewVBox(
			scrollBox,
			layout.NewSpacer(),
			container.NewVBox(
				container.NewHBox(
					layout.NewSpacer(),
					container.NewMax(
						rectStatus,
						status.Connection,
					),
					widget.NewLabel(""),
					container.NewMax(
						rectStatus,
						status.Sync,
					),
					widget.NewLabel(""),
					container.NewMax(
						rectStatus,
						status.Cyberdeck,
					),
					layout.NewSpacer(),
				),
				rectSpacer,
			),
		),
	)
	return layout
}

func layoutTransfers() fyne.CanvasObject {
	resetResources()

	wSpacer := widget.NewLabel(" ")
	sendTitle := canvas.NewText("T R A N S F E R S", colors.Gray)
	sendTitle.TextStyle = fyne.TextStyle{Bold: true}
	sendTitle.TextSize = 16

	sendDesc := canvas.NewText("", colors.Gray)
	sendDesc.TextSize = 18
	sendDesc.Alignment = fyne.TextAlignCenter
	sendDesc.TextStyle = fyne.TextStyle{Bold: true}

	sendHeading := canvas.NewText("Saved Transfers", colors.Green)
	sendHeading.TextSize = 22
	sendHeading.Alignment = fyne.TextAlignCenter
	sendHeading.TextStyle = fyne.TextStyle{Bold: true}

	rectSynapse := canvas.NewRectangle(color.Transparent)
	rectSynapse.SetMinSize(fyne.NewSize(75, 75))
	rectCenter := canvas.NewRectangle(colors.DarkMatter)
	rectCenter.FillColor = colors.DarkMatter
	rectCenter.SetMinSize(fyne.NewSize(10, 10))
	rectUp := canvas.NewRectangle(colors.Green)
	rectUp.SetMinSize(fyne.NewSize(10, 10))
	rectDown := canvas.NewRectangle(colors.DarkMatter)
	rectDown.SetMinSize(fyne.NewSize(10, 10))
	rectLeft := canvas.NewRectangle(colors.DarkMatter)
	rectLeft.SetMinSize(fyne.NewSize(10, 10))
	rectRight := canvas.NewRectangle(colors.DarkMatter)
	rectRight.SetMinSize(fyne.NewSize(10, 10))
	rectStatus := canvas.NewRectangle(color.Transparent)
	rectStatus.SetMinSize(fyne.NewSize(10, 10))
	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(fyne.NewSize(MIN_WIDTH, 20))
	frame := canvas.NewRectangle(color.Transparent)
	frame.SetMinSize(fyne.NewSize(MIN_WIDTH, MIN_HEIGHT))
	rect.SetMinSize(fyne.NewSize(300, 30))
	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(10, 5))
	rect.SetMinSize(fyne.NewSize(10, 10))
	rectEmpty := canvas.NewRectangle(color.Transparent)
	rectEmpty.SetMinSize(fyne.NewSize(10, 10))
	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(fyne.NewSize(300, 35))
	rectListBox := canvas.NewRectangle(color.Transparent)
	rectListBox.SetMinSize(fyne.NewSize(300, 200))

	res.gram.SetMinSize(fyne.NewSize(300, 185))
	res.gram_footer.SetMinSize(fyne.NewSize(300, 80))
	res.rpc_header.SetMinSize(fyne.NewSize(300, 80))
	res.rpc_footer.SetMinSize(fyne.NewSize(300, 20))
	res.home_footer.SetMinSize(fyne.NewSize(300, 20))
	res.nft_footer.SetMinSize(fyne.NewSize(300, 20))

	var pendingList []string

	for i := 0; i < len(tx.Pending); i++ {
		pendingList = append(pendingList, strconv.Itoa(i)+","+globals.FormatMoney(tx.Pending[i].Amount)+","+tx.Pending[i].Destination)
	}

	data := binding.BindStringList(&pendingList)

	scrollBox := widget.NewListWithData(data,
		func() fyne.CanvasObject {
			c := container.NewMax(
				rectList,
				container.NewHBox(
					canvas.NewText("", colors.Account),
					layout.NewSpacer(),
					canvas.NewText("", colors.Account),
					layout.NewSpacer(),
					widget.NewButton("X", nil),
				),
			)
			return c
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			dat := di.(binding.String)
			str, err := dat.Get()
			if err != nil {
				fmt.Printf("Error: %s\n", err)
			}
			dataItem := strings.SplitN(str, ",", 3)
			dest := dataItem[2]
			dest = " ..." + dest[len(dataItem[2])-5:]
			index, _ := strconv.Atoi(dataItem[0])
			//co.(*fyne.Container).Objects[1].(*fyne.Container).Objects[0].(*widget.Label).Bind(dataItem[0])
			co.(*fyne.Container).Objects[1].(*fyne.Container).Objects[0].(*canvas.Text).Text = dest
			co.(*fyne.Container).Objects[1].(*fyne.Container).Objects[0].(*canvas.Text).TextSize = 17
			co.(*fyne.Container).Objects[1].(*fyne.Container).Objects[0].(*canvas.Text).TextStyle.Bold = true
			co.(*fyne.Container).Objects[1].(*fyne.Container).Objects[2].(*canvas.Text).Text = dataItem[1]
			co.(*fyne.Container).Objects[1].(*fyne.Container).Objects[2].(*canvas.Text).TextSize = 17
			co.(*fyne.Container).Objects[1].(*fyne.Container).Objects[2].(*canvas.Text).TextStyle.Bold = true
			co.(*fyne.Container).Objects[1].(*fyne.Container).Objects[4].(*widget.Button).OnTapped = func() {
				if len(pendingList) > index+1 {
					pendingList = append(pendingList[:index], pendingList[index+1:]...)
					tx.Pending = append(tx.Pending[:index], tx.Pending[index+1:]...)
					data.Reload()
				} else if len(pendingList) == 1 {
					pendingList = pendingList[:0]
					tx = Transfers{}
					data.Reload()
				} else {
					pendingList = append(pendingList[:index])
					tx.Pending = append(tx.Pending[:index])
					data.Reload()
				}
			}
		})

	scrollBox.OnSelected = func(id widget.ListItemID) {
		scrollBox.UnselectAll()
	}

	btnSend := widget.NewButton("Send All", func() {
		if len(tx.Pending) == 0 {
			return
		} else {
			err := sendTransfers()
			if err != nil {
				return
			}

			pendingList = pendingList[:0]
			data.Reload()
		}
	})

	btnHistory := widget.NewButton("View History", func() {
		if history.Window != nil {
			history.Window.Show()
		} else {
			history.Window = fyne.CurrentApp().NewWindow("View History")
			history.Window.SetCloseIntercept(func() {
				history.Window.Close()
				history.Window = nil
			})
			history.Window.Resize(fyne.NewSize(800, MIN_HEIGHT))
			history.Window.SetFixedSize(true)
			history.Window.SetContent(layoutHistory())
			history.Window.Show()
		}
	})

	btnCancel := widget.NewButton("Clear", func() {
		pendingList = pendingList[:0]
		tx = Transfers{}
		data.Reload()
	})

	session.Window.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
		if session.Domain != "app.wallet" {
			return
		}

		if k.Name == fyne.KeyDown {
			session.Dashboard = "main"

			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutDashboard())
		}
	})

	synapse := container.NewMax(
		rectSynapse,
		container.NewGridWithColumns(3,
			rectEmpty,
			rectUp,
			rectEmpty,
			rectLeft,
			rectCenter,
			rectRight,
			rectEmpty,
			rectDown,
			rectEmpty,
		),
	)

	sendForm := container.NewVBox(
		wSpacer,
		container.NewCenter(res.rpc_header, container.NewVBox(sendTitle, rectSpacer)),
		rectSpacer,
		sendHeading,
		rectSpacer,
		//sendDesc,
		wSpacer,
		container.NewMax(
			rectListBox,
			scrollBox,
		),
		wSpacer,
		btnSend,
		btnHistory,
		btnCancel,
		wSpacer,
		res.rpc_footer,
	)

	gridItem1 := container.NewCenter(
		sendForm,
	)

	gridItem2 := container.NewCenter()

	gridItem3 := container.NewCenter()

	gridItem4 := container.NewCenter()

	gridItem1.Hidden = false
	gridItem2.Hidden = true
	gridItem3.Hidden = true
	gridItem4.Hidden = true

	features := container.NewCenter(
		layout.NewSpacer(),
		gridItem1,
		layout.NewSpacer(),
		gridItem2,
		layout.NewSpacer(),
		gridItem3,
		layout.NewSpacer(),
		gridItem4,
		layout.NewSpacer(),
	)

	subContainer := container.NewMax(
		container.NewVBox(
			container.NewCenter(
				rectSynapse,
				synapse,
			),
			rectSpacer,
			container.NewHBox(
				layout.NewSpacer(),
				container.NewMax(
					rectStatus,
					status.Connection,
				),
				widget.NewLabel(""),
				container.NewMax(
					rectStatus,
					status.Sync,
				),
				widget.NewLabel(""),
				container.NewMax(
					rectStatus,
					status.Cyberdeck,
				),
				layout.NewSpacer(),
			),
			rectSpacer,
		),
	)

	c := container.NewBorder(
		features,
		subContainer,
		nil,
		nil,
	)

	layout := container.NewMax(
		frame,
		c,
	)

	return layout
}

func layoutTransition() fyne.CanvasObject {
	frame := canvas.NewRectangle(color.Transparent)
	frame.SetMinSize(fyne.NewSize(MIN_WIDTH, MIN_HEIGHT))
	layout := container.NewMax(frame)
	resizeWindow(MIN_WIDTH, MIN_HEIGHT)

	return layout
}

func layoutSettings() fyne.CanvasObject {
	resetResources()

	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(fyne.NewSize(200, 233))
	rectScroll := canvas.NewRectangle(color.Transparent)
	rectScroll.SetMinSize(fyne.NewSize(300, 450))
	frame := canvas.NewRectangle(color.Transparent)
	frame.SetMinSize(fyne.NewSize(MIN_WIDTH, MIN_HEIGHT))
	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(10, 5))

	title := canvas.NewText("S E T T I N G S", colors.Gray)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 16

	heading := canvas.NewText("My Options", colors.Green)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	catNetwork := canvas.NewText("NETWORK", colors.Gray)
	catNetwork.TextStyle = fyne.TextStyle{Bold: true}
	catNetwork.TextSize = 14

	catNode := canvas.NewText("CONNECTION", colors.Gray)
	catNode.TextStyle = fyne.TextStyle{Bold: true}
	catNode.TextSize = 14

	catSecurity := canvas.NewText("SECURITY", colors.Gray)
	catSecurity.TextStyle = fyne.TextStyle{Bold: true}
	catSecurity.TextSize = 14

	catGnomon := canvas.NewText("GNOMON", colors.Gray)
	catGnomon.TextStyle = fyne.TextStyle{Bold: true}
	catGnomon.TextSize = 14

	cg := widget.NewRichTextWithText("Gnomon scans and indexes blockchain data in order to unlock more features, like native asset tracking.")
	cg.Wrapping = fyne.TextWrapWord

	cd := widget.NewRichTextWithText("A username and password is required in order to allow application connectivity.")
	cd.Wrapping = fyne.TextWrapWord

	btnCancel := widget.NewButton("Back", nil)
	btnRestore := widget.NewButton("Restore Defaults", nil)

	address := widget.NewEntry()
	address.OnChanged = func(s string) {
		setDaemon(s)
	}
	address.PlaceHolder = "Daemon Address"
	address.SetText(getDaemon())
	address.Refresh()

	network := widget.NewRadioGroup([]string{"Mainnet", "Testnet"}, nil)
	network.Horizontal = false
	//network.Required = true
	network.OnChanged = func(s string) {
		if s == "Testnet" {
			setNetwork(false)
		} else {
			setNetwork(true)
		}
	}

	net, _ := GetValue("settings", []byte("network"))

	if string(net) == "Testnet" {
		network.SetSelected("Testnet")
	} else {
		network.SetSelected("Mainnet")
	}

	network.Refresh()

	nodeType := widget.NewRadioGroup([]string{"Remote", "Local"}, nil)
	nodeType.Horizontal = false
	nodeType.Required = true
	nodeType.OnChanged = func(s string) {
		if s == "Local" {
			if string(net) == "Testnet" {
				setType("Local")
				setDaemon(DEFAULT_LOCAL_TESTNET_DAEMON)
				address.SetText(DEFAULT_LOCAL_TESTNET_DAEMON)
				address.Disable()
				address.Refresh()
			} else {
				setType("Remote")
				address.SetText(DEFAULT_LOCAL_DAEMON)
				address.Enable()
				address.Refresh()
			}
		} else {
			address.SetText(getDaemon())
			address.Enable()
			address.Refresh()
		}
	}

	t := getType()
	if t == "Remote" {
		nodeType.SetSelected("Remote")
		nodeType.Refresh()
	} else {
		nodeType.SetSelected("Local")
		nodeType.Refresh()
	}

	user := widget.NewEntry()
	user.PlaceHolder = "Username"
	user.SetText(cyberdeck.user)

	pass := widget.NewEntry()
	pass.PlaceHolder = "Password"
	pass.Password = true
	pass.SetText(cyberdeck.pass)

	user.OnChanged = func(s string) {
		cyberdeck.user = s
	}

	pass.OnChanged = func(s string) {
		cyberdeck.pass = s
	}

	checkbox := widget.NewCheck("Use Authenticator", nil)
	checkbox.OnChanged = func(b bool) {
		if b {
			StoreValue("settings", []byte("auth_mode"), []byte("true"))
			checkbox.Checked = true
			cyberdeck.mode = 1
			user.Disable()
			pass.Disable()
		} else {
			StoreValue("settings", []byte("auth_mode"), []byte("false"))
			checkbox.Checked = false
			cyberdeck.mode = 0
			user.Enable()
			pass.Enable()
		}
	}

	mode, err := GetValue("settings", []byte("auth_mode"))
	if err != nil {
		StoreValue("settings", []byte("auth_mode"), []byte("true"))
		checkbox.Checked = true
	} else {
		if string(mode) == "true" {
			checkbox.Checked = true
			cyberdeck.mode = 1
			user.Disable()
			pass.Disable()
		} else {
			checkbox.Checked = false
			cyberdeck.mode = 0
			user.Enable()
			pass.Enable()
		}
	}

	checkGnomon := widget.NewCheck("Enable Gnomon", nil)
	checkGnomon.OnChanged = func(b bool) {
		if b {
			StoreValue("settings", []byte("gnomon"), []byte("1"))
			checkGnomon.Checked = true
			gnomon.Active = 1
		} else {
			StoreValue("settings", []byte("gnomon"), []byte("0"))
			checkGnomon.Checked = false
			gnomon.Active = 0
		}
	}

	gmn, err := GetValue("settings", []byte("gnomon"))
	if err != nil {
		gnomon.Active = 1
		StoreValue("settings", []byte("gnomon"), []byte("1"))
		checkGnomon.Checked = true
	}

	if string(gmn) == "1" {
		checkGnomon.Checked = true
	} else {
		checkGnomon.Checked = false
	}

	btnCancel.OnTapped = func() {
		if network.Selected == "Testnet" {
			setNetwork(false)
		} else {
			setNetwork(true)
		}
		setType(nodeType.Selected)
		setDaemon(address.Text)

		resizeWindow(MIN_WIDTH, MIN_HEIGHT)
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutMain())
	}

	btnRestore.OnTapped = func() {
		setNetwork(true)
		setType("Remote")
		setDaemon(DEFAULT_REMOTE_DAEMON)
		setAuthMode("true")
		setGnomon("1")

		resizeWindow(MIN_WIDTH, MIN_HEIGHT)
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutSettings())
	}

	res.gram.SetMinSize(fyne.NewSize(300, 185))
	res.gram_footer.SetMinSize(fyne.NewSize(300, 80))
	res.rpc_header.SetMinSize(fyne.NewSize(300, 80))
	res.rpc_footer.SetMinSize(fyne.NewSize(300, 20))
	res.home_footer.SetMinSize(fyne.NewSize(300, 20))
	res.nft_footer.SetMinSize(fyne.NewSize(300, 20))

	formSettings := container.NewVBox(
		catNetwork,
		rectSpacer,
		network,
		widget.NewLabel(""),
		catNode,
		rectSpacer,
		nodeType,
		rectSpacer,
		address,
		widget.NewLabel(""),
		catSecurity,
		rectSpacer,
		cd,
		rectSpacer,
		user,
		rectSpacer,
		pass,
		rectSpacer,
		checkbox,
		widget.NewLabel(""),
		catGnomon,
		rectSpacer,
		cg,
		rectSpacer,
		checkGnomon,
		widget.NewLabel(""),
		widget.NewLabel(""),
		widget.NewLabel(""),
		btnRestore,
	)

	scrollBox := container.NewVScroll(
		container.NewHBox(
			layout.NewSpacer(),
			container.NewMax(
				rectScroll,
				formSettings,
			),
			layout.NewSpacer(),
		),
	)

	scrollBox.SetMinSize(fyne.NewSize(300, 450))

	gridItem1 := container.NewCenter(
		container.NewVBox(
			widget.NewLabel(""),
			container.NewCenter(
				res.rpc_header,
				container.NewVBox(
					title,
					rectSpacer,
				),
			),
			rectSpacer,
			heading,
			widget.NewLabel(""),
			scrollBox,
			rectSpacer,
			rectSpacer,
			btnCancel,
			rectSpacer,
			rectSpacer,
			res.rpc_footer,
		),
	)

	features := container.NewCenter(
		layout.NewSpacer(),
		gridItem1,
		layout.NewSpacer(),
	)

	subContainer := container.NewMax()

	c := container.NewBorder(
		features,
		subContainer,
		nil,
		nil,
	)

	layout := container.NewMax(
		frame,
		c,
	)

	return layout
}

func layoutNetrunner() fyne.CanvasObject {
	resetResources()

	wSpacer := widget.NewLabel(" ")
	title := canvas.NewText("N E T R U N N E R", colors.Gray)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 16

	nr.Label = canvas.NewText("", colors.Green)
	nr.Label.TextSize = 28
	nr.Label.Alignment = fyne.TextAlignCenter
	nr.Label.TextStyle = fyne.TextStyle{Bold: true}

	nr.LabelBlocks = canvas.NewText("", colors.Account)
	nr.LabelBlocks.TextSize = 18
	nr.LabelBlocks.Alignment = fyne.TextAlignCenter
	nr.LabelBlocks.TextStyle = fyne.TextStyle{Bold: true}

	heading := canvas.NewText(" Idle", colors.Gray)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	rectStatus := canvas.NewRectangle(color.Transparent)
	rectStatus.SetMinSize(fyne.NewSize(10, 10))
	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(fyne.NewSize(MIN_WIDTH, 20))
	frame := canvas.NewRectangle(color.Transparent)
	frame.SetMinSize(fyne.NewSize(MIN_WIDTH, MIN_HEIGHT))
	rect.SetMinSize(fyne.NewSize(300, 30))
	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(10, 5))
	rect.SetMinSize(fyne.NewSize(10, 10))
	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(fyne.NewSize(300, 35))
	rectListBox := canvas.NewRectangle(color.Transparent)
	rectListBox.SetMinSize(fyne.NewSize(300, 200))

	res.nr_header.SetMinSize(fyne.NewSize(300, 80))
	res.nr_footer.SetMinSize(fyne.NewSize(300, 20))

	nr.Data = binding.BindStringList(&nr.BlockList)

	nr.ScrollBox = widget.NewListWithData(nr.Data,
		func() fyne.CanvasObject {
			c := container.NewMax(
				rectList,
				container.NewHBox(
					canvas.NewText("", colors.Account),
					layout.NewSpacer(),
					widget.NewButton("View", nil),
				),
			)
			return c
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			dat := di.(binding.String)
			str, err := dat.Get()
			if err != nil {
				fmt.Printf("Error: %s\n", err)
			}
			dataItem := strings.Split(str, ",")
			height := dataItem[0]
			view := dataItem[1]
			//co.(*fyne.Container).Objects[1].(*fyne.Container).Objects[0].(*widget.Label).Bind(dataItem[0])
			co.(*fyne.Container).Objects[1].(*fyne.Container).Objects[0].(*canvas.Text).Text = " " + height
			co.(*fyne.Container).Objects[1].(*fyne.Container).Objects[0].(*canvas.Text).TextSize = 17
			co.(*fyne.Container).Objects[1].(*fyne.Container).Objects[0].(*canvas.Text).TextStyle.Bold = true
			co.(*fyne.Container).Objects[1].(*fyne.Container).Objects[2].(*widget.Button).OnTapped = func() {
				if !engram.Disk.GetNetwork() {
					openURL("https://testnetexplorer.dero.io/block/"+view, nil)
				} else {
					openURL("https://explorer.dero.io/block/"+view, nil)
				}
			}

		})

	nr.ScrollBox.OnSelected = func(id widget.ListItemID) {
		nr.ScrollBox.UnselectAll()
	}

	icon := canvas.NewImageFromResource(resourceMinerOffPng)
	icon.SetMinSize(fyne.NewSize(50, 50))
	icon.FillMode = canvas.ImageFillOriginal

	btnRunner := widget.NewButton("Start", nil)
	btnRunner.OnTapped = func() {
		if nr.Mission == 0 {
			nr.Mission = 1
			port := fmt.Sprintf("%d", DEFAULT_WORK_PORT)
			if !engram.Disk.GetNetwork() {
				port = fmt.Sprintf("%d", DEFAULT_TESTNET_WORK_PORT)
			}
			split := strings.Split(session.Daemon, ":")
			daemon := split[0] + ":" + port
			icon.Resource = resourceMinerOnPng
			icon.Refresh()
			btnRunner.SetText("Stop")
			btnRunner.Refresh()
			heading.Text = " Active"
			heading.Color = colors.Green
			heading.Refresh()
			nr.Label.Text = "Starting Up..."
			nr.Label.Refresh()
			nr.LabelBlocks.Text = "Blocks:  0"
			nr.LabelBlocks.Refresh()
			go startRunner(engram.Disk, daemon, runtime.GOMAXPROCS(0)/2)
		} else {
			nr.Mission = 0
			if nr.Connection != nil {
				nr.Connection.UnderlyingConn().Close()
				nr.Connection.Close()
			}
			fmt.Printf("[Netrunner] Shutdown initiated.\n")
			icon.Resource = resourceMinerOffPng
			icon.Refresh()
			btnRunner.SetText("Start")
			btnRunner.Refresh()
			nr.Label.Text = "Job's done."
			nr.Label.Refresh()
			heading.Text = " Idle"
			heading.Color = colors.Gray
			heading.Refresh()
		}
	}

	if nr.Mission == 1 {
		icon.Resource = resourceMinerOnPng
		nr.Label.Text = " " + nr.Hashrate
		nr.LabelBlocks.Text = " Blocks: " + strconv.Itoa(int(nr.Blocks+nr.MiniBlocks))
	} else {
		icon.Resource = resourceMinerOffPng
		nr.Label.Text = ""
		nr.LabelBlocks.Text = ""
	}

	btnConfig := widget.NewButton("Configure", func() {

	})

	btnCancel := widget.NewButton("Hide", func() {
		miner.Window.Hide()
	})

	netForm := container.NewVBox(
		wSpacer,
		container.NewCenter(res.nr_header, container.NewVBox(title, rectSpacer)),
		rectSpacer,
		container.NewHBox(
			layout.NewSpacer(),
			icon,
			heading,
			layout.NewSpacer(),
		),
		rectSpacer,
		nr.Label,
		rectSpacer,
		nr.LabelBlocks,
		wSpacer,
		container.NewMax(
			rectListBox,
			nr.ScrollBox,
		),
		wSpacer,
		btnRunner,
		btnConfig,
		btnCancel,
		wSpacer,
		res.nr_footer,
	)

	gridItem1 := container.NewCenter(
		netForm,
	)

	gridItem2 := container.NewCenter()

	gridItem3 := container.NewCenter()

	gridItem4 := container.NewCenter()

	gridItem1.Hidden = false
	gridItem2.Hidden = true
	gridItem3.Hidden = true
	gridItem4.Hidden = true

	features := container.NewCenter(
		layout.NewSpacer(),
		gridItem1,
		layout.NewSpacer(),
		gridItem2,
		layout.NewSpacer(),
		gridItem3,
		layout.NewSpacer(),
		gridItem4,
		layout.NewSpacer(),
	)

	subContainer := container.NewMax(
		container.NewVBox(
			rectSpacer,
			rectSpacer,
		),
	)

	c := container.NewBorder(
		features,
		subContainer,
		nil,
		nil,
	)

	layout := container.NewMax(
		frame,
		c,
	)

	return layout
}

func layoutMessages() fyne.CanvasObject {
	resetResources()
	wSpacer := widget.NewLabel(" ")
	title := canvas.NewText("M E S S A G E S", colors.Gray)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 16

	heading := canvas.NewText("My Contacts", colors.Green)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	rectSynapse := canvas.NewRectangle(color.Transparent)
	rectSynapse.SetMinSize(fyne.NewSize(75, 75))
	rectCenter := canvas.NewRectangle(colors.DarkMatter)
	rectCenter.FillColor = colors.DarkMatter
	rectCenter.SetMinSize(fyne.NewSize(10, 10))
	rectUp := canvas.NewRectangle(colors.DarkMatter)
	rectUp.SetMinSize(fyne.NewSize(10, 10))
	rectDown := canvas.NewRectangle(colors.Green)
	rectDown.SetMinSize(fyne.NewSize(10, 10))
	rectLeft := canvas.NewRectangle(colors.DarkMatter)
	rectLeft.SetMinSize(fyne.NewSize(10, 10))
	rectRight := canvas.NewRectangle(colors.DarkMatter)
	rectRight.SetMinSize(fyne.NewSize(10, 10))
	rectStatus := canvas.NewRectangle(color.Transparent)
	rectStatus.SetMinSize(fyne.NewSize(10, 10))
	rectEmpty := canvas.NewRectangle(color.Transparent)
	rectEmpty.SetMinSize(fyne.NewSize(10, 10))
	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(fyne.NewSize(MIN_WIDTH, 20))
	frame := canvas.NewRectangle(color.Transparent)
	frame.SetMinSize(fyne.NewSize(MIN_WIDTH, MIN_HEIGHT))
	rect.SetMinSize(fyne.NewSize(300, 30))
	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(10, 5))
	rect.SetMinSize(fyne.NewSize(10, 10))
	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(fyne.NewSize(300, 35))
	rectListBox := canvas.NewRectangle(color.Transparent)
	rectListBox.SetMinSize(fyne.NewSize(300, 257))
	rectMessage := canvas.NewRectangle(color.Transparent)
	rectMessage.SetMinSize(fyne.NewSize(60, 10))

	res.rpc_header.SetMinSize(fyne.NewSize(300, 80))
	res.rpc_footer.SetMinSize(fyne.NewSize(300, 20))

	messages.Data = nil

	data := getMessages(0)

	list := binding.BindStringList(&data)

	messages.Box = widget.NewListWithData(list,
		func() fyne.CanvasObject {
			c := container.NewMax(
				rectList,
				container.NewHBox(
					widget.NewLabel(""),
					layout.NewSpacer(),
				),
			)
			return c
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			dat := di.(binding.String)
			str, err := dat.Get()
			if err != nil {
				return
			}
			dataItem := strings.Split(str, ":")
			short := dataItem[0]
			address := short[len(short)-10:]
			username := dataItem[1]

			if username == "" {
				co.(*fyne.Container).Objects[1].(*fyne.Container).Objects[0].(*widget.Label).SetText("..." + address)
			} else {
				co.(*fyne.Container).Objects[1].(*fyne.Container).Objects[0].(*widget.Label).SetText(username)
			}
			co.(*fyne.Container).Objects[1].(*fyne.Container).Objects[0].(*widget.Label).Wrapping = fyne.TextWrapWord
			co.(*fyne.Container).Objects[1].(*fyne.Container).Objects[0].(*widget.Label).TextStyle.Bold = false
			co.(*fyne.Container).Objects[1].(*fyne.Container).Objects[0].(*widget.Label).Alignment = fyne.TextAlignLeading

		})

	messages.Box.OnSelected = func(id widget.ListItemID) {
		messages.Box.UnselectAll()
		split := strings.Split(data[id], ":")
		if split[1] == "" {
			messages.Contact = split[0]
		} else {
			messages.Contact = split[1]
		}

		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutPM())
	}

	btnSend := widget.NewButton("New Message", func() {
		_, err := globals.ParseValidateAddress(messages.Contact)
		if err != nil {
			_, err := engram.Disk.NameToAddress(messages.Contact)
			if err != nil {
				return
			}
		}

		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutPM())
	})
	btnSend.Disable()

	entryDest := widget.NewEntry()
	entryDest.PlaceHolder = "Username or Address"
	entryDest.Validator = func(s string) error {
		if len(s) > 0 {
			_, err := globals.ParseValidateAddress(s)
			if err != nil {
				btnSend.Disable()
				_, err := engram.Disk.NameToAddress(s)
				if err != nil {
					btnSend.Disable()
					return errors.New("Invalid username or address")
				} else {
					messages.Contact = s
					btnSend.Enable()
					return nil
				}
			} else {
				btnSend.Enable()
				messages.Contact = s
				return nil
			}
		}

		return errors.New("Invalid username or address")
	}

	entryDest.OnChanged = func(s string) {
		entryDest.Validate()
	}

	/*
		entryMessage := widget.NewEntry()
		entryMessage.PlaceHolder = "Message"
		entryMessage.OnChanged = func(s string) {
			messages.Message = s
			if messages.Contact == "" || messages.Message == "" {
				btnSend.Disable()
			} else {
				btnSend.Enable()
			}
		}
	*/

	messageForm := container.NewVBox(
		wSpacer,
		container.NewCenter(res.rpc_header, container.NewVBox(title, rectSpacer)),
		rectSpacer,
		container.NewHBox(
			layout.NewSpacer(),
			heading,
			layout.NewSpacer(),
		),
		wSpacer,
		container.NewMax(
			rectListBox,
			messages.Box,
		),
		rectSpacer,
		widget.NewSeparator(),
		rectSpacer,
		entryDest,
		rectSpacer,
		btnSend,
		wSpacer,
		res.rpc_footer,
	)

	gridItem1 := container.NewCenter(
		messageForm,
	)

	gridItem2 := container.NewCenter()

	gridItem3 := container.NewCenter()

	gridItem4 := container.NewCenter()

	gridItem1.Hidden = false
	gridItem2.Hidden = true
	gridItem3.Hidden = true
	gridItem4.Hidden = true

	features := container.NewCenter(
		layout.NewSpacer(),
		gridItem1,
		layout.NewSpacer(),
		gridItem2,
		layout.NewSpacer(),
		gridItem3,
		layout.NewSpacer(),
		gridItem4,
		layout.NewSpacer(),
	)

	session.Window.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
		if session.Domain != "app.wallet" {
			return
		}

		if k.Name == fyne.KeyUp {
			session.Dashboard = "main"

			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutDashboard())
		} else if k.Name == fyne.KeyF5 {
			session.Window.SetContent(layoutMessages())
		}
	})

	synapse := container.NewMax(
		rectSynapse,
		container.NewGridWithColumns(3,
			rectEmpty,
			rectUp,
			rectEmpty,
			rectLeft,
			rectCenter,
			rectRight,
			rectEmpty,
			rectDown,
			rectEmpty,
		),
	)

	subContainer := container.NewMax(
		container.NewVBox(
			container.NewCenter(
				rectSynapse,
				synapse,
			),
			rectSpacer,
			container.NewHBox(
				layout.NewSpacer(),
				container.NewMax(
					rectStatus,
					status.Connection,
				),
				widget.NewLabel(""),
				container.NewMax(
					rectStatus,
					status.Sync,
				),
				widget.NewLabel(""),
				container.NewMax(
					rectStatus,
					status.Cyberdeck,
				),
				layout.NewSpacer(),
			),
			rectSpacer,
		),
	)

	c := container.NewBorder(
		features,
		subContainer,
		nil,
		nil,
	)

	layout := container.NewMax(
		frame,
		c,
	)

	return layout
}

func layoutPM() fyne.CanvasObject {
	resetResources()

	contactAddress := ""

	_, err := globals.ParseValidateAddress(messages.Contact)
	if err != nil {
		contactAddress = messages.Contact
	} else {
		short := messages.Contact[len(messages.Contact)-10:]
		contactAddress = "..." + short
	}

	wSpacer := widget.NewLabel(" ")
	title := canvas.NewText("M E S S A G E S", colors.Gray)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 16

	heading := canvas.NewText(contactAddress, colors.Green)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	lastActive := canvas.NewText("", colors.Gray)
	lastActive.TextSize = 12
	lastActive.Alignment = fyne.TextAlignCenter
	lastActive.TextStyle = fyne.TextStyle{Bold: false}

	rectSynapse := canvas.NewRectangle(color.Transparent)
	rectSynapse.SetMinSize(fyne.NewSize(75, 75))
	rectCenter := canvas.NewRectangle(colors.DarkMatter)
	rectCenter.FillColor = colors.DarkMatter
	rectCenter.SetMinSize(fyne.NewSize(10, 10))
	rectUp := canvas.NewRectangle(colors.DarkMatter)
	rectUp.SetMinSize(fyne.NewSize(10, 10))
	rectDown := canvas.NewRectangle(colors.Blue)
	rectDown.SetMinSize(fyne.NewSize(10, 10))
	rectLeft := canvas.NewRectangle(colors.DarkMatter)
	rectLeft.SetMinSize(fyne.NewSize(10, 10))
	rectRight := canvas.NewRectangle(colors.DarkMatter)
	rectRight.SetMinSize(fyne.NewSize(10, 10))
	rectStatus := canvas.NewRectangle(color.Transparent)
	rectStatus.SetMinSize(fyne.NewSize(10, 10))
	rectEmpty := canvas.NewRectangle(color.Transparent)
	rectEmpty.SetMinSize(fyne.NewSize(10, 10))
	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(fyne.NewSize(MIN_WIDTH, 20))
	frame := canvas.NewRectangle(color.Transparent)
	frame.SetMinSize(fyne.NewSize(MIN_WIDTH, MIN_HEIGHT))
	rect.SetMinSize(fyne.NewSize(300, 30))
	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(10, 5))
	rect.SetMinSize(fyne.NewSize(10, 10))
	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(fyne.NewSize(300, 35))
	rectListBox := canvas.NewRectangle(color.Transparent)
	rectListBox.SetMinSize(fyne.NewSize(300, 250))
	rectMessage := canvas.NewRectangle(color.Transparent)
	rectMessage.SetMinSize(fyne.NewSize(60, 10))

	res.rpc_header.SetMinSize(fyne.NewSize(300, 80))
	res.rpc_footer.SetMinSize(fyne.NewSize(300, 20))

	messages.Data = nil

	chats := container.NewVBox()

	chatFrame := container.NewCenter(
		container.NewMax(
			rectListBox,
			chats,
		),
	)

	chatbox := container.NewVScroll(
		container.NewMax(chatFrame),
	)

	data := getMessagesFromUser(messages.Contact, 0)
	for d := range data {
		if data[d].Incoming {
			if data[d].Payload_RPC.Has(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString) {
				if data[d].Payload_RPC.Value(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString).(string) == "" {

				} else {
					t := data[d].Time
					time := t.Format("2006-01-02 15:04:05")
					comment := data[d].Payload_RPC.Value(rpc.RPC_COMMENT, rpc.DataString).(string)
					messages.Data = append(messages.Data, data[d].Payload_RPC.Value(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString).(string)+";;"+comment+";;"+time)
				}
			}
		} else {
			t := data[d].Time
			time := t.Format("2006-01-02 15:04:05")
			comment := data[d].Payload_RPC.Value(rpc.RPC_COMMENT, rpc.DataString).(string)
			messages.Data = append(messages.Data, engram.Disk.GetAddress().String()+";;"+comment+";;"+time)
		}
	}

	if len(data) > 0 {
		for m := range messages.Data {
			var sender string
			split := strings.Split(messages.Data[m], ";;")
			align := fyne.TextAlignLeading
			mdata := widget.NewLabel("")
			mdata.Alignment = align
			mdata.Wrapping = fyne.TextWrapWord
			boxColor := colors.Flint

			uname, err := engram.Disk.NameToAddress(split[0])
			if err != nil {
				sender = split[0]
			} else {
				sender = uname
			}

			if sender == engram.Disk.GetAddress().String() {
				boxColor = colors.Flint
				align = fyne.TextAlignTrailing
				mdata.SetText("› " + split[1])
			} else {
				boxColor = colors.DarkMatter
				align = fyne.TextAlignLeading
				mdata.SetText(split[1])
			}

			rect := canvas.NewRectangle(boxColor)
			rect.SetMinSize(mdata.MinSize())

			var entry *fyne.Container

			entry = container.NewMax(
				rect,
				mdata,
			)

			lastActive.Text = "LATEST:  " + split[2] + ""
			lastActive.Refresh()

			chats.Add(entry)
			chats.Refresh()
			chatbox.Refresh()
			chatbox.ScrollToBottom()
		}
	}

	btnSend := widget.NewButton("Send", nil)
	btnSend.Disable()

	entry := widget.NewEntry()
	entry.PlaceHolder = "Message"
	entry.OnChanged = func(s string) {
		messages.Message = s
		if messages.Message == "" {
			btnSend.Disable()
		} else {
			btnSend.Enable()
		}
	}

	btnSend.OnTapped = func() {
		if messages.Message == "" {
			return
		}
		contact := ""
		_, err := globals.ParseValidateAddress(messages.Contact)
		if err != nil {
			check, err := engram.Disk.NameToAddress(messages.Contact)
			if err != nil {
				return
			}
			contact = check
		} else {
			contact = messages.Contact
		}

		err = sendMessage(messages.Message, session.Username, contact)
		if err != nil {
			fmt.Printf("[Message] Failed to send: %s\n", err)
			return
		}

		messages.Message = ""
		entry.Text = ""
		entry.Refresh()
		btnSend.Disable()
	}

	messageForm := container.NewVBox(
		wSpacer,
		container.NewCenter(res.rpc_header, container.NewVBox(title, rectSpacer)),
		rectSpacer,
		container.NewHBox(
			layout.NewSpacer(),
			heading,
			layout.NewSpacer(),
		),
		rectSpacer,
		lastActive,
		rectSpacer,
		rectSpacer,
		container.NewMax(
			rectListBox,
			chatbox,
		),
		rectSpacer,
		widget.NewSeparator(),
		rectSpacer,
		entry,
		rectSpacer,
		btnSend,
		wSpacer,
		res.rpc_footer,
	)

	gridItem1 := container.NewCenter(
		messageForm,
	)

	gridItem2 := container.NewCenter()

	gridItem3 := container.NewCenter()

	gridItem4 := container.NewCenter()

	gridItem1.Hidden = false
	gridItem2.Hidden = true
	gridItem3.Hidden = true
	gridItem4.Hidden = true

	features := container.NewCenter(
		layout.NewSpacer(),
		gridItem1,
		layout.NewSpacer(),
		gridItem2,
		layout.NewSpacer(),
		gridItem3,
		layout.NewSpacer(),
		gridItem4,
		layout.NewSpacer(),
	)

	session.Window.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
		if session.Domain != "app.wallet" {
			return
		}

		if k.Name == fyne.KeyUp {
			session.Dashboard = "messages"
			messages.Contact = ""
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutMessages())
		} else if k.Name == fyne.KeyEscape {
			session.Dashboard = "messages"
			messages.Contact = ""
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutMessages())
		} else if k.Name == fyne.KeyF5 {
			session.Window.SetContent(layoutPM())
		}
	})

	synapse := container.NewMax(
		rectSynapse,
		container.NewGridWithColumns(3,
			rectEmpty,
			rectUp,
			rectEmpty,
			rectLeft,
			rectCenter,
			rectRight,
			rectEmpty,
			rectDown,
			rectEmpty,
		),
	)

	subContainer := container.NewMax(
		container.NewVBox(
			container.NewCenter(
				rectSynapse,
				synapse,
			),
			rectSpacer,
			container.NewHBox(
				layout.NewSpacer(),
				container.NewMax(
					rectStatus,
					status.Connection,
				),
				widget.NewLabel(""),
				container.NewMax(
					rectStatus,
					status.Sync,
				),
				widget.NewLabel(""),
				container.NewMax(
					rectStatus,
					status.Cyberdeck,
				),
				layout.NewSpacer(),
			),
			rectSpacer,
		),
	)

	c := container.NewBorder(
		features,
		subContainer,
		nil,
		nil,
	)

	layout := container.NewMax(
		frame,
		c,
	)

	return layout
}

func layoutWaiting(title *canvas.Text, heading *canvas.Text, sub *canvas.Text, btn *widget.Button) fyne.CanvasObject {
	resetResources()

	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(fyne.NewSize(200, 233))
	frame := canvas.NewRectangle(color.Transparent)
	frame.SetMinSize(fyne.NewSize(MIN_WIDTH, MIN_HEIGHT))
	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(10, 5))

	res.gram.SetMinSize(fyne.NewSize(300, 185))
	res.gram_footer.SetMinSize(fyne.NewSize(300, 80))
	res.rpc_header.SetMinSize(fyne.NewSize(300, 80))
	res.rpc_footer.SetMinSize(fyne.NewSize(300, 20))
	res.home_footer.SetMinSize(fyne.NewSize(300, 20))
	res.nft_footer.SetMinSize(fyne.NewSize(300, 20))

	session.Gif, _ = newGif(resourceAnimation2Gif)
	session.Gif.SetMinSize(rect.MinSize())
	session.Gif.Start()

	waitForm := container.NewVBox(
		widget.NewLabel(""),
		container.NewCenter(res.rpc_header, container.NewVBox(title, rectSpacer)),
		rectSpacer,
		heading,
		rectSpacer,
		sub,
		widget.NewLabel(""),
		container.NewMax(
			session.Gif,
		),
		widget.NewLabel(""),
		widget.NewLabel(""),
		widget.NewLabel(""),
		widget.NewLabel(""),
		btn,
		widget.NewLabel(""),
		res.rpc_footer,
	)

	gridItem1 := container.NewCenter(
		waitForm,
	)

	features := container.NewCenter(
		layout.NewSpacer(),
		gridItem1,
		layout.NewSpacer(),
	)

	subContainer := container.NewMax()

	c := container.NewBorder(
		features,
		subContainer,
		nil,
		nil,
	)

	layout := container.NewMax(
		frame,
		c,
	)

	return layout
}

func layoutHistory() fyne.CanvasObject {
	var data []string
	var entries []rpc.Entry
	var zeroscid crypto.Hash
	var listData binding.StringList
	var listBox *widget.List
	var txid string

	view := ""

	header := canvas.NewText("  Transaction History", colors.Green)
	header.TextSize = 22
	header.TextStyle = fyne.TextStyle{Bold: true}

	details_header := canvas.NewText("     Transaction Detail", colors.Green)
	details_header.TextSize = 22
	details_header.TextStyle = fyne.TextStyle{Bold: true}

	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(fyne.NewSize(200, 35))

	listData = binding.BindStringList(&data)
	listBox = widget.NewListWithData(listData,
		func() fyne.CanvasObject {
			return container.NewHBox(
				container.NewMax(
					rect,
					widget.NewLabel(""),
				),
				container.NewMax(
					rect,
					widget.NewLabel(""),
				),
				container.NewMax(
					rect,
					widget.NewLabel(""),
				),
				container.NewMax(
					rect,
					widget.NewLabel(""),
				),
			)
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			dat := di.(binding.String)
			str, err := dat.Get()
			if err != nil {
				return
			}

			split := strings.Split(str, ";")

			co.(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*widget.Label).SetText(split[0])
			co.(*fyne.Container).Objects[1].(*fyne.Container).Objects[1].(*widget.Label).SetText(split[1])
			co.(*fyne.Container).Objects[2].(*fyne.Container).Objects[1].(*widget.Label).SetText(split[2])
			co.(*fyne.Container).Objects[3].(*fyne.Container).Objects[1].(*widget.Label).SetText(split[3])
		})

	menu := widget.NewSelect([]string{"Normal", "Coinbase", "Smart Contracts", "Messages"}, nil)
	menu.PlaceHolder = "(Select Transaction Type)"
	menu.OnChanged = func(s string) {
		switch s {
		case "Normal":
			data = nil
			listData.Set(nil)
			entries = engram.Disk.Show_Transfers(zeroscid, false, true, true, 0, engram.Disk.Get_Height(), "", "", 0, 0)

			if entries != nil {
				go func() {
					for e := range entries {
						var height string
						var direction string
						var stamp string

						entries[e].ProcessPayload()

						if !entries[e].Coinbase {
							timefmt := entries[e].Time
							stamp = string(timefmt.Format(time.RFC822))
							height = strconv.FormatUint(entries[e].Height, 10)
							amount := ""
							txid = entries[e].TXID

							if !entries[e].Incoming {
								direction = "Sent"
								amount = "(" + globals.FormatMoney(entries[e].Amount) + ")"
							} else {
								direction = "Received"
								amount = globals.FormatMoney(entries[e].Amount)
							}

							data = append(data, direction+";"+amount+";"+height+";"+stamp+";"+txid)
						}
					}

					listData.Set(data)
					listBox.Refresh()
					listBox.ScrollToBottom()
				}()
			}
		case "Coinbase":
			data = nil
			listData.Set(nil)
			entries = engram.Disk.Show_Transfers(zeroscid, true, true, true, 0, engram.Disk.Get_Height(), "", "", 0, 0)

			if entries != nil {
				go func() {
					for e := range entries {
						var height string
						var direction string
						var stamp string

						entries[e].ProcessPayload()

						if entries[e].Coinbase {
							direction = "Miner Reward"
							timefmt := entries[e].Time
							stamp = string(timefmt.Format(time.RFC822))
							height = strconv.FormatUint(entries[e].Height, 10)
							amount := globals.FormatMoney(entries[e].Amount)
							txid = entries[e].TXID

							data = append(data, direction+";"+amount+";"+height+";"+stamp+";"+txid)
						}
					}

					listData.Set(data)
					listBox.Refresh()
					listBox.ScrollToBottom()
				}()
			}
		case "Smart Contracts":
			data = nil
			listData.Set(nil)
			if gnomon.Active == 1 && engram.Disk != nil {
				if engram.Disk.GetNetwork() {
					gnomon.DB.DBFolder = "datashards/gnomon"
				} else {
					gnomon.DB.DBFolder = "datashards/gnomon_testnet"
				}
				scList := gnomon.DB.GetAllNormalTxWithSCIDByAddr(engram.Disk.GetAddress().String())
				for sc := range scList {
					scid := crypto.HashHexToHash(scList[sc].Scid)
					height := strconv.FormatInt(scList[sc].Height, 10)
					bal, _ := engram.Disk.Get_Balance_scid(scid)
					balance := globals.FormatMoney(bal)

					data = append(data, balance+";"+scList[sc].Scid+";"+height+";")
				}

				listData.Set(data)
				listBox.Refresh()
				listBox.ScrollToBottom()
			}
			/*
				entries = engram.Disk.Show_Transfers(zeroscid, false, true, true, 0, engram.Disk.Get_Height(), "", "", 0, 0)

				if entries != nil {
					go func() {
						for e := range entries {
							var height string
							var action string
							var stamp string
							var scid string

							entries[e].ProcessPayload()

							if entries[e].Payload_RPC.HasValue("SC_ID", "H") {
								scid = entries[e].Payload_RPC.Value("SC_ID", "H").(string)

								for r := range entries[e].Payload_RPC {
									action = entries[e].Payload_RPC[r].Value.(string)
									timefmt := entries[e].Time
									stamp = string(timefmt.Format(time.RFC822))
									height = strconv.FormatUint(entries[e].Height, 10)
									txid = entries[e].TXID
									fmtSCID := "..." + scid[:10]

									data = append(data, action+";"+fmtSCID+";"+height+";"+stamp+";"+txid)
								}
							}


								tx := gnomon.DB.GetAllSCIDInvokeDetailsBySigner(entries[e].TXID, engram.Disk.GetAddress().String())

								if len(tx) > 0 {
									for t := range tx {
										action = tx[t].Entrypoint
									}

									timefmt := entries[e].Time
									stamp = string(timefmt.Format(time.RFC822))
									height = strconv.FormatUint(entries[e].Height, 10)

									data = append(data, action+";"+entries[e].TXID+";"+height+";"+stamp)
								}


						}

						listData.Set(data)
						listBox.Refresh()
						listBox.ScrollToBottom()
					}()
			*/

		case "Messages":
			data = nil
			listData.Set(nil)
			entries = engram.Disk.Get_Payments_DestinationPort(zeroscid, uint64(1337), 0)

			if entries != nil {
				go func() {
					for e := range entries {
						var stamp string
						var direction string
						var comment string

						entries[e].ProcessPayload()

						timefmt := entries[e].Time
						stamp = string(timefmt.Format(time.RFC822))

						temp := entries[e].Incoming
						if !temp {
							direction = "Sent    "
						} else {
							direction = "Received"
						}
						if entries[e].Payload_RPC.HasValue(rpc.RPC_COMMENT, rpc.DataString) {
							username := ""
							if entries[e].Payload_RPC.HasValue(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString) {
								username = entries[e].Payload_RPC.Value(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString).(string)
								if len(username) > 15 {
									username = username[0:15] + "..."
								}
							}

							comment = entries[e].Payload_RPC.Value(rpc.RPC_COMMENT, rpc.DataString).(string)
							if len(comment) > 15 {
								comment = comment[0:15] + "..."
							}

							txid = entries[e].TXID

							data = append(data, direction+";"+username+";"+comment+";"+stamp+";"+txid)
						}
					}

					listData.Set(data)
					listBox.Refresh()
					listBox.ScrollToBottom()
				}()
			}
		default:

		}
	}

	btnClose := widget.NewButton("Return", nil)

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(10, 5))
	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(fyne.NewSize(800, MIN_HEIGHT-200))
	rect1 := canvas.NewRectangle(colors.DarkMatter)
	rect1.SetMinSize(fyne.NewSize(245, 100))
	rect2 := canvas.NewRectangle(colors.DarkMatter)
	rect2.SetMinSize(fyne.NewSize(245, 40))

	label := canvas.NewText(view, colors.Account)
	label.TextSize = 15
	label.TextStyle = fyne.TextStyle{Bold: true}

	sub := canvas.NewText("   ", colors.Gray)
	sub.TextSize = 14
	sub.TextStyle = fyne.TextStyle{Bold: true}

	txidText := widget.NewEntry()
	txidText.PlaceHolder = "N/A"
	txidText.Disable()

	txidText_label := canvas.NewText("   TRANSACTION  ID", colors.Gray)
	txidText_label.TextSize = 14
	txidText_label.TextStyle = fyne.TextStyle{Bold: true}

	amount := canvas.NewText("", colors.Account)
	amount.TextSize = 34
	amount.TextStyle = fyne.TextStyle{Bold: true}

	amount_label := canvas.NewText("   AMOUNT", colors.Gray)
	amount_label.TextSize = 14
	amount_label.TextStyle = fyne.TextStyle{Bold: true}

	proof := widget.NewEntry()
	proof.PlaceHolder = "N/A"
	proof.Disable()

	proof_label := canvas.NewText("   PROOF", colors.Gray)
	proof_label.TextSize = 14
	proof_label.TextStyle = fyne.TextStyle{Bold: true}

	hash := widget.NewEntry()
	hash.PlaceHolder = "N/A"
	hash.Disable()

	hash_label := canvas.NewText("   BLOCK HASH", colors.Gray)
	hash_label.TextSize = 14
	hash_label.TextStyle = fyne.TextStyle{Bold: true}

	block := canvas.NewText("", colors.Account)
	block.TextSize = 22
	block.TextStyle = fyne.TextStyle{Bold: true}

	block_label := canvas.NewText("BLOCK", colors.Gray)
	block_label.TextSize = 14
	block_label.TextStyle = fyne.TextStyle{Bold: false}

	fees := canvas.NewText("", colors.Account)
	fees.TextSize = 22
	fees.TextStyle = fyne.TextStyle{Bold: true}

	fees_label := canvas.NewText("FEES", colors.Gray)
	fees_label.TextSize = 14
	fees_label.TextStyle = fyne.TextStyle{Bold: false}

	stamp := canvas.NewText("", colors.Account)
	stamp.TextSize = 22
	stamp.TextStyle = fyne.TextStyle{Bold: true}

	stamp_label := canvas.NewText("", colors.Gray)
	stamp_label.TextSize = 14
	stamp_label.TextStyle = fyne.TextStyle{Bold: false}

	btnView := widget.NewButton("  View in Explorer  ", nil)

	details := container.NewMax(
		container.NewBorder(
			container.NewHBox(
				container.NewVBox(
					rectSpacer,
					details_header,
				),
				layout.NewSpacer(),
				container.NewVBox(
					rectSpacer,
					container.NewHBox(
						container.NewMax(
							btnView,
						),
						widget.NewLabel(""),
					),
				),
			),
			btnClose,
			container.NewMax(
				rectList,
				container.NewVBox(
					widget.NewLabel(""),
					container.NewHBox(
						layout.NewSpacer(),
						rectSpacer,
						container.NewMax(
							rect1,
							container.NewCenter(
								container.NewVBox(
									block,
								),
							),
						),
						rectSpacer,
						container.NewMax(
							rect1,
							container.NewCenter(fees),
						),
						rectSpacer,
						container.NewMax(
							rect1,
							container.NewCenter(stamp),
						),
						layout.NewSpacer(),
					),
					container.NewHBox(
						layout.NewSpacer(),
						rectSpacer,
						container.NewMax(
							rect2,
							container.NewCenter(
								block_label,
							),
						),
						rectSpacer,
						container.NewMax(
							rect2,
							container.NewCenter(
								fees_label,
							),
						),
						rectSpacer,
						container.NewMax(
							rect2,
							container.NewCenter(
								stamp_label,
							),
						),
						layout.NewSpacer(),
					),
					widget.NewLabel(""),
					container.NewVBox(
						amount,
						amount_label,
					),
					widget.NewLabel(""),
					container.NewVBox(
						txidText_label,
						txidText,
					),
					widget.NewLabel(""),
					container.NewVBox(
						proof_label,
						proof,
					),
					widget.NewLabel(""),
					container.NewVBox(
						hash_label,
						hash,
					),
				),
			),
			nil,
		),
	)
	details.Hide()

	listing := container.NewMax(
		container.NewBorder(
			container.NewVBox(
				rectSpacer,
				header,
				rectSpacer,
				menu,
				rectSpacer,
			),
			nil,
			container.NewMax(
				rectList,
				listBox,
			),
			nil,
		),
	)
	listing.Show()

	listBox.OnSelected = func(id widget.ListItemID) {
		split := strings.Split(data[id], ";")
		result := engram.Disk.Get_Payments_TXID(split[4])

		listing.Hide()

		if result.TXID == "" {
			label.Text = "   ---"
		} else {
			label.Text = "   " + result.TXID
		}
		label.Refresh()

		btnView.OnTapped = func() {
			if engram.Disk.GetNetwork() {
				link, _ := url.Parse("https://explorer.dero.io/tx/" + result.TXID)
				_ = fyne.CurrentApp().OpenURL(link)
			} else {
				link, _ := url.Parse("https://testnetexplorer.dero.io/tx/" + result.TXID)
				_ = fyne.CurrentApp().OpenURL(link)
			}
		}

		sub.Text = fmt.Sprintf("        POSTIION IN BLOCK:  %d", result.TransactionPos)
		sub.Refresh()

		if result.Incoming {
			amount.Text = "    " + globals.FormatMoney(result.Amount)
			amount_label.Text = "          RECEIVED"
		} else {
			amount.Text = "    (" + globals.FormatMoney(result.Amount) + ")"
			amount_label.Text = "          SENT"
		}
		amount.Refresh()
		amount_label.Refresh()

		txidText.Text = result.TXID
		txidText.Refresh()

		proof.Text = result.Proof
		proof.Refresh()

		block.Text = fmt.Sprintf("%d", result.Height)
		block.Refresh()

		fees.Text = globals.FormatMoney(result.Fees)
		fees.Refresh()

		stamp.Text = result.Time.Local().Format("Jan 02, 2006")
		stamp.Refresh()

		stamp_label.Text = result.Time.Local().Format(time.Kitchen)
		stamp_label.Refresh()

		hash.Text = result.BlockHash
		hash.Refresh()

		details.Show()
	}

	btnClose.OnTapped = func() {
		if details.Hidden {
			history.Window.Close()
			history.Window = nil
		} else {
			details.Hide()
			listing.Show()
		}
	}

	layout := container.NewMax(
		listing,
		details,
	)

	return layout
}

func layoutAssets() fyne.CanvasObject {
	layout := container.NewMax()

	return layout
}

func layoutAccount() fyne.CanvasObject {
	resetResources()

	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(fyne.NewSize(200, 233))
	rectScroll := canvas.NewRectangle(color.Transparent)
	rectScroll.SetMinSize(fyne.NewSize(300, 450))
	frame := canvas.NewRectangle(color.Transparent)
	frame.SetMinSize(fyne.NewSize(MIN_WIDTH, MIN_HEIGHT))
	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(10, 5))

	title := canvas.NewText("M Y   A C C O U N T", colors.Gray)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 16

	heading := canvas.NewText("..."+engram.Disk.GetAddress().String()[len(engram.Disk.GetAddress().String())-10:len(engram.Disk.GetAddress().String())], colors.Green)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	sub := canvas.NewText("ADDRESS", colors.Gray)
	sub.TextSize = 14
	sub.Alignment = fyne.TextAlignCenter
	sub.TextStyle = fyne.TextStyle{Bold: true}

	catChange := canvas.NewText("CHANGE PASSWORD", colors.Gray)
	catChange.TextStyle = fyne.TextStyle{Bold: true}
	catChange.TextSize = 14
	catChange.Alignment = fyne.TextAlignCenter

	btnCancel := widget.NewButton("Back", nil)
	btnCopyAddress := widget.NewButton("Copy Address", nil)
	btnCopySeed := widget.NewButton("Copy Recovery Words", nil)

	address := widget.NewEntry()
	address.OnChanged = func(s string) {
		setDaemon(s)
	}
	address.PlaceHolder = "Daemon Address"
	address.SetText(getDaemon())
	address.Refresh()

	btnCancel.OnTapped = func() {
		session.Domain = "app.wallet"
		resizeWindow(MIN_WIDTH, MIN_HEIGHT)
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutDashboard())
	}

	btnCopyAddress.OnTapped = func() {
		session.Window.Clipboard().SetContent(engram.Disk.GetAddress().String())
		session.Window.Clipboard().SetContent(engram.Disk.GetAddress().String())
	}

	btnCopySeed.OnTapped = func() {
		session.Window.Clipboard().SetContent(engram.Disk.GetSeed())
		session.Window.Clipboard().SetContent(engram.Disk.GetSeed())
	}

	errorText := canvas.NewText("", colors.Red)
	errorText.Alignment = fyne.TextAlignCenter
	errorText.TextSize = 12

	curPass := widget.NewEntry()
	curPass.Password = true
	curPass.PlaceHolder = "Current Password"

	newPass := widget.NewEntry()
	newPass.Password = true
	newPass.PlaceHolder = "New Password"

	confirm := widget.NewEntry()
	confirm.Password = true
	confirm.PlaceHolder = "Confirm Password"

	btnChange := widget.NewButton("Submit", nil)
	btnChange.OnTapped = func() {
		errorText.Text = ""
		errorText.Color = colors.Red
		errorText.Refresh()
		if engram.Disk.Check_Password(curPass.Text) {
			if newPass.Text == confirm.Text && newPass.Text != "" {
				err := engram.Disk.Set_Encrypted_Wallet_Password(newPass.Text)
				if err != nil {
					errorText.Text = "Error changing password."
					errorText.Refresh()
				} else {
					curPass.Text = ""
					curPass.Refresh()
					newPass.Text = ""
					newPass.Refresh()
					confirm.Text = ""
					confirm.Refresh()
					errorText.Text = "Password updated successfully."
					errorText.Color = colors.Green
					errorText.Refresh()
				}
			} else {
				errorText.Text = "New passwords do not match."
				errorText.Refresh()
			}
		} else {
			errorText.Text = "Incorrect password entered."
			errorText.Refresh()
		}
	}

	res.gram.SetMinSize(fyne.NewSize(300, 185))
	res.gram_footer.SetMinSize(fyne.NewSize(300, 80))
	res.rpc_header.SetMinSize(fyne.NewSize(300, 80))
	res.rpc_footer.SetMinSize(fyne.NewSize(300, 20))
	res.home_footer.SetMinSize(fyne.NewSize(300, 20))
	res.nft_footer.SetMinSize(fyne.NewSize(300, 20))

	formSettings := container.NewVBox(
		btnCopyAddress,
		btnCopySeed,
		widget.NewLabel(""),
		catChange,
		rectSpacer,
		rectSpacer,
		curPass,
		widget.NewSeparator(),
		newPass,
		confirm,
		rectSpacer,
		errorText,
		rectSpacer,
		btnChange,
	)

	scrollBox := container.NewVScroll(
		container.NewHBox(
			layout.NewSpacer(),
			container.NewMax(
				rectScroll,
				formSettings,
			),
			layout.NewSpacer(),
		),
	)

	scrollBox.SetMinSize(fyne.NewSize(300, 420))

	gridItem1 := container.NewCenter(
		container.NewVBox(
			widget.NewLabel(""),
			container.NewCenter(
				res.rpc_header,
				container.NewVBox(
					title,
					rectSpacer,
				),
			),
			rectSpacer,
			heading,
			rectSpacer,
			sub,
			widget.NewLabel(""),
			scrollBox,
			rectSpacer,
			rectSpacer,
			btnCancel,
			rectSpacer,
			rectSpacer,
			res.rpc_footer,
		),
	)

	features := container.NewCenter(
		layout.NewSpacer(),
		gridItem1,
		layout.NewSpacer(),
	)

	subContainer := container.NewMax()

	c := container.NewBorder(
		features,
		subContainer,
		nil,
		nil,
	)

	layout := container.NewMax(
		frame,
		c,
	)

	return layout
}
