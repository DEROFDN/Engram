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
	"crypto/sha1"
	"errors"
	"fmt"
	"image/color"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	x "fyne.io/x/fyne/widget"
	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/dvm"
	"github.com/deroproject/derohe/globals"
	"github.com/deroproject/derohe/rpc"
	"github.com/deroproject/derohe/walletapi"
	"github.com/deroproject/derohe/walletapi/mnemonics"
	"github.com/deroproject/graviton"
)

func layoutMain() fyne.CanvasObject {
	// Set theme
	a.Settings().SetTheme(themes.main)
	// Reset UI resources
	resetResources()
	initSettings()
	session.Domain = "app.main"
	session.Path = ""
	session.Password = ""

	// Define objects

	btnLogin := widget.NewButton("Sign In", nil)

	if session.Error != "" {
		btnLogin.Text = session.Error
		btnLogin.Disable()
		btnLogin.Refresh()
		session.Error = ""
	}

	btnLogin.OnTapped = func() {
		if session.Path == "" {
			btnLogin.Text = "No account selected..."
			btnLogin.Disable()
			btnLogin.Refresh()
		} else if session.Password == "" {
			btnLogin.Text = "Invalid password..."
			btnLogin.Disable()
			btnLogin.Refresh()
		} else {
			btnLogin.Text = "Sign In"
			btnLogin.Enable()
			btnLogin.Refresh()
			login()
			btnLogin.Text = session.Error
			btnLogin.Disable()
			btnLogin.Refresh()
			session.Error = ""
		}
	}

	btnLogin.Disable()

	session.Window.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
		if session.Domain == "app.main" || session.Domain == "app.register" {
			if k.Name == fyne.KeyReturn {
				if session.Path == "" {
					btnLogin.Text = "No account selected..."
					btnLogin.Disable()
					btnLogin.Refresh()
				} else if session.Password == "" {
					btnLogin.Text = "Invalid password..."
					btnLogin.Disable()
					btnLogin.Refresh()
				} else {
					btnLogin.Text = "Sign In"
					btnLogin.Enable()
					btnLogin.Refresh()
					login()
					btnLogin.Text = "Invalid password..."
					btnLogin.Disable()
					btnLogin.Refresh()
					session.Error = ""
				}
			}
		} else {
			return
		}
	})

	menuLabel := canvas.NewText("  M O R E   O P T I O N S  ", colors.Gray)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkCreate := widget.NewHyperlinkWithStyle("Create a new account", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkCreate.OnTapped = func() {
		session.Domain = "app.create"
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutNewAccount())
		removeOverlays()
	}

	linkRecover := widget.NewHyperlinkWithStyle("Recover an existing account", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkRecover.OnTapped = func() {
		session.Domain = "app.restore"
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutRestore())
		removeOverlays()
	}

	linkSettings := widget.NewHyperlinkWithStyle("Settings", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkSettings.OnTapped = func() {
		session.Domain = "app.settings"
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutSettings())
		removeOverlays()
	}

	modeData := binding.BindBool(&session.Offline)
	mode := widget.NewCheckWithData(" Offline Mode", modeData)
	mode.OnChanged = func(b bool) {
		if b {
			session.Offline = true
		} else {
			session.Offline = false
		}
	}

	footer := canvas.NewText("© 2023  DERO FOUNDATION  |  VERSION  "+version.String(), colors.Gray)
	footer.TextSize = 10
	footer.Alignment = fyne.TextAlignCenter
	footer.TextStyle = fyne.TextStyle{Bold: true}

	wPassword := NewReturnEntry()

	wPassword.Password = true
	wPassword.OnChanged = func(s string) {
		session.Error = ""
		btnLogin.Text = "Sign In"
		btnLogin.Enable()
		btnLogin.Refresh()
		session.Password = s

		if len(s) < 1 {
			btnLogin.Disable()
			btnLogin.Refresh()
		} else if session.Path == "" {
			btnLogin.Disable()
			btnLogin.Refresh()
		} else {
			btnLogin.Enable()
		}

		btnLogin.Refresh()
	}
	wPassword.SetPlaceHolder("Password")

	// Get account databases in app directory
	list, err := GetAccounts()
	if err != nil {
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutAlert(2))
	}

	// Populate the accounts in dropdown menu
	wAccount := widget.NewSelect(list, nil)
	wAccount.PlaceHolder = "(Select Account)"
	wAccount.OnChanged = func(s string) {
		session.Error = ""
		btnLogin.Text = "Sign In"
		btnLogin.Refresh()

		// OnChange set wallet path
		if session.Testnet {
			session.Path = filepath.Join(AppPath(), "testnet") + string(filepath.Separator) + s
		} else {
			session.Path = filepath.Join(AppPath(), "mainnet") + string(filepath.Separator) + s
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

	rectStatus := canvas.NewRectangle(color.Transparent)
	rectStatus.SetMinSize(fyne.NewSize(10, 10))

	res.bg2.SetMinSize(fyne.NewSize(ui.Width, ui.MaxHeight*0.2))
	res.bg2.Refresh()

	headerBox := canvas.NewRectangle(color.Transparent)
	headerBox.SetMinSize(fyne.NewSize(ui.Width, 1))

	frame := &iframe{}

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(ui.Width, 5))

	status.Connection.FillColor = colors.Gray
	status.Cyberdeck.FillColor = colors.Gray
	status.Sync.FillColor = colors.Gray

	form := container.NewVBox(
		wSpacer,
		container.NewStack(
			res.bg2,
		),
		rectSpacer,
		rectSpacer,
		wAccount,
		rectSpacer,
		wPassword,
		rectSpacer,
		mode,
		rectSpacer,
		rectSpacer,
		btnLogin,
		wSpacer,
		container.NewStack(
			container.NewHBox(
				line1,
				layout.NewSpacer(),
				menuLabel,
				layout.NewSpacer(),
				line2,
			),
		),
		rectSpacer,
		rectSpacer,
		container.NewStack(
			container.NewHBox(
				layout.NewSpacer(),
				linkCreate,
				layout.NewSpacer(),
			),
		),
		rectSpacer,
		container.NewStack(
			container.NewHBox(
				layout.NewSpacer(),
				linkRecover,
				layout.NewSpacer(),
			),
		),
		rectSpacer,
		container.NewStack(
			container.NewHBox(
				layout.NewSpacer(),
				linkSettings,
				layout.NewSpacer(),
			),
		),
	)

	layout := container.NewStack(
		frame,
		container.NewBorder(
			container.NewVBox(
				container.NewCenter(
					form,
				),
			),
			container.NewVBox(
				footer,
				wSpacer,
			),
			nil,
			nil,
		),
	)

	return layout
}

func layoutDashboard() fyne.CanvasObject {
	resizeWindow(ui.MaxWidth, ui.MaxHeight)
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

	if session.BalanceUSD == "" {
		session.BalanceUSDText = canvas.NewText("", colors.Gray)
		session.BalanceUSDText.TextSize = 14
		session.BalanceUSDText.TextStyle = fyne.TextStyle{Bold: true}
	} else {
		session.BalanceUSDText = canvas.NewText("USD  "+session.BalanceUSD, colors.Gray)
		session.BalanceUSDText.TextSize = 14
		session.BalanceUSDText.TextStyle = fyne.TextStyle{Bold: true}
	}

	network := ""
	if session.Testnet {
		network = " T  E  S  T  N  E  T "
	} else {
		network = " M  A  I  N  N  E  T "
	}

	rectStatus := canvas.NewRectangle(color.Transparent)
	rectStatus.SetMinSize(fyne.NewSize(10, 10))
	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(fyne.NewSize(ui.Width, 20))
	frame := &iframe{}

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

	path := strings.Split(session.Path, string(filepath.Separator))
	accountName := canvas.NewText(path[len(path)-1], colors.Green)
	accountName.TextStyle = fyne.TextStyle{Bold: true}
	accountName.TextSize = 18

	shortAddress := canvas.NewText("····"+short, colors.Gray)
	shortAddress.TextStyle = fyne.TextStyle{Bold: true}
	shortAddress.TextSize = 22

	gramSend := widget.NewButton(" Send ", nil)

	heading := canvas.NewText("B A L A N C E", colors.Gray)
	heading.TextSize = 16
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

	headerLabel := canvas.NewText("  "+network+"  ", colors.Gray)
	headerLabel.TextSize = 11
	headerLabel.Alignment = fyne.TextAlignCenter
	headerLabel.TextStyle = fyne.TextStyle{Bold: true}

	statusLabel := canvas.NewText("  S T A T U S  ", colors.Gray)
	statusLabel.TextSize = 11
	statusLabel.Alignment = fyne.TextAlignCenter
	statusLabel.TextStyle = fyne.TextStyle{Bold: true}

	daemonLabel := canvas.NewText("OFFLINE", colors.Gray)
	daemonLabel.TextSize = 12
	daemonLabel.Alignment = fyne.TextAlignCenter
	daemonLabel.TextStyle = fyne.TextStyle{Bold: false}

	if !session.Offline {
		daemonLabel.Text = session.Daemon
	}

	session.WalletHeight = engram.Disk.Get_Height()
	session.StatusText = canvas.NewText(fmt.Sprintf("%d", session.WalletHeight), colors.Gray)
	session.StatusText.TextSize = 12
	session.StatusText.Alignment = fyne.TextAlignCenter
	session.StatusText.TextStyle = fyne.TextStyle{Bold: false}

	menuLabel := canvas.NewText("  M O D U L E S  ", colors.Gray)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkLogout := widget.NewHyperlinkWithStyle("Sign Out", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkLogout.OnTapped = func() {
		closeWallet()
	}

	linkHistory := widget.NewHyperlinkWithStyle("View History", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkHistory.OnTapped = func() {
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutHistory())
		removeOverlays()
	}

	menu := widget.NewSelect([]string{"Identity", "My Account", "Messages", "Transfers", "Asset Explorer", "Services", "Cyberdeck", "Datapad", " "}, nil)
	menu.PlaceHolder = "Select Module ..."
	menu.OnChanged = func(s string) {
		if s == "My Account" {
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutAccount())
			removeOverlays()
		} else if s == "Transfers" {
			session.Window.Canvas().SetContent(layoutTransition())
			session.Window.Canvas().SetContent(layoutTransfers())
			removeOverlays()
		} else if s == "Asset Explorer" {
			session.Window.Canvas().SetContent(layoutTransition())
			session.Window.Canvas().SetContent(layoutAssetExplorer())
			removeOverlays()
		} else if s == "Datapad" {
			session.Window.Canvas().SetContent(layoutTransition())
			session.Window.Canvas().SetContent(layoutDatapad())
			removeOverlays()
		} else if s == "Messages" {
			session.Window.Canvas().SetContent(layoutTransition())
			session.Window.Canvas().SetContent(layoutMessages())
			removeOverlays()
		} else if s == "Cyberdeck" {
			session.Window.Canvas().SetContent(layoutTransition())
			session.Window.Canvas().SetContent(layoutCyberdeck())
			removeOverlays()
		} else if s == "Identity" {
			session.Window.Canvas().SetContent(layoutTransition())
			session.Window.Canvas().SetContent(layoutIdentity())
			removeOverlays()
		} else if s == "Services" {
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutServiceAddress())
			removeOverlays()
		} else {
			session.Window.Canvas().SetContent(layoutTransition())
			session.Window.Canvas().SetContent(layoutDashboard())
			removeOverlays()
		}
	}

	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(fyne.NewSize(ui.Width, 260))

	rect300 := canvas.NewRectangle(color.Transparent)
	rect300.SetMinSize(fyne.NewSize(ui.Width, 50))

	rect.SetMinSize(fyne.NewSize(ui.Width, 30))

	res.gram.SetMinSize(fyne.NewSize(ui.Width, 150))

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(10, 5))

	deroForm := container.NewVBox(
		rectSpacer,
		res.gram,
		rectSpacer,
		container.NewStack(
			container.NewHBox(
				line1,
				layout.NewSpacer(),
				headerLabel,
				layout.NewSpacer(),
				line2,
			),
		),
		rectSpacer,
		rectSpacer,
		heading,
		rectSpacer,
		balanceCenter,
		rectSpacer,
		rectSpacer,
		gramSend,
		rectSpacer,
		container.NewHBox(
			layout.NewSpacer(),
			linkHistory,
			layout.NewSpacer(),
		),
		rectSpacer,
		rectSpacer,
		container.NewHBox(
			line1,
			layout.NewSpacer(),
			menuLabel,
			layout.NewSpacer(),
			line2,
		),
		rectSpacer,
		rectSpacer,
		menu,
		rectSpacer,
		rectSpacer,
		container.NewHBox(
			line1,
			layout.NewSpacer(),
			statusLabel,
			layout.NewSpacer(),
			line2,
		),
		rectSpacer,
		rectSpacer,
		container.NewHBox(
			container.NewStack(
				rectStatus,
				status.Connection,
			),
			rectSpacer,
			daemonLabel,
			layout.NewSpacer(),
			container.NewStack(
				rectStatus,
				status.Sync,
			),
			rectSpacer,
			session.StatusText,
		),
	)

	grid := container.NewCenter(
		deroForm,
	)

	gramSend.OnTapped = func() {
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutSend())
		removeOverlays()
	}

	session.Window.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
		if session.Domain != "app.wallet" {
			return
		}

		if k.Name == fyne.KeyRight {
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutCyberdeck())
			removeOverlays()
		} else if k.Name == fyne.KeyLeft {
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutIdentity())
			removeOverlays()
		} else if k.Name == fyne.KeyUp {
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutTransfers())
			removeOverlays()
		} else if k.Name == fyne.KeyDown {
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutMessages())
			removeOverlays()
		}

	})

	top := container.NewCenter(
		layout.NewSpacer(),
		grid,
		layout.NewSpacer(),
	)

	bottom := container.NewStack(
		container.NewVBox(
			container.NewHBox(
				layout.NewSpacer(),
				container.NewHBox(
					layout.NewSpacer(),
					linkLogout,
					layout.NewSpacer(),
				),
				layout.NewSpacer(),
			),
			wSpacer,
		),
	)

	c := container.NewBorder(
		top,
		bottom,
		nil,
		nil,
	)

	layout := container.NewStack(
		frame,
		c,
	)

	return layout
}

func layoutSend() fyne.CanvasObject {
	// Reset UI resources
	resetResources()
	session.Domain = "app.send"

	wSpacer := widget.NewLabel(" ")
	frame := &iframe{}

	btnSend := widget.NewButton("Save", nil)

	wAmount := widget.NewEntry()
	wMessage := widget.NewEntry()
	wPaymentID := widget.NewEntry()

	options := []string{"Anonymity Set:   2  (None)", "Anonymity Set:   4  (Low)", "Anonymity Set:   8  (Low)", "Anonymity Set:   16  (Recommended)", "Anonymity Set:   32  (Medium)", "Anonymity Set:   64  (High)", "Anonymity Set:   128  (High)"}
	wRings := widget.NewSelect(options, nil)

	wReceiver := widget.NewEntry()
	wReceiver.Validator = func(s string) error {
		address, err := globals.ParseValidateAddress(s)
		if err != nil {
			tx.Address = nil
			_, addr, _ := checkUsername(s, -1)
			if addr == "" {
				btnSend.Disable()
				err = errors.New("invalid username or address")
				wReceiver.SetValidationError(err)
				tx.Address = nil
				return err
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
		} else {
			if address.IsIntegratedAddress() {
				tx.Address = address

				if address.Arguments.HasValue(rpc.RPC_VALUE_TRANSFER, rpc.DataUint64) {
					amount := address.Arguments[address.Arguments.Index(rpc.RPC_VALUE_TRANSFER, rpc.DataUint64)].Value
					tx.Amount = amount.(uint64)
					wAmount.Text = globals.FormatMoney(amount.(uint64))
					if amount.(uint64) != 0.00000 {
						wAmount.Disable()
					}
					wAmount.Refresh()
				}

				if address.Arguments.HasValue(rpc.RPC_DESTINATION_PORT, rpc.DataUint64) {
					port := address.Arguments[address.Arguments.Index(rpc.RPC_DESTINATION_PORT, rpc.DataUint64)].Value
					tx.PaymentID = port.(uint64)
					wPaymentID.Text = strconv.FormatUint(port.(uint64), 10)
					wPaymentID.Disable()
					wPaymentID.Refresh()
				}

				if address.Arguments.HasValue(rpc.RPC_COMMENT, rpc.DataString) {
					comment := address.Arguments[address.Arguments.Index(rpc.RPC_COMMENT, rpc.DataString)].Value
					tx.Comment = comment.(string)
					wMessage.Text = comment.(string)
					if comment.(string) != "" {
						wMessage.Disable()
					}
					wMessage.Refresh()
				}

				if tx.Ringsize == 0 {
					wRings.SetSelected("Anonymity Set:   16  (Recommended)")
				} else {

				}

				if tx.Amount != 0 {
					balance, _ := engram.Disk.Get_Balance()
					if tx.Amount <= balance {
						btnSend.Enable()
					}
				}
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
		}
		return nil
	}

	wReceiver.SetPlaceHolder("Receiver username or address")
	wReceiver.SetValidationError(nil)

	wAmount.SetPlaceHolder("Amount")

	wMessage.SetPlaceHolder("Message")
	wMessage.OnChanged = func(s string) {
		bytes := []byte(s)
		if len(bytes) <= 130 {
			tx.Comment = s
		}
		wMessage.SetText(tx.Comment)
	}

	/*
		wAll := widget.NewCheck(" All", func(b bool) {
			if b {
				tx.Amount = engram.Disk.GetAccount().Balance_Mature
				wAmount.SetText(walletapi.FormatMoney(tx.Amount))
			} else {
				tx.Amount = 0
				wAmount.SetText("")
			}
		})
	*/

	wAmount.Validator = func(s string) error {
		if s == "" {
			tx.Amount = 0
			wAmount.SetValidationError(errors.New("Invalid transaction amount"))
			btnSend.Disable()
		} else {
			balance, _ := engram.Disk.Get_Balance()
			entry, err := globals.ParseAmount(s)
			if err != nil {
				tx.Amount = 0
				wAmount.SetValidationError(errors.New("Invalid transaction amount"))
				btnSend.Disable()
				return errors.New("Invalid transaction amount")
			}
			if entry <= balance {
				tx.Amount = entry
				wAmount.SetValidationError(nil)
				if wReceiver.Validate() == nil {
					btnSend.Enable()
				}
			} else {
				tx.Amount = 0
				btnSend.Disable()
				wAmount.SetValidationError(errors.New("Insufficient funds"))
			}
			return nil
		}
		return errors.New("Invalid transaction amount")
	}

	wAmount.SetValidationError(nil)

	var err error

	wPaymentID.OnChanged = func(s string) {
		tx.PaymentID, err = strconv.ParseUint(s, 10, 64)
		if err != nil {
			tx.PaymentID = 0
		}
	}
	wPaymentID.SetPlaceHolder("Payment ID / Service Port")

	wRings.PlaceHolder = "(Select Anonymity Set)"
	wRings.OnChanged = func(s string) {
		regex := regexp.MustCompile("[0-9]+")
		result := regex.FindAllString(s, -1)
		tx.Ringsize, err = strconv.ParseUint(result[0], 10, 64)
		if err != nil {
			tx.Ringsize = 16
			wRings.SetSelected(options[3])
		}
		session.Window.Canvas().Focus(wReceiver)
	}

	btnSend.OnTapped = func() {
		_, err := globals.ParseAmount(wAmount.Text)
		if tx.Address != nil {
			if wRings != nil && err == nil && tx.Address != nil {
				err = addTransfer()
				if err == nil {
					session.Window.SetContent(layoutTransition())
					session.Window.SetContent(layoutTransfers())
					removeOverlays()
				}
			} else {
				wReceiver.SetValidationError(errors.New("Invalid Address"))
				wReceiver.Refresh()
			}
		}
	}

	sendHeading := canvas.NewText("Send Money", colors.Green)
	sendHeading.TextSize = 22
	sendHeading.Alignment = fyne.TextAlignCenter
	sendHeading.TextStyle = fyne.TextStyle{Bold: true}

	optionalLabel := canvas.NewText("  O P T I O N A L  ", colors.Gray)
	optionalLabel.TextSize = 11
	optionalLabel.Alignment = fyne.TextAlignCenter
	optionalLabel.TextStyle = fyne.TextStyle{Bold: true}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkCancel := widget.NewHyperlinkWithStyle("Cancel", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(fyne.NewSize(ui.Width, 260))

	rect300 := canvas.NewRectangle(color.Transparent)
	rect300.SetMinSize(fyne.NewSize(ui.Width, 30))

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(10, 5))

	form := container.NewVBox(
		wSpacer,
		container.NewCenter(
			rect300,
			sendHeading,
		),
		wSpacer,
		wRings,
		rectSpacer,
		wReceiver,
		wAmount,
		rectSpacer,
		rectSpacer,
		container.NewHBox(
			line1,
			layout.NewSpacer(),
			optionalLabel,
			layout.NewSpacer(),
			line2,
		),
		rectSpacer,
		rectSpacer,
		wPaymentID,
		wMessage,
		wSpacer,
	)

	grid := container.NewCenter(
		form,
	)

	linkCancel.OnTapped = func() {
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutDashboard())
		removeOverlays()
	}

	top := container.NewCenter(
		layout.NewSpacer(),
		grid,
		layout.NewSpacer(),
	)

	bottom := container.NewStack(
		container.NewVBox(
			rectSpacer,
			container.NewHBox(
				layout.NewSpacer(),
				container.NewStack(
					rect300,
					btnSend,
				),
				layout.NewSpacer(),
			),
			rectSpacer,
			container.NewHBox(
				layout.NewSpacer(),
				container.NewHBox(
					layout.NewSpacer(),
					linkCancel,
					layout.NewSpacer(),
				),
				layout.NewSpacer(),
			),
			wSpacer,
		),
	)

	c := container.NewBorder(
		top,
		bottom,
		nil,
		nil,
	)

	layout := container.NewStack(
		frame,
		c,
	)

	return layout
}

func layoutServiceAddress() fyne.CanvasObject {
	resetResources()
	session.Domain = "app.service"

	wSpacer := widget.NewLabel(" ")
	frame := &iframe{}

	btnCreate := widget.NewButton("Create", nil)

	wPaymentID := widget.NewEntry()

	wReceiver := widget.NewEntry()
	wReceiver.Text = engram.Disk.GetAddress().String()
	wReceiver.Disable()

	tx.Address, _ = globals.ParseValidateAddress(engram.Disk.GetAddress().String())

	wReceiver.SetPlaceHolder("Receiver username or address")
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

	wAmount.Validator = func(s string) error {
		if s == "" {
			tx.Amount = 0
			wAmount.SetValidationError(errors.New("Invalid transaction amount"))
			btnCreate.Disable()
		} else {
			amount, err := globals.ParseAmount(s)
			if err != nil {
				tx.Amount = 0
				wAmount.SetValidationError(errors.New("Invalid transaction amount"))
				btnCreate.Disable()
				return errors.New("Invalid transaction amount")
			}
			wAmount.SetValidationError(nil)
			tx.Amount = amount
			btnCreate.Enable()

			return nil
		}
		return errors.New("Invalid transaction amount")
	}

	wAmount.SetValidationError(nil)

	var err error

	wPaymentID.OnChanged = func(s string) {
		tx.PaymentID, err = strconv.ParseUint(s, 10, 64)
		if err != nil {
			tx.PaymentID = 0
			btnCreate.Disable()
		} else {
			if wReceiver.Text != "" {
				btnCreate.Enable()
			}
		}
	}
	wPaymentID.SetPlaceHolder("Payment ID / Service Port")

	sendHeading := canvas.NewText("New Service Address", colors.Green)
	sendHeading.TextSize = 22
	sendHeading.Alignment = fyne.TextAlignCenter
	sendHeading.TextStyle = fyne.TextStyle{Bold: true}

	optionalLabel := canvas.NewText("  O P T I O N A L  ", colors.Gray)
	optionalLabel.TextSize = 11
	optionalLabel.Alignment = fyne.TextAlignCenter
	optionalLabel.TextStyle = fyne.TextStyle{Bold: true}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkCancel := widget.NewHyperlinkWithStyle("Cancel", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(fyne.NewSize(ui.Width, 260))

	rect300 := canvas.NewRectangle(color.Transparent)
	rect300.SetMinSize(fyne.NewSize(ui.Width, 30))

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(10, 5))

	btnCreate.OnTapped = func() {
		var err error
		if tx.Address != nil && tx.PaymentID != 0 {
			if wAmount.Text != "" {
				_, err = globals.ParseAmount(wAmount.Text)
			}

			if err == nil {
				header := canvas.NewText("CREATE  SERVICE  ADDRESS", colors.Gray)
				header.TextSize = 14
				header.Alignment = fyne.TextAlignCenter
				header.TextStyle = fyne.TextStyle{Bold: true}

				subHeader := canvas.NewText("Successfully Created", colors.Account)
				subHeader.TextSize = 22
				subHeader.Alignment = fyne.TextAlignCenter
				subHeader.TextStyle = fyne.TextStyle{Bold: true}

				labelAddress := canvas.NewText("-------------    INTEGRATED  ADDRESS    -------------", colors.Gray)
				labelAddress.TextSize = 12
				labelAddress.Alignment = fyne.TextAlignCenter
				labelAddress.TextStyle = fyne.TextStyle{Bold: true}

				btnCopy := widget.NewButton("Copy Service Address", nil)

				valueAddress := widget.NewRichTextFromMarkdown("")
				valueAddress.Wrapping = fyne.TextWrapBreak

				address := engram.Disk.GetRandomIAddress8()
				address.Arguments = nil
				address.Arguments = append(address.Arguments, rpc.Argument{Name: rpc.RPC_NEEDS_REPLYBACK_ADDRESS, DataType: rpc.DataUint64, Value: uint64(1)})
				address.Arguments = append(address.Arguments, rpc.Argument{Name: rpc.RPC_VALUE_TRANSFER, DataType: rpc.DataUint64, Value: tx.Amount})
				address.Arguments = append(address.Arguments, rpc.Argument{Name: rpc.RPC_DESTINATION_PORT, DataType: rpc.DataUint64, Value: tx.PaymentID})
				address.Arguments = append(address.Arguments, rpc.Argument{Name: rpc.RPC_COMMENT, DataType: rpc.DataString, Value: tx.Comment})

				err := address.Arguments.Validate_Arguments()
				if err != nil {
					fmt.Printf("[Service Address] Error: %s\n", err)
					subHeader.Text = "Error"
					subHeader.Refresh()
					btnCopy.Disable()
				} else {
					fmt.Printf("[Service Address] New Integrated Address: %s\n", address.String())
					fmt.Printf("[Service Address] Arguments: %s\n", address.Arguments)

					valueAddress.ParseMarkdown("" + address.String())
					valueAddress.Refresh()
				}

				btnCopy.OnTapped = func() {
					session.Window.Clipboard().SetContent(address.String())
				}

				linkClose := widget.NewHyperlinkWithStyle("Go Back", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
				linkClose.OnTapped = func() {
					overlay := session.Window.Canvas().Overlays()
					overlay.Top().Hide()
					overlay.Remove(overlay.Top())
					overlay.Remove(overlay.Top())
				}

				span := canvas.NewRectangle(color.Transparent)
				span.SetMinSize(fyne.NewSize(ui.Width, 10))

				overlay := session.Window.Canvas().Overlays()

				overlay.Add(
					container.NewStack(
						&iframe{},
						canvas.NewRectangle(colors.DarkMatter),
					),
				)

				overlay.Add(
					container.NewStack(
						&iframe{},
						container.NewCenter(
							container.NewVBox(
								span,
								container.NewCenter(
									header,
								),
								rectSpacer,
								rectSpacer,
								subHeader,
								rectSpacer,
								rectSpacer,
								rectSpacer,
								labelAddress,
								rectSpacer,
								valueAddress,
								rectSpacer,
								rectSpacer,
								btnCopy,
								rectSpacer,
								rectSpacer,
								container.NewHBox(
									layout.NewSpacer(),
									linkClose,
									layout.NewSpacer(),
								),
								rectSpacer,
								rectSpacer,
							),
						),
					),
				)
			} else {
				wReceiver.SetValidationError(errors.New("Invalid Address"))
				wReceiver.Refresh()
			}
		}
	}

	form := container.NewVBox(
		wSpacer,
		container.NewCenter(
			rect300,
			sendHeading,
		),
		wSpacer,
		wReceiver,
		wPaymentID,
		rectSpacer,
		rectSpacer,
		container.NewHBox(
			line1,
			layout.NewSpacer(),
			optionalLabel,
			layout.NewSpacer(),
			line2,
		),
		rectSpacer,
		rectSpacer,
		wAmount,
		wMessage,
		wSpacer,
	)

	grid := container.NewCenter(
		form,
	)

	linkCancel.OnTapped = func() {
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutDashboard())
		removeOverlays()
	}

	top := container.NewCenter(
		layout.NewSpacer(),
		grid,
		layout.NewSpacer(),
	)

	bottom := container.NewStack(
		container.NewVBox(
			rectSpacer,
			container.NewHBox(
				layout.NewSpacer(),
				container.NewStack(
					rect300,
					btnCreate,
				),
				layout.NewSpacer(),
			),
			rectSpacer,
			container.NewHBox(
				layout.NewSpacer(),
				container.NewHBox(
					layout.NewSpacer(),
					linkCancel,
					layout.NewSpacer(),
				),
				layout.NewSpacer(),
			),
			wSpacer,
		),
	)

	c := container.NewBorder(
		top,
		bottom,
		nil,
		nil,
	)

	layout := container.NewStack(
		frame,
		c,
	)

	return layout
}

func layoutLoading() fyne.CanvasObject {
	res.load.FillMode = canvas.ImageFillStretch
	layout := container.NewStack(
		res.load,
	)

	return layout
}

func layoutNewAccount() fyne.CanvasObject {
	resizeWindow(ui.MaxWidth, ui.MaxHeight)
	a.Settings().SetTheme(themes.alt)
	resetResources()

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

	linkCancel := widget.NewHyperlinkWithStyle("Return to Login", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkCancel.OnTapped = func() {
		session.Domain = "app.main"
		session.Error = ""
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutMain())
		removeOverlays()
	}

	btnCopySeed := widget.NewButton("Copy Recovery Words", nil)
	btnCopyAddress := widget.NewButton("Copy Address", nil)

	if !a.Driver().Device().IsMobile() {
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
	}

	wPassword := NewReturnEntry()
	wPassword.Password = true
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
	wPassword.Wrapping = fyne.TextWrap(fyne.TextTruncateClip)

	wPasswordConfirm := NewReturnEntry()
	wPasswordConfirm.Password = true
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
	wPasswordConfirm.Wrapping = fyne.TextWrap(fyne.TextTruncateClip)

	wAccount := widget.NewEntry()
	wAccount.SetPlaceHolder("Account Name")
	wAccount.Validator = func(s string) (err error) {
		session.Error = ""
		errorText.Text = ""
		errorText.Refresh()

		if len(s) > 25 {
			err = errors.New("Account name is too long.")
			wAccount.SetText(session.Name)
			wAccount.Refresh()
			return
		}

		err = checkDir()
		if err != nil {
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutAlert(2))
			return
		}

		if getTestnet() {
			session.Path = filepath.Join(AppPath(), "testnet", s+".db")
		} else {
			session.Path = filepath.Join(AppPath(), "mainnet", s+".db")
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

	rectHeader := canvas.NewRectangle(color.Transparent)
	rectHeader.SetMinSize(fyne.NewSize(ui.Width, 10))

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(ui.Width, 5))

	grid := container.NewVBox()
	grid.Objects = nil

	header := container.NewVBox(
		wSpacer,
		heading,
		rectSpacer,
		rectSpacer,
	)

	form := container.NewVBox(
		wLanguage,
		rectSpacer,
		wAccount,
		wPassword,
		wPasswordConfirm,
		rectSpacer,
		errorText,
		rectSpacer,
		btnCreate,
	)

	footer := container.NewVBox(
		container.NewHBox(
			layout.NewSpacer(),
			linkCancel,
			layout.NewSpacer(),
		),
		wSpacer,
	)

	body := widget.NewLabel("Please save the following 25 recovery words in a safe place. These are the keys to your account, so never share them with anyone.")
	body.Wrapping = fyne.TextWrapWord
	body.Alignment = fyne.TextAlignCenter
	body.TextStyle = fyne.TextStyle{Bold: true}

	formSuccess := container.NewVBox(
		body,
		wSpacer,
		container.NewCenter(grid),
		rectSpacer,
		errorText,
		rectSpacer,
		btnCopyAddress,
		btnCopySeed,
		rectSpacer,
	)

	formSuccess.Hide()

	scrollBox := container.NewVScroll(
		container.NewHBox(
			layout.NewSpacer(),
			container.NewStack(
				formSuccess,
				form,
			),
			layout.NewSpacer(),
		),
	)
	scrollBox.SetMinSize(fyne.NewSize(ui.Width, ui.Height*0.70))

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

		rect := canvas.NewRectangle(color.RGBA{21, 27, 36, 255})
		rect.SetMinSize(fyne.NewSize(ui.Width, 25))

		for i := 0; i < len(formatted); i++ {
			pos := fmt.Sprintf("%d", i+1)
			word := strings.ReplaceAll(formatted[i], " ", "")
			grid.Add(container.NewStack(
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

	layout := container.NewBorder(
		container.NewVBox(
			header,
			scrollBox,
		),
		footer,
		nil,
		nil,
	)
	return layout
}

func layoutRestore() fyne.CanvasObject {
	resizeWindow(ui.MaxWidth, ui.MaxHeight)
	a.Settings().SetTheme(themes.main)
	resetResources()

	session.Domain = "app.restore"
	session.Language = -1
	session.Error = ""
	session.Name = ""
	session.Password = ""
	session.PasswordConfirm = ""

	var seed [25]string

	scrollBox := container.NewVScroll(nil)

	errorText := canvas.NewText(" ", colors.Green)
	errorText.TextSize = 12
	errorText.Alignment = fyne.TextAlignCenter

	btnCreate := widget.NewButton("Recover", nil)
	btnCreate.Disable()

	linkReturn := widget.NewHyperlinkWithStyle("Return to Login", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkReturn.OnTapped = func() {
		session.Domain = "app.main"
		session.Error = ""
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutMain())
		removeOverlays()
	}

	btnCopyAddress := widget.NewButton("Copy Address", nil)

	wPassword := NewMobileEntry()
	wPassword.OnFocusGained = func() {
		offset := wPassword.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
		}
	}

	wPassword.Password = true
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
	wPassword.Wrapping = fyne.TextWrap(fyne.TextTruncateClip)

	wPasswordConfirm := NewMobileEntry()
	wPasswordConfirm.OnFocusGained = func() {
		offset := wPasswordConfirm.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
		}
	}

	wPasswordConfirm.Password = true
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
	wPasswordConfirm.Wrapping = fyne.TextWrap(fyne.TextTruncateClip)

	wAccount := NewMobileEntry()
	wAccount.OnFocusGained = func() {
		scrollBox.Offset = fyne.NewPos(0, 0)
		scrollBox.Refresh()
	}

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
	wAccount.Wrapping = fyne.TextWrap(fyne.TextTruncateClip)
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

		err = checkDir()
		if err != nil {
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutAlert(2))
			return
		}

		if getTestnet() {
			session.Path = filepath.Join(AppPath(), "testnet") + string(filepath.Separator) + s + ".db"
		} else {
			session.Path = filepath.Join(AppPath(), "mainnet") + string(filepath.Separator) + s + ".db"
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

	rectHeader := canvas.NewRectangle(color.Transparent)
	rectHeader.SetMinSize(fyne.NewSize(ui.Width, 10))

	//frame := &iframe{}

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(ui.Width, 5))

	status.Connection.FillColor = colors.Gray
	status.Cyberdeck.FillColor = colors.Gray
	status.Sync.FillColor = colors.Gray

	grid := container.NewVBox()
	grid.Objects = nil

	word1 := NewMobileEntry()
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
	word1.OnFocusGained = func() {
		offset := word1.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
		}
	}

	word2 := NewMobileEntry()
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
	word2.OnFocusGained = func() {
		offset := word2.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
		}
	}

	word3 := NewMobileEntry()
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
	word3.OnFocusGained = func() {
		offset := word3.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
		}
	}

	word4 := NewMobileEntry()
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
	word4.OnFocusGained = func() {
		offset := word4.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
		}
	}

	word5 := NewMobileEntry()
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
	word5.OnFocusGained = func() {
		offset := word5.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
		}
	}

	word6 := NewMobileEntry()
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
	word6.OnFocusGained = func() {
		offset := word6.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			fmt.Printf("scrollBox - before: %f\n", scrollBox.Offset.Y)
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
			fmt.Printf("scrollBox - after: %f\n", scrollBox.Offset.Y)
		}
		fmt.Printf("offset: %f\n", offset)
	}

	word7 := NewMobileEntry()
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
	word7.OnFocusGained = func() {
		offset := word7.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			fmt.Printf("scrollBox - before: %f\n", scrollBox.Offset.Y)
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
			fmt.Printf("scrollBox - after: %f\n", scrollBox.Offset.Y)
		}
		fmt.Printf("offset: %f\n", offset)
	}

	word8 := NewMobileEntry()
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
	word8.OnFocusGained = func() {
		offset := word8.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
		}
	}

	word9 := NewMobileEntry()
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
	word9.OnFocusGained = func() {
		offset := word9.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
		}
	}

	word10 := NewMobileEntry()
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
	word10.OnFocusGained = func() {
		offset := word10.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
		}
	}

	word11 := NewMobileEntry()
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
	word11.OnFocusGained = func() {
		offset := word11.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
		}
	}

	word12 := NewMobileEntry()
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
	word12.OnFocusGained = func() {
		offset := word12.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
		}
	}

	word13 := NewMobileEntry()
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
	word13.OnFocusGained = func() {
		offset := word13.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
		}
	}

	word14 := NewMobileEntry()
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
	word14.OnFocusGained = func() {
		offset := word14.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
		}
	}

	word15 := NewMobileEntry()
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
	word15.OnFocusGained = func() {
		offset := word15.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
		}
	}

	word16 := NewMobileEntry()
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
	word16.OnFocusGained = func() {
		offset := word16.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
		}
	}

	word17 := NewMobileEntry()
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
	word17.OnFocusGained = func() {
		offset := word17.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
		}
	}

	word18 := NewMobileEntry()
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
	word18.OnFocusGained = func() {
		offset := word18.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
		}
	}

	word19 := NewMobileEntry()
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
	word19.OnFocusGained = func() {
		offset := word19.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
		}
	}

	word20 := NewMobileEntry()
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
	word20.OnFocusGained = func() {
		offset := word20.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
		}
	}

	word21 := NewMobileEntry()
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
	word21.OnFocusGained = func() {
		offset := word21.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
		}
	}

	word22 := NewMobileEntry()
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
	word22.OnFocusGained = func() {
		offset := word22.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
		}
	}

	word23 := NewMobileEntry()
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
	word23.OnFocusGained = func() {
		offset := word23.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
		}
	}

	word24 := NewMobileEntry()
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
	word24.OnFocusGained = func() {
		offset := word24.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
		}
	}

	word25 := NewMobileEntry()
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
	word25.OnFocusGained = func() {
		offset := word25.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			fmt.Printf("scrollBox - before: %f\n", scrollBox.Offset.Y)
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
			fmt.Printf("scrollBox - after: %f\n", scrollBox.Offset.Y)
		}
	}

	// Create a new form for account/password inputs
	form := container.NewHBox(
		layout.NewSpacer(),
		container.NewVBox(
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
			rectSpacer,
		),
		layout.NewSpacer(),
	)

	body := widget.NewLabel("Your account has been successfully recovered. ")
	body.Wrapping = fyne.TextWrapWord
	body.Alignment = fyne.TextAlignCenter
	body.TextStyle = fyne.TextStyle{Bold: true}

	formSuccess := container.NewHBox(
		layout.NewSpacer(),
		container.NewVBox(
			wSpacer,
			heading2,
			rectSpacer,
			body,
			wSpacer,
			container.NewCenter(grid),
			rectSpacer,
			errorText,
			rectSpacer,
			btnCopyAddress,
			rectSpacer,
		),
		layout.NewSpacer(),
	)

	formSuccess.Hide()

	scrollBox = container.NewVScroll(
		container.NewStack(
			rectHeader,
			container.NewHBox(
				layout.NewSpacer(),
				container.NewCenter(
					form,
					formSuccess,
				),
				layout.NewSpacer(),
			),
		),
	)

	scrollBox.SetMinSize(fyne.NewSize(ui.Width, ui.Height*0.60))

	btnCreate.OnTapped = func() {
		if engram.Disk != nil {
			closeWallet()
		}

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

		getTestnet()

		var words string

		for i := 0; i < 25; i++ {
			words += seed[i] + " "
		}

		language, _, err := mnemonics.Words_To_Key(words)

		temp, err := walletapi.Create_Encrypted_Wallet_From_Recovery_Words(session.Path, session.Password, words)
		if err != nil {
			errorText.Text = err.Error()
			errorText.Color = colors.Red
			errorText.Refresh()
			return
		}

		engram.Disk = temp
		temp = nil

		if session.Testnet {
			engram.Disk.SetNetwork(false)
		} else {
			engram.Disk.SetNetwork(true)
		}

		engram.Disk.SetSeedLanguage(language)

		address := engram.Disk.GetAddress().String()

		btnCopyAddress.OnTapped = func() {
			session.Window.Clipboard().SetContent(address)
		}

		engram.Disk.Get_Balance_Rescan()
		engram.Disk.Save_Wallet()
		engram.Disk.Close_Encrypted_Wallet()

		session.WalletOpen = false
		engram.Disk = nil
		session.Path = ""
		session.Name = ""
		tx = Transfers{}

		btnCreate.Hide()
		form.Hide()
		form.Refresh()
		formSuccess.Show()
		formSuccess.Refresh()
		grid.Refresh()
		scrollBox.Refresh()
		session.Window.Canvas().Content().Refresh()
		session.Window.Canvas().Refresh(session.Window.Content())
	}

	header := container.NewVBox(
		wSpacer,
		heading,
		wSpacer,
	)

	rect1 := canvas.NewRectangle(color.Transparent)
	rect1.SetMinSize(fyne.NewSize(ui.Width, 1))

	footer := container.NewCenter(
		rect1,
		container.NewVBox(
			btnCreate,
			rectSpacer,
			container.NewHBox(
				layout.NewSpacer(),
				linkReturn,
				layout.NewSpacer(),
			),
			wSpacer,
		),
	)

	layout := container.NewBorder(
		container.NewVBox(
			header,
			scrollBox,
			rectSpacer,
		),
		footer,
		nil,
		nil,
	)
	return layout
}

func layoutAssetExplorer() fyne.CanvasObject {
	session.Domain = "app.explorer"
	resetResources()
	var data []string
	var listData binding.StringList
	var listBox *widget.List

	frame := &iframe{}
	rectLeft := canvas.NewRectangle(color.Transparent)
	rectLeft.SetMinSize(fyne.NewSize(ui.Width*0.40, 35))
	rectRight := canvas.NewRectangle(color.Transparent)
	rectRight.SetMinSize(fyne.NewSize(ui.Width*0.58, 35))
	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(fyne.NewSize(ui.Width, ui.Height*0.47))
	rectWidth := canvas.NewRectangle(color.Transparent)
	rectWidth.SetMinSize(fyne.NewSize(ui.Width, 10))

	heading := canvas.NewText("Asset Explorer", colors.Green)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(6, 5))

	results := canvas.NewText("", colors.Green)
	results.TextSize = 13

	listData = binding.BindStringList(&data)
	listBox = widget.NewListWithData(listData,
		func() fyne.CanvasObject {
			return container.NewStack(
				container.NewHBox(
					container.NewStack(
						rectLeft,
						widget.NewLabel(""),
					),
					container.NewStack(
						rectRight,
						widget.NewLabel(""),
					),
				),
			)
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			dat := di.(binding.String)
			str, err := dat.Get()
			if err != nil {
				return
			}

			split := strings.Split(str, ";;;")

			co.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*widget.Label).SetText(split[0])
			co.(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*fyne.Container).Objects[1].(*widget.Label).SetText(split[1])
			//co.(*fyne.Container).Objects[3].(*fyne.Container).Objects[1].(*widget.Label).SetText(split[3])
		})

	menu := widget.NewSelect([]string{"My Assets", "Search By SCID"}, nil)
	menu.PlaceHolder = "(Select One)"

	menuLabel := canvas.NewText("  M O R E   O P T I O N S  ", colors.Gray)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	entrySCID := widget.NewEntry()
	entrySCID.PlaceHolder = "Search by SCID"

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	btnSearch := widget.NewButton("Search", nil)
	btnSearch.OnTapped = func() {

	}

	linkBack := widget.NewHyperlinkWithStyle("Back to Dashboard", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkBack.OnTapped = func() {
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutDashboard())
		removeOverlays()
	}

	btnMyAssets := widget.NewButton("My Assets", nil)
	btnMyAssets.OnTapped = func() {
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutMyAssets())
	}

	layoutExplorer := container.NewStack(
		rectWidth,
		container.NewHBox(
			layout.NewSpacer(),
			container.NewVBox(
				rectSpacer,
				results,
				rectSpacer,
				rectSpacer,
				entrySCID,
				rectSpacer,
				rectSpacer,
				container.NewStack(
					rectList,
					listBox,
				),
				rectSpacer,
				rectSpacer,
				btnMyAssets,
			),
			layout.NewSpacer(),
		),
	)

	listing := layoutExplorer

	var assetData []string

	found := 0
	assetData = nil

	results.Text = fmt.Sprintf("  Results:  %d", found)
	results.Color = colors.Green
	results.Refresh()

	listData.Set(nil)

	if session.Offline {
		results.Text = "  Asset Explorer is disabled in offline mode."
		results.Color = colors.Gray
		results.Refresh()
	} else if gnomon.Index == nil {
		results.Text = "  Asset Explorer is disabled. Gnomon is inactive."
		results.Color = colors.Gray
		results.Refresh()
	}

	entrySCID.OnChanged = func(s string) {
		if entrySCID.Text != "" && len(s) == 64 {
			result := gnomon.Index.GravDBBackend.GetSCIDVariableDetailsAtTopoheight(s, engram.Disk.Get_Daemon_TopoHeight())

			if len(result) == 0 {
				_, err := getTxData(s)
				if err != nil {
					return
				}
			}

			err := StoreEncryptedValue("Explorer History", []byte(s), []byte(""))
			if err != nil {
				fmt.Printf("[Asset Explorer] Error saving search result: %s\n", err)
				return
			}

			scid := crypto.HashHexToHash(s)

			bal, _, err := engram.Disk.GetDecryptedBalanceAtTopoHeight(scid, -1, engram.Disk.GetAddress().String())

			title, desc, _, _, _ := getContractHeader(scid)

			if title == "" {
				title = scid.String()
			}

			if len(title) > 18 {
				title = title[0:18] + "..."
			}

			if desc == "" {
				desc = "N/A"
			}

			if len(desc) > 40 {
				desc = desc[0:40] + "..."
			}

			assetData = append(data, globals.FormatMoney(bal)+";;;"+title+";;;"+desc+";;;;;;"+scid.String())
			listData.Set(assetData)
			found += 1

			overlay := session.Window.Canvas().Overlays()
			overlay.Add(
				container.NewStack(
					&iframe{},
					canvas.NewRectangle(colors.DarkMatter),
				),
			)
			overlay.Add(
				container.NewStack(
					&iframe{},
					layoutAssetManager(s),
				),
			)
			overlay.Top().Show()

			entrySCID.Text = ""
			entrySCID.Refresh()

			results.Text = fmt.Sprintf("  Results:  %d", found)
			results.Color = colors.Green
			results.Refresh()
		}
	}

	go func() {
		if engram.Disk != nil && gnomon.Index != nil {
			for gnomon.Index.LastIndexedHeight < int64(engram.Disk.Get_Daemon_Height()) {
				results.Text = fmt.Sprintf("  Gnomon is syncing... [%d / %d]", gnomon.Index.LastIndexedHeight, int64(engram.Disk.Get_Daemon_Height()))
				results.Color = colors.Yellow
				results.Refresh()
				time.Sleep(time.Second * 1)
			}

			results.Text = fmt.Sprintf("  Loading previous scan history...")
			results.Color = colors.Yellow
			results.Refresh()

			shard, err := GetShard()
			if err != nil {
				return
			}

			store, err := graviton.NewDiskStore(shard)
			if err != nil {
				return
			}

			ss, err := store.LoadSnapshot(0)

			if err != nil {
				return
			}

			tree, err := ss.GetTree("Explorer History")
			if err != nil {
				return
			}

			c := tree.Cursor()

			for k, _, err := c.First(); err == nil; k, _, err = c.Next() {
				scid := crypto.HashHexToHash(string(k))

				bal, _, err := engram.Disk.GetDecryptedBalanceAtTopoHeight(scid, -1, engram.Disk.GetAddress().String())
				if err != nil {
					bal = 0
				}

				title, desc, _, _, _ := getContractHeader(scid)

				if title == "" {
					title = scid.String()
				}

				if len(title) > 18 {
					title = title[0:18] + "..."
				}

				if desc == "" {
					desc = "N/A"
				}

				if len(desc) > 40 {
					desc = desc[0:40] + "..."
				}

				assetData = append(data, globals.FormatMoney(bal)+";;;"+title+";;;"+desc+";;;;;;"+scid.String())
				listData.Set(assetData)
				found += 1
			}
		}

		results.Text = fmt.Sprintf("  Search History:  %d", found)
		results.Color = colors.Green
		results.Refresh()

		listData.Set(assetData)

		listBox.OnSelected = func(id widget.ListItemID) {
			split := strings.Split(assetData[id], ";;;")
			overlay := session.Window.Canvas().Overlays()
			overlay.Add(
				container.NewStack(
					&iframe{},
					canvas.NewRectangle(colors.DarkMatter),
				),
			)
			overlay.Add(
				container.NewStack(
					&iframe{},
					layoutAssetManager(split[4]),
				),
			)
			overlay.Top().Show()
			listBox.UnselectAll()
		}
		listBox.Refresh()
	}()

	top := container.NewVBox(
		rectSpacer,
		rectSpacer,
		container.NewCenter(
			heading,
		),
		rectSpacer,
		rectSpacer,
		container.NewCenter(
			listing,
		),
	)

	bottom := container.NewStack(
		container.NewVBox(
			rectSpacer,
			rectSpacer,
			container.NewStack(
				container.NewHBox(
					layout.NewSpacer(),
					line1,
					layout.NewSpacer(),
					menuLabel,
					layout.NewSpacer(),
					line2,
					layout.NewSpacer(),
				),
			),
			rectSpacer,
			rectSpacer,
			container.NewCenter(
				layout.NewSpacer(),
				linkBack,
				layout.NewSpacer(),
			),
			rectSpacer,
			rectSpacer,
			rectSpacer,
			rectSpacer,
		),
	)

	layout := container.NewStack(
		frame,
		container.NewBorder(
			top,
			bottom,
			nil,
			nil,
		),
	)

	return layout
}

func layoutMyAssets() fyne.CanvasObject {
	resetResources()
	var data []string
	var listData binding.StringList
	var listBox *widget.List

	frame := &iframe{}
	rectLeft := canvas.NewRectangle(color.Transparent)
	rectLeft.SetMinSize(fyne.NewSize(ui.Width*0.40, 35))
	rectRight := canvas.NewRectangle(color.Transparent)
	rectRight.SetMinSize(fyne.NewSize(ui.Width*0.59, 35))
	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(fyne.NewSize(ui.Width, ui.Height*0.55))
	rectWidth := canvas.NewRectangle(color.Transparent)
	rectWidth.SetMinSize(fyne.NewSize(ui.MaxWidth, 10))

	heading := canvas.NewText("My Assets", colors.Green)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(6, 5))

	results := canvas.NewText("", colors.Green)
	results.TextSize = 13

	listData = binding.BindStringList(&data)
	listBox = widget.NewListWithData(listData,
		func() fyne.CanvasObject {
			return container.NewStack(
				container.NewHBox(
					container.NewStack(
						rectLeft,
						widget.NewLabel(""),
					),
					container.NewStack(
						rectRight,
						widget.NewLabel(""),
					),
				),
			)
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			dat := di.(binding.String)
			str, err := dat.Get()
			if err != nil {
				return
			}

			split := strings.Split(str, ";;;")

			co.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*widget.Label).SetText(split[0])
			co.(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*fyne.Container).Objects[1].(*widget.Label).SetText(split[1])
			//co.(*fyne.Container).Objects[3].(*fyne.Container).Objects[1].(*widget.Label).SetText(split[3])
		})

	menuLabel := canvas.NewText("  M O R E   O P T I O N S  ", colors.Gray)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	entrySCID := widget.NewEntry()
	entrySCID.PlaceHolder = "Search by SCID"

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkBack := widget.NewHyperlinkWithStyle("Back to Asset Explorer", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkBack.OnTapped = func() {
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutAssetExplorer())
		removeOverlays()
	}

	btnRescan := widget.NewButton("  Rescan Blockchain  ", nil)
	btnRescan.Disable()

	layoutAssets := container.NewStack(
		rectWidth,
		container.NewHBox(
			layout.NewSpacer(),
			container.NewVBox(
				rectSpacer,
				results,
				rectSpacer,
				rectSpacer,
				container.NewStack(
					rectList,
					listBox,
				),
				rectSpacer,
				rectSpacer,
				btnRescan,
			),
			layout.NewSpacer(),
		),
	)

	listing := layoutAssets

	var assetData []string
	assetCount := 0
	assetTotal := 0
	owned := 0

	owned = 0
	assetData = nil
	listData.Set(nil)

	if session.Offline {
		results.Text = "  Asset tracking is disabled in offline mode."
		results.Color = colors.Gray
		results.Refresh()
	} else if gnomon.Index == nil {
		results.Text = "  Asset tracking is disabled. Gnomon is inactive."
		results.Color = colors.Gray
		results.Refresh()
	}

	go func() {
		if engram.Disk != nil && gnomon.Index != nil {
			if gnomon.Index.LastIndexedHeight < int64(engram.Disk.Get_Daemon_Height()) {
				btnRescan.Disable()
			} else {
				btnRescan.Enable()
			}

			results.Text = "  Gathering an index of smart contracts... "
			results.Color = colors.Yellow
			results.Refresh()

			for gnomon.Index.LastIndexedHeight < int64(engram.Disk.Get_Daemon_Height()) {
				results.Text = fmt.Sprintf("  Gnomon is syncing... [%d / %d]", gnomon.Index.LastIndexedHeight, int64(engram.Disk.Get_Daemon_Height()))
				results.Color = colors.Yellow
				results.Refresh()
				time.Sleep(time.Second * 1)
			}

			results.Text = fmt.Sprintf("  Loading previous scan results...")
			results.Color = colors.Yellow
			results.Refresh()

			var assetList map[string]string
			var zerobal uint64

			shard, err := GetShard()
			if err != nil {
				return
			}

			store, err := graviton.NewDiskStore(shard)
			if err != nil {
				return
			}

			ss, err := store.LoadSnapshot(0)

			if err != nil {
				return
			}

			tree, err := ss.GetTree("My Assets")
			if err != nil {
				return
			}

			c := tree.Cursor()

			for k, _, err := c.First(); err == nil; k, _, err = c.Next() {
				scid := string(k)

				hash := crypto.HashHexToHash(scid)

				bal, _, err := engram.Disk.GetDecryptedBalanceAtTopoHeight(hash, -1, engram.Disk.GetAddress().String())
				if err != nil {
					return
				} else {
					//if bal != zerobal {
					title, desc, _, _, _ := getContractHeader(hash)

					if title == "" {
						title = scid
					}

					if len(title) > 18 {
						title = title[0:18] + "..."
					}

					if desc == "" {
						desc = "N/A"
					}

					if len(desc) > 40 {
						desc = desc[0:40] + "..."
					}

					balance := globals.FormatMoney(bal)
					assetData = append(data, balance+";;;"+title+";;;"+desc+";;;;;;"+scid)
					listData.Set(assetData)
					owned += 1
				}
				//}
			}

			rescan := func() {
				assetTotal = 0
				assetCount = 0

				results.Text = fmt.Sprintf("  Indexing Gnomon results... Please wait.")
				results.Color = colors.Yellow
				results.Refresh()

				owned = 0

				assetData = []string{}
				listBox.UnselectAll()
				listData.Set(assetData)

				assetList = gnomon.Index.GravDBBackend.GetAllOwnersAndSCIDs()

				for len(assetList) < 5 {
					fmt.Printf("[Gnomon] Asset Scan Status: [%d / %d / %d]\n", gnomon.Index.LastIndexedHeight, engram.Disk.Get_Daemon_Height(), len(assetList))
					results.Color = colors.Yellow
					assetList = gnomon.Index.GravDBBackend.GetAllOwnersAndSCIDs()
					time.Sleep(time.Second * 5)
				}

				results.Text = fmt.Sprintf("  Indexing complete - Scanning balances...")
				results.Color = colors.Yellow
				results.Refresh()

				assetList = gnomon.Index.GravDBBackend.GetAllOwnersAndSCIDs()

				contracts := []crypto.Hash{}

				for sc := range assetList {
					scid := crypto.HashHexToHash(sc)

					if !scid.IsZero() {
						assetCount += 1
						contracts = append(contracts, scid)
					}
				}

				wg := sync.WaitGroup{}
				maxWorkers := 15
				lastJob := 0

			parse:

				if lastJob+maxWorkers > len(contracts) {
					maxWorkers = assetCount - lastJob
				}

				wg.Add(maxWorkers)

				// Parse each smart contract ID and check for a balance
				for i := 0; i < maxWorkers; i++ {
					index := lastJob
					go func(i int) {
						defer wg.Done()

						scid := contracts[index]

						desc := ""
						title := ""

						assetTotal += 1

						results.Text = "  Scanning smart contracts... " + fmt.Sprintf("%d / %d", assetTotal, assetCount)
						results.Color = colors.Yellow
						results.Refresh()

						balance := globals.FormatMoney(0)

						bal, _, err := engram.Disk.GetDecryptedBalanceAtTopoHeight(scid, -1, engram.Disk.GetAddress().String())
						if err != nil {
							return
						} else {
							balance = globals.FormatMoney(bal)

							if bal != zerobal {
								err = StoreEncryptedValue("My Assets", []byte(scid.String()), []byte(balance))
								if err != nil {
									fmt.Printf("[History] Failed to store asset: %s\n", err)
								}

								title, desc, _, _, _ = getContractHeader(scid)

								if title == "" {
									title = scid.String()
								}

								if len(title) > 20 {
									title = title[0:20] + "..."
								}

								if desc == "" {
									desc = "N/A"
								}

								if len(desc) > 40 {
									desc = desc[0:40] + "..."
								}

								owned += 1
								assetData = append(assetData, balance+";;;"+title+";;;"+desc+";;;;;;"+scid.String())
								listData.Set(assetData)
								fmt.Printf("[Assets] Found asset: %s\n", scid.String())
							}
						}
					}(i)

					lastJob += 1
				}

				wg.Wait()

				if lastJob < len(contracts) {
					goto parse
				}

				results.Text = fmt.Sprintf("  Owned Assets:  %d", owned)
				results.Color = colors.Green
				results.Refresh()

				listData.Set(assetData)
			}

			btnRescan.OnTapped = rescan

			if len(assetData) == 0 {
				rescan()
			}

			results.Text = fmt.Sprintf("  Owned Assets:  %d", owned)
			results.Color = colors.Green
			results.Refresh()

			listData.Set(assetData)

			listBox.OnSelected = func(id widget.ListItemID) {
				split := strings.Split(assetData[id], ";;;")
				overlay := session.Window.Canvas().Overlays()
				overlay.Add(
					container.NewStack(
						&iframe{},
						canvas.NewRectangle(colors.DarkMatter),
					),
				)
				overlay.Add(
					container.NewStack(
						&iframe{},
						layoutAssetManager(split[4]),
					),
				)
				overlay.Top().Show()
				listBox.UnselectAll()
			}
			listBox.Refresh()
			btnRescan.Enable()
		}
	}()

	top := container.NewVBox(
		rectSpacer,
		rectSpacer,
		container.NewCenter(
			heading,
		),
		rectSpacer,
		rectSpacer,
		container.NewCenter(
			listing,
		),
	)

	bottom := container.NewStack(
		container.NewVBox(
			rectSpacer,
			rectSpacer,
			container.NewStack(
				container.NewHBox(
					layout.NewSpacer(),
					line1,
					layout.NewSpacer(),
					menuLabel,
					layout.NewSpacer(),
					line2,
					layout.NewSpacer(),
				),
			),
			rectSpacer,
			rectSpacer,
			container.NewCenter(
				layout.NewSpacer(),
				linkBack,
				layout.NewSpacer(),
			),
			rectSpacer,
			rectSpacer,
			rectSpacer,
			rectSpacer,
		),
	)

	layout := container.NewStack(
		frame,
		container.NewBorder(
			top,
			bottom,
			nil,
			nil,
		),
	)

	return layout
}

func layoutAssetManager(scid string) fyne.CanvasObject {
	resetResources()

	session.Domain = "app.manager"

	wSpacer := widget.NewLabel(" ")

	frame := &iframe{}

	rectBox := canvas.NewRectangle(color.Transparent)
	rectBox.SetMinSize(fyne.NewSize(ui.MaxWidth*0.99, ui.MaxHeight*0.58))
	rectWidth90 := canvas.NewRectangle(color.Transparent)
	rectWidth90.SetMinSize(fyne.NewSize(ui.Width, 10))

	heading := canvas.NewText("Asset Manager", colors.Green)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(6, 5))

	labelSigner := canvas.NewText("SMART  CONTRACT  AUTHOR", colors.Gray)
	labelSigner.TextSize = 11
	labelSigner.Alignment = fyne.TextAlignLeading
	labelSigner.TextStyle = fyne.TextStyle{Bold: true}

	labelOwner := canvas.NewText("SMART  CONTRACT  OWNER", colors.Gray)
	labelOwner.TextSize = 11
	labelOwner.Alignment = fyne.TextAlignLeading
	labelOwner.TextStyle = fyne.TextStyle{Bold: true}

	labelSCID := canvas.NewText("SMART  CONTRACT  ID", colors.Gray)
	labelSCID.TextSize = 11
	labelSCID.Alignment = fyne.TextAlignLeading
	labelSCID.TextStyle = fyne.TextStyle{Bold: true}

	labelBalance := canvas.NewText("ASSET  BALANCE", colors.Gray)
	labelBalance.TextSize = 11
	labelBalance.Alignment = fyne.TextAlignLeading
	labelBalance.TextStyle = fyne.TextStyle{Bold: true}

	labelExecute := canvas.NewText("EXECUTE  ACTION", colors.Gray)
	labelExecute.TextSize = 11
	labelExecute.Alignment = fyne.TextAlignLeading
	labelExecute.TextStyle = fyne.TextStyle{Bold: true}

	menuLabel := canvas.NewText("  M O R E   O P T I O N S  ", colors.Gray)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	entryAddress := widget.NewEntry()
	entryAddress.PlaceHolder = "Username or Address"

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	sc := widget.NewLabel(scid)
	sc.Wrapping = fyne.TextWrap(fyne.TextWrapWord)

	hash := crypto.HashHexToHash(scid)
	name, desc, icon, owner, code := getContractHeader(hash)

	if owner == "" {
		owner = "--"
	}

	signer := "--"

	result, err := getTxData(scid)
	if err != nil {
		signer = "--"
	} else {
		signer = result.Txs[0].Signer
	}

	labelName := widget.NewRichTextFromMarkdown(name)
	labelName.Wrapping = fyne.TextWrapOff
	labelName.ParseMarkdown("## " + name)

	labelDesc := widget.NewRichTextFromMarkdown(desc)
	labelDesc.Wrapping = fyne.TextWrapWord
	labelDesc.ParseMarkdown(desc)

	textSigner := widget.NewRichTextFromMarkdown(owner)
	textSigner.Wrapping = fyne.TextWrapWord
	textSigner.ParseMarkdown(signer)

	textOwner := widget.NewRichTextFromMarkdown(owner)
	textOwner.Wrapping = fyne.TextWrapWord
	textOwner.ParseMarkdown(owner)

	btnSend := widget.NewButton("Send Asset", nil)

	entryAddress.Validator = func(s string) error {
		btnSend.Text = "Send Asset"
		btnSend.Refresh()
		_, err := globals.ParseValidateAddress(s)
		if err != nil {
			go func() {
				exists, _, err := checkUsername(s, -1)
				if err != nil && !exists {
					btnSend.Disable()
					entryAddress.SetValidationError(errors.New("invalid username or address"))
				} else {
					entryAddress.SetValidationError(nil)
					btnSend.Enable()
				}
			}()
		} else {
			entryAddress.SetValidationError(nil)
			btnSend.Enable()
		}
		return nil
	}

	entryAmount := widget.NewEntry()
	entryAmount.PlaceHolder = "Asset Amount (Numbers Only)"
	entryAmount.Validator = func(s string) error {
		if s != "" {
			amount, err := globals.ParseAmount(s)
			if err != nil {
				btnSend.Disable()
				entryAmount.SetValidationError(errors.New("invalid amount entered"))
				return err
			} else {
				bal, _, err := engram.Disk.GetDecryptedBalanceAtTopoHeight(hash, -1, engram.Disk.GetAddress().String())
				if err != nil {
					btnSend.Disable()
					entryAmount.SetValidationError(errors.New("error parsing asset balance"))
					return err
				} else {
					if amount > bal || amount == 0 {
						err = errors.New("insufficient asset balance")
						btnSend.Text = "Insufficient transfer amount..."
						btnSend.Disable()
						entryAmount.SetValidationError(err)
						return err
					}
				}
			}
		}

		btnSend.Text = "Send Asset"
		btnSend.Enable()
		entryAmount.SetValidationError(nil)

		return nil
	}

	var zerobal uint64

	balance := widget.NewRichTextFromMarkdown(fmt.Sprintf("%d", zerobal))
	balance.Wrapping = fyne.TextWrapWord

	btnSend.OnTapped = func() {
		btnSend.Text = "Setting up transfer..."
		btnSend.Disable()
		btnSend.Refresh()

		txid, err := transferAsset(hash, entryAddress.Text, entryAmount.Text)
		if err != nil {
			entryAddress.Text = ""
			entryAddress.Refresh()
			entryAmount.Text = ""
			entryAmount.Refresh()
			btnSend.Text = "Transaction Failed..."
			btnSend.Disable()
			btnSend.Refresh()
		} else {
			entryAddress.Text = ""
			entryAddress.Refresh()
			entryAmount.Text = ""
			entryAmount.Refresh()
			btnSend.Text = "Confirming..."
			btnSend.Disable()
			btnSend.Refresh()

			go func() {
				walletapi.WaitNewHeightBlock()

				for session.Domain == "app.manager" {
					result := engram.Disk.Get_Payments_TXID(txid.String())

					if result.TXID != txid.String() {
						time.Sleep(time.Second * 1)
					} else {
						break
					}
				}

				bal, _, err := engram.Disk.GetDecryptedBalanceAtTopoHeight(hash, -1, engram.Disk.GetAddress().String())
				if err == nil {
					err = StoreEncryptedValue("My Assets", []byte(hash.String()), []byte(globals.FormatMoney(bal)))
					if err != nil {
						fmt.Printf("[Asset] Error storing new asset balance for: %s\n", hash)
					}
					balance.ParseMarkdown(globals.FormatMoney(bal))
					balance.Refresh()
				}

				if bal != zerobal {
					btnSend.Text = "Send Asset"
					btnSend.Enable()
					btnSend.Refresh()
				} else {
					btnSend.Text = "You do not own this asset"
					btnSend.Disable()
					btnSend.Refresh()
				}
			}()
		}
	}

	bal, _, err := engram.Disk.GetDecryptedBalanceAtTopoHeight(hash, -1, engram.Disk.GetAddress().String())
	if err == nil {

		balance.ParseMarkdown(globals.FormatMoney(bal))
		balance.Refresh()
	}

	linkBack := widget.NewHyperlinkWithStyle("Back to Asset Explorer", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkBack.OnTapped = func() {
		removeOverlays()
		session.Domain = "app.explorer"
	}

	var image *canvas.Image
	image = canvas.NewImageFromResource(resourceBlockGrayPng)
	image.SetMinSize(fyne.NewSize(ui.Width*0.2, ui.Width*0.2))
	image.FillMode = canvas.ImageFillContain

	if icon != "" {
		var path fyne.Resource
		path, err = fyne.LoadResourceFromURLString(icon)
		if err != nil {
			image.Resource = resourceBlockGrayPng
		} else {
			image.Resource = path
		}

		image.SetMinSize(fyne.NewSize(ui.Width*0.2, ui.Width*0.2))
		image.FillMode = canvas.ImageFillContain
		image.Refresh()
	}

	if name == "" {
		labelName.ParseMarkdown("## No name provided")
	}

	if desc == "" {
		labelDesc.ParseMarkdown("No description provided")
	}

	if bal != zerobal {
		btnSend.Text = "Send Asset"
		btnSend.Enable()
	} else {
		btnSend.Text = "You do not own this asset"
		btnSend.Disable()
	}
	btnSend.Refresh()

	linkCopySigner := widget.NewHyperlinkWithStyle("Copy Address", nil, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	linkCopySigner.OnTapped = func() {
		session.Window.Clipboard().SetContent(signer)
	}

	linkCopyOwner := widget.NewHyperlinkWithStyle("Copy Address", nil, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	linkCopyOwner.OnTapped = func() {
		session.Window.Clipboard().SetContent(owner)
	}

	linkMessageAuthor := widget.NewHyperlinkWithStyle("Message the Author", nil, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	linkMessageAuthor.OnTapped = func() {
		if signer != "" && signer != "--" {
			messages.Contact = signer
			session.Window.Canvas().SetContent(layoutTransition())
			removeOverlays()
			session.Window.Canvas().SetContent(layoutPM())
		}
	}

	linkMessageOwner := widget.NewHyperlinkWithStyle("Message the Owner", nil, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	linkMessageOwner.OnTapped = func() {
		if owner != "" && owner != "--" {
			messages.Contact = owner
			session.Window.Canvas().SetContent(layoutTransition())
			removeOverlays()
			session.Window.Canvas().SetContent(layoutPM())
		}
	}

	linkCopySCID := widget.NewHyperlinkWithStyle("Copy SCID", nil, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	linkCopySCID.OnTapped = func() {
		session.Window.Clipboard().SetContent(scid)
	}

	linkView := widget.NewHyperlinkWithStyle("View in Explorer", nil, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	linkView.OnTapped = func() {
		if engram.Disk.GetNetwork() {
			link, _ := url.Parse("https://explorer.dero.io/tx/" + scid)
			_ = fyne.CurrentApp().OpenURL(link)
		} else {
			link, _ := url.Parse("https://testnetexplorer.dero.io/tx/" + scid)
			_ = fyne.CurrentApp().OpenURL(link)
		}
	}

	// Now let's parse the smart contract code for exported functions

	var contract dvm.SmartContract

	contract, _, err = dvm.ParseSmartContract(code)
	if err != nil {
		contract = dvm.SmartContract{}
	}

	data := []string{}

	for f := range contract.Functions {
		r, _ := utf8.DecodeRuneInString(contract.Functions[f].Name)

		if !unicode.IsUpper(r) {
			fmt.Printf("[DVM] Function %s is not an exported function - skipping it\n", contract.Functions[f].Name)
		} else if contract.Functions[f].Name == "Initialize" || contract.Functions[f].Name == "InitializePrivate" {
			fmt.Printf("[DVM] Function %s is an initialization function - skipping it\n", contract.Functions[f].Name)
		} else {
			data = append(data, contract.Functions[f].Name)
		}
	}

	data = append(data, " ")

	var paramList []fyne.Widget
	var dero_amount uint64
	var asset_amount uint64

	functionList := widget.NewSelect(data, nil)
	functionList.OnChanged = func(s string) {
		if s == " " {
			functionList.ClearSelected()
			return
		}

		var params []dvm.Variable

		overlay := session.Window.Canvas().Overlays()

		for f := range contract.Functions {
			if contract.Functions[f].Name == s {
				params = contract.Functions[f].Params

				header := canvas.NewText("EXECUTE  CONTRACT  FUNCTION", colors.Gray)
				header.TextSize = 14
				header.Alignment = fyne.TextAlignCenter
				header.TextStyle = fyne.TextStyle{Bold: true}

				funcName := canvas.NewText(s, colors.Account)
				funcName.TextSize = 22
				funcName.Alignment = fyne.TextAlignCenter
				funcName.TextStyle = fyne.TextStyle{Bold: true}

				linkClose := widget.NewHyperlinkWithStyle("Close", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
				linkClose.OnTapped = func() {
					dero_amount = 0
					asset_amount = 0
					overlay.Top().Hide()
					overlay.Remove(overlay.Top())
					overlay.Remove(overlay.Top())
				}

				span := canvas.NewRectangle(color.Transparent)
				span.SetMinSize(fyne.NewSize(ui.Width, 10))

				overlay.Add(
					container.NewStack(
						&iframe{},
						canvas.NewRectangle(colors.DarkMatter),
					),
				)

				entryDEROValue := widget.NewEntry()
				entryDEROValue.PlaceHolder = "DERO Amount (Numbers Only)"
				entryDEROValue.Validator = func(s string) error {
					dero_amount, err = globals.ParseAmount(s)
					if err != nil {
						entryDEROValue.SetValidationError(err)
						return err
					}

					return nil
				}

				entryAssetValue := widget.NewEntry()
				entryAssetValue.PlaceHolder = "Asset Amount (Numbers Only)"
				entryAssetValue.Validator = func(s string) error {
					asset_amount, err = globals.ParseAmount(s)
					if err != nil {
						entryAssetValue.SetValidationError(err)
						return err
					}

					return nil
				}

				a := container.NewStack(
					span,
					entryAssetValue,
				)

				d := container.NewStack(
					span,
					entryDEROValue,
				)

				paramsContainer := container.NewVBox()

				// Scan code for ASSETVALUE and DEROVALUE
				for l := range contract.Functions[f].Lines {
					for i := range contract.Functions[f].Lines[l] {
						existsDEROValue := false
						existsAssetValue := false

						for v := range paramList {
							if paramList[v] == entryDEROValue {
								existsDEROValue = true
							} else if paramList[v] == entryAssetValue {
								existsAssetValue = true
							}
						}

						if strings.Contains(contract.Functions[f].Lines[l][i], "DEROVALUE") && !existsDEROValue {
							paramList = append(paramList, entryDEROValue)
							paramsContainer.Add(d)
							paramsContainer.Refresh()
						}

						if strings.Contains(contract.Functions[f].Lines[l][i], "ASSETVALUE") && !existsAssetValue {
							paramList = append(paramList, entryAssetValue)
							paramsContainer.Add(a)
							paramsContainer.Refresh()
							break
						}
					}
				}

				btnExecute := widget.NewButton("Execute", nil)

				overlay.Add(
					container.NewStack(
						&iframe{},
						container.NewCenter(
							container.NewVBox(
								span,
								container.NewCenter(
									header,
								),
								rectSpacer,
								rectSpacer,
								container.NewCenter(
									funcName,
								),
								wSpacer,
								paramsContainer,
								wSpacer,
								btnExecute,
								rectSpacer,
								rectSpacer,
								container.NewHBox(
									layout.NewSpacer(),
									linkClose,
									layout.NewSpacer(),
								),
								rectSpacer,
								rectSpacer,
							),
						),
					),
				)

				for p := range params {
					entry := widget.NewEntry()
					entry.PlaceHolder = params[p].Name
					if params[p].Type == 0x4 {
						entry.PlaceHolder = params[p].Name + " (Numbers Only)"
					}
					entry.Validator = func(s string) error {
						for p := range params {
							if params[p].Type == 0x5 {
								if params[p].Name == entry.PlaceHolder {
									fmt.Printf("[%s] String: %s\n", params[p].Name, s)
									params[p].ValueString = s
								}
							} else if params[p].Type == 0x4 {
								if params[p].Name+" (Numbers Only)" == entry.PlaceHolder {
									amount, err := globals.ParseAmount(s)
									if err != nil {
										fmt.Printf("[%s] Err: %s\n", params[p].Name, err)
										entry.SetValidationError(err)
										return err
									} else {
										fmt.Printf("[%s] Amount: %d\n", params[p].Name, amount)
										params[p].ValueUint64 = amount
									}
								}
							}
						}

						return nil
					}

					c := container.NewStack(
						span,
						entry,
					)

					paramList = append(paramList, entry)
					paramsContainer.Add(c)
					paramsContainer.Refresh()

				}

				btnExecute.OnTapped = func() {
					var funcType rpc.DataType
					for f := range contract.Functions {
						if contract.Functions[f].Name == funcName.Text {
							params = contract.Functions[f].Params

							if contract.Functions[f].ReturnValue.Type == 0x4 {
								funcType = rpc.DataUint64
							} else {
								funcType = rpc.DataUint64
							}
						}
					}

					err = executeContractFunction(hash, dero_amount, asset_amount, funcName.Text, funcType, params)
					if err != nil {
						btnExecute.Text = "Error executing function..."
						btnExecute.Disable()
						btnExecute.Refresh()
					} else {
						btnExecute.Text = "Function executed successfully!"
						btnExecute.Disable()
						btnExecute.Refresh()
					}
				}

				paramsContainer.Refresh()
				overlay.Top().Show()
				functionList.ClearSelected()
			}
		}
	}

	center := container.NewStack(
		rectBox,
		container.NewVScroll(
			container.NewStack(
				rectWidth90,
				container.NewHBox(
					layout.NewSpacer(),
					container.NewVBox(
						container.NewHBox(
							image,
							rectSpacer,
							container.NewVBox(
								layout.NewSpacer(),
								labelName,
								layout.NewSpacer(),
							),
							layout.NewSpacer(),
						),
						rectSpacer,
						rectSpacer,
						labelDesc,
						rectSpacer,
						rectSpacer,
						labelSigner,
						rectSpacer,
						textSigner,
						container.NewHBox(
							linkMessageAuthor,
							layout.NewSpacer(),
						),
						container.NewHBox(
							linkCopySigner,
							layout.NewSpacer(),
						),
						rectSpacer,
						rectSpacer,
						labelOwner,
						rectSpacer,
						textOwner,
						container.NewHBox(
							linkMessageOwner,
							layout.NewSpacer(),
						),
						container.NewHBox(
							linkCopyOwner,
							layout.NewSpacer(),
						),
						rectSpacer,
						rectSpacer,
						labelSCID,
						rectSpacer,
						container.NewStack(
							rectWidth90,
							sc,
						),
						container.NewHBox(
							linkView,
							layout.NewSpacer(),
						),
						container.NewHBox(
							linkCopySCID,
							layout.NewSpacer(),
						),
						rectSpacer,
						rectSpacer,
						rectSpacer,
						labelBalance,
						rectSpacer,
						balance,
						rectSpacer,
						entryAddress,
						rectSpacer,
						entryAmount,
						rectSpacer,
						btnSend,
						rectSpacer,
						rectSpacer,
						rectSpacer,
						labelExecute,
						rectSpacer,
						functionList,
						wSpacer,
					),
					layout.NewSpacer(),
				),
			),
		),
		rectSpacer,
		rectSpacer,
	)

	top := container.NewVBox(
		rectSpacer,
		rectSpacer,
	)

	bottom := container.NewStack(
		container.NewVBox(
			rectSpacer,
			rectSpacer,
			container.NewStack(
				container.NewHBox(
					layout.NewSpacer(),
					line1,
					layout.NewSpacer(),
					menuLabel,
					layout.NewSpacer(),
					line2,
					layout.NewSpacer(),
				),
			),
			rectSpacer,
			rectSpacer,
			container.NewCenter(
				layout.NewSpacer(),
				linkBack,
				layout.NewSpacer(),
			),
			rectSpacer,
			rectSpacer,
			rectSpacer,
			rectSpacer,
		),
	)

	layout := container.NewStack(
		frame,
		container.NewBorder(
			top,
			bottom,
			nil,
			center,
		),
	)

	return layout
}

func layoutTransfers() fyne.CanvasObject {
	resetResources()
	session.Domain = "app.transfers"

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

	rectStatus := canvas.NewRectangle(color.Transparent)
	rectStatus.SetMinSize(fyne.NewSize(10, 10))
	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(fyne.NewSize(ui.Width, 20))
	frame := &iframe{}
	rect.SetMinSize(fyne.NewSize(ui.Width, 30))
	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(10, 5))
	rect.SetMinSize(fyne.NewSize(10, 10))
	rectEmpty := canvas.NewRectangle(color.Transparent)
	rectEmpty.SetMinSize(fyne.NewSize(10, 10))
	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(fyne.NewSize(ui.Width, 35))
	rectListBox := canvas.NewRectangle(color.Transparent)
	rectListBox.SetMinSize(fyne.NewSize(ui.Width, ui.Height*0.43))

	menuLabel := canvas.NewText("  M O R E   O P T I O N S  ", colors.Gray)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkBack := widget.NewHyperlinkWithStyle("Back to Dashboard", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkBack.OnTapped = func() {
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutDashboard())
		removeOverlays()
	}

	var pendingList []string

	for i := 0; i < len(tx.Pending); i++ {
		pendingList = append(pendingList, strconv.Itoa(i)+","+globals.FormatMoney(tx.Pending[i].Amount)+","+tx.Pending[i].Destination)
	}

	data := binding.BindStringList(&pendingList)

	scrollBox := widget.NewListWithData(data,
		func() fyne.CanvasObject {
			c := container.NewStack(
				rectList,
				container.NewHBox(
					canvas.NewText("", colors.Account),
					layout.NewSpacer(),
					canvas.NewText("", colors.Account),
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
			dest = "   " + dest[0:4] + " ... " + dest[len(dataItem[2])-10:]
			co.(*fyne.Container).Objects[1].(*fyne.Container).Objects[0].(*canvas.Text).Text = dest
			co.(*fyne.Container).Objects[1].(*fyne.Container).Objects[0].(*canvas.Text).TextSize = 17
			co.(*fyne.Container).Objects[1].(*fyne.Container).Objects[0].(*canvas.Text).TextStyle.Bold = true
			co.(*fyne.Container).Objects[1].(*fyne.Container).Objects[2].(*canvas.Text).Text = dataItem[1] + "   "
			co.(*fyne.Container).Objects[1].(*fyne.Container).Objects[2].(*canvas.Text).TextSize = 17
			co.(*fyne.Container).Objects[1].(*fyne.Container).Objects[2].(*canvas.Text).TextStyle.Bold = true
		})

	scrollBox.OnSelected = func(id widget.ListItemID) {
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutTransfersDetail(id))
	}

	btnSend := widget.NewButton("Send All", nil)

	btnClear := widget.NewButton("Clear", func() {
		pendingList = pendingList[:0]
		tx = Transfers{}
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutTransfers())
	})

	if len(pendingList) > 0 {
		btnClear.Enable()
		btnSend.Enable()
	} else {
		btnClear.Disable()
		btnSend.Disable()
	}

	if session.Offline {
		btnSend.Text = "Disabled in Offline Mode"
		btnSend.Disable()
	}

	btnSend.OnTapped = func() {
		overlay := session.Window.Canvas().Overlays()

		header := canvas.NewText("ACCOUNT  VERIFICATION  REQUIRED", colors.Gray)
		header.TextSize = 14
		header.Alignment = fyne.TextAlignCenter
		header.TextStyle = fyne.TextStyle{Bold: true}

		subHeader := canvas.NewText("Confirm Password", colors.Account)
		subHeader.TextSize = 22
		subHeader.Alignment = fyne.TextAlignCenter
		subHeader.TextStyle = fyne.TextStyle{Bold: true}

		linkClose := widget.NewHyperlinkWithStyle("Cancel", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		linkClose.OnTapped = func() {
			overlay := session.Window.Canvas().Overlays()
			overlay.Top().Hide()
			overlay.Remove(overlay.Top())
			overlay.Remove(overlay.Top())
		}

		btnSubmit := widget.NewButton("Submit", nil)

		entryPassword := widget.NewEntry()
		entryPassword.Password = true
		entryPassword.PlaceHolder = "Password"
		entryPassword.OnChanged = func(s string) {
			if s == "" {
				btnSubmit.Text = "Submit"
				btnSubmit.Disable()
				btnSubmit.Refresh()
			} else {
				btnSubmit.Text = "Submit"
				btnSubmit.Enable()
				btnSubmit.Refresh()
			}
		}

		btnSubmit.OnTapped = func() {
			if engram.Disk.Check_Password(entryPassword.Text) {
				removeOverlays()
				if len(tx.Pending) == 0 {
					return
				} else {
					btnSend.Text = "Setting up transfer..."
					btnSend.Disable()
					btnSend.Refresh()
					txid, err := sendTransfers()
					if err != nil {
						btnSend.Text = "Send All"
						btnSend.Enable()
						btnSend.Refresh()
						return
					}

					go func() {
						btnClear.Disable()
						btnSend.Text = "Confirming..."
						btnSend.Refresh()

						walletapi.WaitNewHeightBlock()

						for session.Domain == "app.transfers" {
							result := engram.Disk.Get_Payments_TXID(txid.String())

							if result.TXID == txid.String() {
								btnSend.Text = "Transfer Successful!"
								btnSend.Refresh()

								break
							}

							time.Sleep(time.Second * 1)
						}
					}()

					pendingList = pendingList[:0]
					data.Reload()
					btnSend.Disable()
					btnClear.Disable()
				}
			} else {
				btnSubmit.Text = "Invalid Password..."
				btnSubmit.Disable()
				btnSubmit.Refresh()
			}
		}

		btnSubmit.Disable()

		span := canvas.NewRectangle(color.Transparent)
		span.SetMinSize(fyne.NewSize(ui.Width, 10))

		overlay.Add(
			container.NewStack(
				&iframe{},
				canvas.NewRectangle(colors.DarkMatter),
			),
		)

		overlay.Add(
			container.NewStack(
				&iframe{},
				container.NewCenter(
					container.NewVBox(
						span,
						container.NewCenter(
							header,
						),
						rectSpacer,
						rectSpacer,
						subHeader,
						widget.NewLabel(""),
						container.NewCenter(
							container.NewStack(
								span,
								entryPassword,
							),
						),
						rectSpacer,
						rectSpacer,
						btnSubmit,
						rectSpacer,
						rectSpacer,
						container.NewHBox(
							layout.NewSpacer(),
							linkClose,
							layout.NewSpacer(),
						),
						rectSpacer,
						rectSpacer,
					),
				),
			),
		)
	}

	session.Window.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
		if session.Domain != "app.transfers" {
			return
		}

		if k.Name == fyne.KeyDown {
			session.Dashboard = "main"

			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutDashboard())
			removeOverlays()
		}
	})

	sendForm := container.NewVBox(
		wSpacer,
		sendHeading,
		rectSpacer,
		wSpacer,
		container.NewStack(
			rectListBox,
			scrollBox,
		),
		wSpacer,
		btnSend,
		rectSpacer,
		btnClear,
		rectSpacer,
		rectSpacer,
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

	bottom := container.NewStack(
		container.NewVBox(
			container.NewStack(
				container.NewHBox(
					layout.NewSpacer(),
					line1,
					layout.NewSpacer(),
					menuLabel,
					layout.NewSpacer(),
					line2,
					layout.NewSpacer(),
				),
			),
			rectSpacer,
			rectSpacer,
			container.NewCenter(
				layout.NewSpacer(),
				linkBack,
				layout.NewSpacer(),
			),
			rectSpacer,
			rectSpacer,
			rectSpacer,
			rectSpacer,
		),
	)

	c := container.NewBorder(
		features,
		bottom,
		nil,
		nil,
	)

	layout := container.NewStack(
		frame,
		c,
	)

	return layout
}

func layoutTransfersDetail(index int) fyne.CanvasObject {
	resetResources()

	wSpacer := widget.NewLabel(" ")

	rectWidth := canvas.NewRectangle(color.Transparent)
	rectWidth.SetMinSize(fyne.NewSize(ui.MaxWidth*0.99, 10))

	rectWidth90 := canvas.NewRectangle(color.Transparent)
	rectWidth90.SetMinSize(fyne.NewSize(ui.Width, 10))

	frame := &iframe{}

	heading := canvas.NewText("T R A N S F E R    D E T A I L", colors.Gray)
	heading.TextSize = 16
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(6, 5))

	labelDestination := canvas.NewText("RECEIVER  ADDRESS", colors.Gray)
	labelDestination.TextSize = 11
	labelDestination.Alignment = fyne.TextAlignLeading
	labelDestination.TextStyle = fyne.TextStyle{Bold: true}

	labelAmount := canvas.NewText("AMOUNT", colors.Gray)
	labelAmount.TextSize = 11
	labelAmount.Alignment = fyne.TextAlignLeading
	labelAmount.TextStyle = fyne.TextStyle{Bold: true}

	labelService := canvas.NewText("SERVICE  ADDRESS", colors.Gray)
	labelService.TextSize = 11
	labelService.Alignment = fyne.TextAlignLeading
	labelService.TextStyle = fyne.TextStyle{Bold: true}

	labelDestPort := canvas.NewText("DESTINATION  PORT", colors.Gray)
	labelDestPort.TextSize = 11
	labelDestPort.TextStyle = fyne.TextStyle{Bold: true}

	labelSourcePort := canvas.NewText("SOURCE  PORT", colors.Gray)
	labelSourcePort.TextSize = 11
	labelSourcePort.TextStyle = fyne.TextStyle{Bold: true}

	labelFees := canvas.NewText("TRANSACTION  FEES", colors.Gray)
	labelFees.TextSize = 11
	labelFees.TextStyle = fyne.TextStyle{Bold: true}

	labelPayload := canvas.NewText("PAYLOAD", colors.Gray)
	labelPayload.TextSize = 11
	labelPayload.TextStyle = fyne.TextStyle{Bold: true}

	labelReply := canvas.NewText("REPLY  ADDRESS", colors.Gray)
	labelReply.TextSize = 11
	labelReply.TextStyle = fyne.TextStyle{Bold: true}

	menuLabel := canvas.NewText("  M O R E   O P T I O N S  ", colors.Gray)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	details := tx.Pending[index]

	valueDestination := widget.NewRichTextFromMarkdown("--")
	valueDestination.Wrapping = fyne.TextWrapBreak

	valueType := widget.NewRichTextFromMarkdown("--")
	valueType.Wrapping = fyne.TextWrapOff

	if details.Destination != "" {
		address, _ := globals.ParseValidateAddress(details.Destination)
		if address.IsIntegratedAddress() {
			valueDestination.ParseMarkdown(address.BaseAddress().String())
			valueType.ParseMarkdown("### SERVICE")
		} else {
			valueDestination.ParseMarkdown(details.Destination)
			valueType.ParseMarkdown("### NORMAL")
		}
	}

	valueReply := widget.NewRichTextFromMarkdown("--")
	valueReply.Wrapping = fyne.TextWrapBreak

	if details.Payload_RPC.HasValue(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString) {
		if details.Payload_RPC.Value(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString).(string) != "" {
			valueReply.ParseMarkdown("" + details.Payload_RPC.Value(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString).(string))
		}
	}

	valuePayload := widget.NewRichTextFromMarkdown("--")
	valuePayload.Wrapping = fyne.TextWrapBreak

	if details.Payload_RPC.HasValue(rpc.RPC_COMMENT, rpc.DataString) {
		if details.Payload_RPC.Value(rpc.RPC_COMMENT, rpc.DataString).(string) != "" {
			valuePayload.ParseMarkdown("" + details.Payload_RPC.Value(rpc.RPC_COMMENT, rpc.DataString).(string))
		}
	}

	valueAmount := canvas.NewText("", colors.Account)
	valueAmount.TextSize = 22
	valueAmount.TextStyle = fyne.TextStyle{Bold: true}
	valueAmount.Text = " " + globals.FormatMoney(details.Amount)

	linkBack := widget.NewHyperlinkWithStyle("Back to Transfers", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkBack.OnTapped = func() {
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutTransfers())
	}

	btnDelete := widget.NewButton("Cancel Transfer", nil)
	btnDelete.OnTapped = func() {
		if len(tx.Pending) > index+1 {
			tx.Pending = append(tx.Pending[:index], tx.Pending[index+1:]...)
		} else if len(tx.Pending) == 1 {
			tx = Transfers{}
		} else {
			tx.Pending = append(tx.Pending[:index])
		}

		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutTransfers())
	}

	top := container.NewVBox(
		rectSpacer,
		rectSpacer,
		container.NewCenter(
			heading,
		),
		rectSpacer,
		container.NewCenter(
			valueType,
		),
		rectSpacer,
		rectSpacer,
	)

	center := container.NewStack(
		container.NewVScroll(
			container.NewStack(
				rectWidth,
				container.NewHBox(
					layout.NewSpacer(),
					container.NewVBox(
						rectSpacer,
						rectSpacer,
						labelDestination,
						rectSpacer,
						valueDestination,
						rectSpacer,
						rectSpacer,
						labelAmount,
						rectSpacer,
						container.NewStack(
							rectWidth90,
							valueAmount,
						),
						rectSpacer,
						rectSpacer,
						labelReply,
						rectSpacer,
						valueReply,
						rectSpacer,
						rectSpacer,
						labelPayload,
						rectSpacer,
						container.NewStack(
							rectWidth90,
							valuePayload,
						),
						wSpacer,
					),
					layout.NewSpacer(),
				),
			),
		),
	)

	bottom := container.NewStack(
		container.NewVBox(
			rectSpacer,
			rectSpacer,
			container.NewStack(
				rectWidth,
				container.NewHBox(
					layout.NewSpacer(),
					container.NewStack(
						rectWidth90,
						btnDelete,
					),
					layout.NewSpacer(),
				),
			),
			rectSpacer,
			rectSpacer,
			rectSpacer,
			container.NewStack(
				container.NewHBox(
					layout.NewSpacer(),
					line1,
					layout.NewSpacer(),
					menuLabel,
					layout.NewSpacer(),
					line2,
					layout.NewSpacer(),
				),
			),
			rectSpacer,
			rectSpacer,
			container.NewCenter(
				layout.NewSpacer(),
				linkBack,
				layout.NewSpacer(),
			),
			rectSpacer,
			rectSpacer,
			rectSpacer,
			rectSpacer,
		),
	)

	layout := container.NewStack(
		frame,
		container.NewBorder(
			top,
			bottom,
			nil,
			center,
		),
	)

	return layout
}

func layoutTransition() fyne.CanvasObject {
	frame := &iframe{}
	resizeWindow(ui.MaxWidth, ui.MaxHeight)

	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(fyne.NewSize(ui.Width*0.45, ui.Width*0.45))

	if res.loading == nil {
		res.loading, _ = x.NewAnimatedGifFromResource(resourceLoadingGif)
		res.loading.SetMinSize(fyne.NewSize(ui.Width*0.45, ui.Width*0.45))
		res.loading.Resize(fyne.NewSize(ui.Width*0.45, ui.Width*0.45))
	}

	res.loading.Start()

	layout := container.NewStack(
		frame,
		container.NewCenter(
			rect,
			res.loading,
		),
	)

	return layout
}

func layoutSettings() fyne.CanvasObject {
	resetResources()
	stopGnomon()

	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(fyne.NewSize(ui.Width, 10))
	rectScroll := canvas.NewRectangle(color.Transparent)
	rectScroll.SetMinSize(fyne.NewSize(ui.Width, ui.Height*0.65))
	frame := &iframe{}
	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(10, 5))

	heading := canvas.NewText("My Settings", colors.Green)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	labelNetwork := canvas.NewText("NETWORK", colors.Gray)
	labelNetwork.TextStyle = fyne.TextStyle{Bold: true}
	labelNetwork.TextSize = 14

	labelNode := canvas.NewText("CONNECTION", colors.Gray)
	labelNode.TextStyle = fyne.TextStyle{Bold: true}
	labelNode.TextSize = 14

	labelSecurity := canvas.NewText("SECURITY", colors.Gray)
	labelSecurity.TextStyle = fyne.TextStyle{Bold: true}
	labelSecurity.TextSize = 14

	labelGnomon := canvas.NewText("GNOMON", colors.Gray)
	labelGnomon.TextStyle = fyne.TextStyle{Bold: true}
	labelGnomon.TextSize = 14

	textGnomon := widget.NewRichTextWithText("Gnomon scans and indexes blockchain data in order to unlock more features, like native asset tracking.")
	textGnomon.Wrapping = fyne.TextWrapWord

	textCyberdeck := widget.NewRichTextWithText("A username and password is required in order to allow application connectivity.")
	textCyberdeck.Wrapping = fyne.TextWrapWord

	btnRestore := widget.NewButton("Restore Defaults", nil)
	btnDelete := widget.NewButton("Clear Local Data", nil)

	scrollBox := container.NewVScroll(nil)

	entryAddress := widget.NewEntry()
	entryAddress.Validator = func(s string) (err error) {
		/*
			_, err := net.ResolveTCPAddr("tcp", s)
		*/
		regex := `^(?:[a-zA-Z0-9]{1,62}(?:[-\.][a-zA-Z0-9]{1,62})+)(:\d+)?$`
		test := regexp.MustCompile(regex)

		if test.MatchString(s) {
			entryAddress.SetValidationError(nil)
			setDaemon(s)
		} else {
			err = errors.New("invalid host name")
			entryAddress.SetValidationError(err)
		}

		return
	}
	entryAddress.PlaceHolder = "0.0.0.0:10102"
	entryAddress.SetText(getDaemon())
	entryAddress.Refresh()

	selectNodes := widget.NewSelect(nil, nil)
	selectNodes.PlaceHolder = "Select Public Node ..."
	if session.Testnet {
		selectNodes.Options = []string{"testnetexplorer.dero.io:40402", "127.0.0.1:40402"}
	} else {
		selectNodes.Options = []string{"node.derofoundation.org:11012", "127.0.0.1:10102"}
	}
	selectNodes.OnChanged = func(s string) {
		if s != "" {
			err := setDaemon(s)
			if err == nil {
				entryAddress.Text = s
				entryAddress.Refresh()
			}
			selectNodes.ClearSelected()
		}
	}

	labelScan := widget.NewRichTextFromMarkdown("Enter the number of past blocks that the wallet should scan:")
	labelScan.Wrapping = fyne.TextWrapWord

	entryScan := widget.NewEntry()
	entryScan.PlaceHolder = "# of Latest Blocks (Optional)"
	entryScan.Validator = func(s string) (err error) {
		blocks, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			entryScan.SetValidationError(err)
		} else {
			entryScan.SetValidationError(nil)
			if blocks > 0 {
				session.TrackRecentBlocks = blocks
			} else {
				session.TrackRecentBlocks = 0
			}
		}

		return
	}

	if session.TrackRecentBlocks > 0 {
		blocks := strconv.FormatInt(session.TrackRecentBlocks, 10)
		entryScan.Text = blocks
		entryScan.Refresh()
	}

	radioNetwork := widget.NewRadioGroup([]string{"Mainnet", "Testnet"}, nil)
	radioNetwork.Horizontal = false
	radioNetwork.OnChanged = func(s string) {
		if s == "Testnet" {
			setTestnet(true)
			selectNodes.Options = []string{"testnetexplorer.dero.io:40402", "127.0.0.1:40402"}
		} else {
			setTestnet(false)
			selectNodes.Options = []string{"node.derofoundation.org:11012", "127.0.0.1:10102"}
		}

		selectNodes.Refresh()
	}

	net, _ := GetValue("settings", []byte("network"))

	if string(net) == "Testnet" {
		radioNetwork.SetSelected("Testnet")
	} else {
		radioNetwork.SetSelected("Mainnet")
	}

	radioNetwork.Refresh()

	entryUser := widget.NewEntry()
	entryUser.PlaceHolder = "Username"
	entryUser.SetText(cyberdeck.user)

	entryPass := widget.NewEntry()
	entryPass.PlaceHolder = "Password"
	entryPass.Password = true
	entryPass.SetText(cyberdeck.pass)

	entryUser.OnChanged = func(s string) {
		cyberdeck.user = s
	}

	entryPass.OnChanged = func(s string) {
		cyberdeck.pass = s
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

	labelBack := widget.NewHyperlinkWithStyle("Return to Login", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	labelBack.OnTapped = func() {
		if radioNetwork.Selected == "Testnet" {
			setTestnet(true)
		} else {
			setTestnet(false)
		}
		setDaemon(entryAddress.Text)

		initSettings()

		resizeWindow(ui.MaxWidth, ui.MaxHeight)
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutMain())
		removeOverlays()
	}

	btnRestore.OnTapped = func() {
		setTestnet(false)
		setDaemon(DEFAULT_REMOTE_DAEMON)
		setAuthMode("true")
		setGnomon("1")

		resizeWindow(ui.MaxWidth, ui.MaxHeight)
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutSettings())
		removeOverlays()
	}

	statusText := canvas.NewText("", colors.Account)
	statusText.TextSize = 12

	btnDelete.OnTapped = func() {
		err := cleanGnomonData()
		if err != nil {
			statusText.Color = colors.Red
			statusText.Text = err.Error()
			statusText.Refresh()
			return
		}

		statusText.Color = colors.Green
		statusText.Text = "Gnomon data successfully deleted."
		statusText.Refresh()
	}

	formSettings := container.NewVBox(
		labelNetwork,
		rectSpacer,
		radioNetwork,
		widget.NewLabel(""),
		labelNode,
		rectSpacer,
		rectSpacer,
		selectNodes,
		rectSpacer,
		entryAddress,
		rectSpacer,
		rectSpacer,
		labelScan,
		rectSpacer,
		entryScan,
		widget.NewLabel(""),
		labelSecurity,
		rectSpacer,
		textCyberdeck,
		rectSpacer,
		entryUser,
		rectSpacer,
		entryPass,
		rectSpacer,
		widget.NewLabel(""),
		labelGnomon,
		rectSpacer,
		textGnomon,
		rectSpacer,
		checkGnomon,
		rectSpacer,
		statusText,
		rectSpacer,
		rectSpacer,
		btnDelete,
		rectSpacer,
		btnRestore,
	)

	scrollBox = container.NewVScroll(
		container.NewHBox(
			layout.NewSpacer(),
			container.NewStack(
				rectScroll,
				formSettings,
			),
			layout.NewSpacer(),
		),
	)

	scrollBox.SetMinSize(fyne.NewSize(ui.MaxWidth, ui.Height*0.68))

	gridItem1 := container.NewCenter(
		container.NewVBox(
			widget.NewLabel(""),
			heading,
			widget.NewLabel(""),
			scrollBox,
			rectSpacer,
			rectSpacer,
		),
	)

	features := container.NewCenter(
		layout.NewSpacer(),
		gridItem1,
		layout.NewSpacer(),
	)

	footer := container.NewVBox(
		container.NewHBox(
			layout.NewSpacer(),
			labelBack,
			layout.NewSpacer(),
		),
		widget.NewLabel(" "),
	)

	c := container.NewBorder(
		features,
		footer,
		nil,
		nil,
	)

	layout := container.NewStack(
		frame,
		c,
	)

	return layout
}

func layoutMessages() fyne.CanvasObject {
	resetResources()
	session.Domain = "app.messages"

	if !walletapi.Connected {
		session.Window.SetContent(layoutSettings())
	}

	title := canvas.NewText("M E S S A G E S", colors.Gray)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 16

	heading := canvas.NewText("My Contacts", colors.Green)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	menuLabel := canvas.NewText("  M O R E   O P T I O N S  ", colors.Gray)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	checkLimit := widget.NewCheck(" Show only recent messages", nil)
	checkLimit.OnChanged = func(b bool) {
		if b {
			if int(engram.Disk.Get_Height()) > 1000000 {
				session.LimitMessages = uint64(int(engram.Disk.Get_Height()) - 1000000)
				session.Window.SetContent(layoutTransition())
				session.Window.SetContent(layoutMessages())
				removeOverlays()
			}
		} else {
			session.LimitMessages = 0
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutMessages())
			removeOverlays()
		}
	}

	if session.LimitMessages != uint64(0) {
		checkLimit.Checked = true
	}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkBack := widget.NewHyperlinkWithStyle("Back to Dashboard", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkBack.OnTapped = func() {
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutDashboard())
		removeOverlays()
	}

	rectStatus := canvas.NewRectangle(color.Transparent)
	rectStatus.SetMinSize(fyne.NewSize(10, 10))
	rectEmpty := canvas.NewRectangle(color.Transparent)
	rectEmpty.SetMinSize(fyne.NewSize(10, 10))
	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(fyne.NewSize(ui.Width, 20))
	frame := &iframe{}
	rect.SetMinSize(fyne.NewSize(ui.Width, 30))
	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(10, 5))
	rect.SetMinSize(fyne.NewSize(10, 10))
	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(fyne.NewSize(ui.Width, 35))
	rectListBox := canvas.NewRectangle(color.Transparent)
	rectListBox.SetMinSize(fyne.NewSize(ui.Width, ui.Height*0.40))

	messages.Data = nil

	data := getMessages(engram.Disk.Get_Height() - session.LimitMessages)
	temp := data

	list := binding.BindStringList(&data)

	messages.Box = widget.NewListWithData(list,
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
			dataItem := strings.Split(str, "~~~")
			short := dataItem[0]
			address := short[len(short)-10:]
			username := dataItem[1]

			if username == "" {
				co.(*fyne.Container).Objects[0].(*widget.Label).SetText("..." + address)
			} else {
				co.(*fyne.Container).Objects[0].(*widget.Label).SetText(username)
			}
			co.(*fyne.Container).Objects[0].(*widget.Label).Wrapping = fyne.TextWrapWord
			co.(*fyne.Container).Objects[0].(*widget.Label).TextStyle.Bold = false
			co.(*fyne.Container).Objects[0].(*widget.Label).Alignment = fyne.TextAlignLeading
		})

	messages.Box.OnSelected = func(id widget.ListItemID) {
		messages.Box.UnselectAll()
		split := strings.Split(data[id], "~~~")
		if split[1] == "" {
			messages.Contact = split[0]
		} else {
			messages.Contact = split[1]
		}

		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutPM())
		removeOverlays()
	}

	searchList := []string{}

	entrySearch := widget.NewEntry()
	entrySearch.PlaceHolder = "Search for a Contact"
	entrySearch.OnChanged = func(s string) {
		s = strings.ToLower(s)
		searchList = []string{}
		if s == "" {
			data = temp
			list.Reload()
		} else {
			for _, d := range temp {
				tempd := strings.ToLower(d)
				split := strings.Split(tempd, "~~~")

				if split[1] == "" {
					if strings.Contains(split[0], s) {
						searchList = append(searchList, d)
					}
				} else {
					if strings.Contains(split[1], s) {
						searchList = append(searchList, d)
					}
				}
			}

			data = searchList
			list.Reload()
		}
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
		removeOverlays()
	})
	btnSend.Disable()

	entryDest := widget.NewEntry()
	entryDest.MultiLine = false
	entryDest.Wrapping = fyne.TextWrap(fyne.TextTruncateClip)
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
		entryDest.OnFocusGained = func() {
			if fyne.CurrentDevice().IsMobile() {
				rectListBox.SetMinSize(fyne.NewSize(ui.Width, 170))
				rectListBox.Resize(fyne.NewSize(ui.Width, 170))
				rectListBox.Refresh()
				session.Window.Canvas().Content().Refresh()
			}
		}

		entryDest.OnFocusLost = func() {
			if fyne.CurrentDevice().IsMobile() {
				rectListBox.SetMinSize(fyne.NewSize(ui.Width, 270))
				rectListBox.Resize(fyne.NewSize(ui.Width, 270))
				rectListBox.Refresh()
				session.Window.Canvas().Content().Refresh()
			}
		}
	*/

	messageForm := container.NewVBox(
		rectSpacer,
		rectSpacer,
		container.NewHBox(
			layout.NewSpacer(),
			heading,
			layout.NewSpacer(),
		),
		rectSpacer,
		rectSpacer,
		entrySearch,
		rectSpacer,
		rectSpacer,
		container.NewStack(
			rectListBox,
			messages.Box,
		),
		rectSpacer,
		entryDest,
		rectSpacer,
		btnSend,
		rectSpacer,
		checkLimit,
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
		if session.Domain != "app.messages" {
			return
		}

		if k.Name == fyne.KeyUp {
			session.Dashboard = "main"

			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutDashboard())
			removeOverlays()
		} else if k.Name == fyne.KeyF5 {
			session.Window.SetContent(layoutMessages())
			removeOverlays()
		}
	})

	subContainer := container.NewStack(
		container.NewVBox(
			container.NewStack(
				container.NewHBox(
					layout.NewSpacer(),
					line1,
					layout.NewSpacer(),
					menuLabel,
					layout.NewSpacer(),
					line2,
					layout.NewSpacer(),
				),
			),
			rectSpacer,
			rectSpacer,
			container.NewCenter(
				layout.NewSpacer(),
				linkBack,
				layout.NewSpacer(),
			),
			rectSpacer,
			rectSpacer,
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

	layout := container.NewStack(
		frame,
		c,
	)

	return layout
}

func layoutPM() fyne.CanvasObject {
	resetResources()
	session.Domain = "app.messages.contact"

	if !walletapi.Connected {
		session.Window.SetContent(layoutSettings())
	}

	getPrimaryUsername()

	contactAddress := ""

	_, err := globals.ParseValidateAddress(messages.Contact)
	if err != nil {
		_, err := engram.Disk.NameToAddress(messages.Contact)
		if err == nil {
			contactAddress = messages.Contact
		}
	} else {
		short := messages.Contact[len(messages.Contact)-10:]
		contactAddress = "..." + short
	}

	//wSpacer := widget.NewLabel(" ")
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

	menuLabel := canvas.NewText("  M O R E   O P T I O N S  ", colors.Gray)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkBack := widget.NewHyperlinkWithStyle("Back to Messages", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkBack.OnTapped = func() {
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutMessages())
		removeOverlays()
	}

	rectStatus := canvas.NewRectangle(color.Transparent)
	rectStatus.SetMinSize(fyne.NewSize(10, 10))
	rectEmpty := canvas.NewRectangle(color.Transparent)
	rectEmpty.SetMinSize(fyne.NewSize(10, 10))
	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(fyne.NewSize(ui.Width*0.7, 30))
	frame := &iframe{}
	subframe := canvas.NewRectangle(color.Transparent)
	subframe.SetMinSize(fyne.NewSize(ui.Width, ui.Height*0.50))
	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(10, 5))
	rect.SetMinSize(fyne.NewSize(10, 10))
	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(fyne.NewSize(ui.Width, 35))
	rectListBox := canvas.NewRectangle(color.Transparent)
	rectListBox.SetMinSize(fyne.NewSize(ui.Width*0.42, 30))
	rectOutbound := canvas.NewRectangle(color.Transparent)
	rectOutbound.SetMinSize(fyne.NewSize(ui.Width*0.166, 30))

	messages.Data = nil

	chats := container.NewVBox()

	chatFrame := container.NewStack(
		rectListBox,
		container.NewStack(
			chats,
		),
	)

	chatbox := container.NewVScroll(
		container.NewStack(
			chatFrame,
		),
	)

	var e *fyne.Container

	data := getMessagesFromUser(messages.Contact, engram.Disk.Get_Height()-session.LimitMessages)
	for d := range data {
		if data[d].Incoming {
			if data[d].Payload_RPC.Has(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString) {
				if data[d].Payload_RPC.Value(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString).(string) == "" {

				} else {
					t := data[d].Time
					time := string(t.Format(time.RFC822))
					comment := data[d].Payload_RPC.Value(rpc.RPC_COMMENT, rpc.DataString).(string)
					links := getTextURL(comment)

					for i := range links {
						if comment == links[i] {
							if len(links[i]) > 25 {
								comment = `[ ` + links[i][0:25] + "..." + ` ](` + links[i] + `)`
							} else {
								comment = `[ ` + links[i] + ` ](` + links[i] + `)`
							}
						} else {
							linkText := ""
							split := strings.Split(comment, links[i])
							if len(links[i]) > 25 {
								linkText = links[i][0:25] + "..."
							} else {
								linkText = links[i]
							}
							comment = `` + split[0] + `[link]` + split[1] + "\n\n›" + `[ ` + linkText + ` ](` + links[i] + `)`
						}
					}
					messages.Data = append(messages.Data, data[d].Payload_RPC.Value(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString).(string)+";;;;"+comment+";;;;"+time)
				}
			}
		} else {
			t := data[d].Time
			time := string(t.Format(time.RFC822))
			comment := data[d].Payload_RPC.Value(rpc.RPC_COMMENT, rpc.DataString).(string)
			links := getTextURL(comment)

			for i := range links {
				if comment == links[i] {
					if len(links[i]) > 25 {
						comment = `[ ` + links[i][0:25] + "..." + ` ](` + links[i] + `)`
					} else {
						comment = `[ ` + links[i] + ` ](` + links[i] + `)`
					}
				} else {
					linkText := ""
					split := strings.Split(comment, links[i])
					if len(links[i]) > 25 {
						linkText = links[i][0:25] + "..."
					} else {
						linkText = links[i]
					}
					comment = `` + split[0] + `[link]` + split[1] + "\n\n›" + `[ ` + linkText + ` ](` + links[i] + `)`
				}
			}
			messages.Data = append(messages.Data, engram.Disk.GetAddress().String()+";;;;"+comment+";;;;"+time)
		}
	}

	if len(data) > 0 {
		for m := range messages.Data {
			var sender string
			split := strings.Split(messages.Data[m], ";;;;")
			mdata := widget.NewRichTextFromMarkdown("")
			mdata.Wrapping = fyne.TextWrapWord
			datetime := canvas.NewText("", colors.Green)
			datetime.TextSize = 11
			boxColor := colors.Flint
			rect := canvas.NewRectangle(boxColor)
			rect.SetMinSize(fyne.NewSize(ui.Width*0.80, 30))
			rect.CornerRadius = 5.0
			rect5 := canvas.NewRectangle(color.Transparent)
			rect5.SetMinSize(fyne.NewSize(5, 5))

			uname, err := engram.Disk.NameToAddress(split[0])
			if err != nil {
				sender = split[0]
			} else {
				sender = uname
			}

			if sender == engram.Disk.GetAddress().String() {
				rect.FillColor = colors.DarkGreen
				mdata.ParseMarkdown(split[1])
				datetime.Text = split[2]
				e = container.NewBorder(
					nil,
					container.NewVBox(
						container.NewHBox(
							layout.NewSpacer(),
							datetime,
							rect5,
						),
						rect5,
					),
					rectOutbound,
					container.NewStack(
						rect,
						container.NewVBox(
							mdata,
						),
					),
				)
			} else {
				rect.FillColor = colors.Flint
				mdata.ParseMarkdown(split[1])
				datetime.Text = split[2]
				e = container.NewBorder(
					nil,
					container.NewVBox(
						container.NewHBox(
							rect5,
							datetime,
							layout.NewSpacer(),
						),
						rect5,
					),
					container.NewStack(
						rect,
						container.NewVBox(
							mdata,
						),
					),
					rectOutbound,
				)
			}

			lastActive.Text = "Last Updated:  " + time.Now().Format(time.RFC822)
			lastActive.Refresh()

			chats.Add(e)
			chats.Refresh()
			chatbox.Refresh()
			chatbox.ScrollToBottom()
		}
	}

	btnSend := widget.NewButton("Send", nil)
	btnSend.Disable()

	entry := widget.NewEntry()
	entry.MultiLine = false
	entry.Wrapping = fyne.TextWrap(fyne.TextTruncateClip)
	entry.PlaceHolder = "Message"
	entry.OnChanged = func(s string) {
		messages.Message = s
		contact := messages.Contact
		check, err := engram.Disk.NameToAddress(messages.Contact)
		if err == nil {
			contact = check
		}

		_, err = globals.ParseValidateAddress(contact)
		if err != nil {
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutMessages())
			removeOverlays()
			return
		}

		err = checkMessagePack(messages.Message, session.Username, contact)
		if err != nil {
			btnSend.Text = "Message too long..."
			btnSend.Disable()
			btnSend.Refresh()
			return
		} else {
			if messages.Message == "" {
				btnSend.Text = "Send"
				btnSend.Disable()
				btnSend.Refresh()
			} else {
				btnSend.Text = "Send"
				btnSend.Enable()
				btnSend.Refresh()
			}
		}
	}

	/*
		entry.OnFocusGained = func() {
			if fyne.CurrentDevice().IsMobile() {
				subframe.SetMinSize(fyne.NewSize(ui.Width, 165))
				subframe.Resize(fyne.NewSize(ui.Width, 165))
				subframe.Refresh()
				session.Window.Canvas().Content().Refresh()
			}
			entry.CursorRow = 0
			entry.CursorColumn = 0
		}

		entry.OnFocusLost = func() {
			if fyne.CurrentDevice().IsMobile() {
				subframe.SetMinSize(fyne.NewSize(ui.Width, 270))
				subframe.Resize(fyne.NewSize(ui.Width, 270))
				subframe.Refresh()
				session.Window.Canvas().Content().Refresh()
			}
		}
	*/

	btnSend.OnTapped = func() {
		if messages.Message == "" {
			return
		}
		contact := ""
		_, err := globals.ParseValidateAddress(messages.Contact)
		if err != nil {
			check, err := engram.Disk.NameToAddress(messages.Contact)
			if err != nil {
				fmt.Printf("[Message] Failed to send: %s\n", err)
				btnSend.Text = "Failed to verify address..."
				btnSend.Disable()
				btnSend.Refresh()
				return
			}
			contact = check
		} else {
			contact = messages.Contact
		}

		txid, err := sendMessage(messages.Message, session.Username, contact)
		if err != nil {
			fmt.Printf("[Message] Failed to send: %s\n", err)
			btnSend.Text = "Failed to send message..."
			btnSend.Disable()
			btnSend.Refresh()
			return
		}

		fmt.Printf("[Message] Sent message successfully to: %s\n", messages.Contact)
		btnSend.Text = "Confirming..."
		btnSend.Disable()
		btnSend.Refresh()
		messages.Message = ""
		entry.Text = ""
		entry.Refresh()

		walletapi.WaitNewHeightBlock()
		for {
			result := engram.Disk.Get_Payments_TXID(txid.String())

			if result.TXID != txid.String() {
				time.Sleep(time.Second * 1)
			} else {
				break
			}
		}

		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutPM())
	}

	messageForm := container.NewVBox(
		rectSpacer,
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
		container.NewStack(
			subframe,
			chatbox,
		),
		rectSpacer,
		rectSpacer,
		entry,
		rectSpacer,
		btnSend,
		rectSpacer,
		rectSpacer,
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
		if session.Domain != "app.messages.contact" {
			return
		}

		if k.Name == fyne.KeyUp {
			session.Dashboard = "app.messages"
			messages.Contact = ""
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutMessages())
			removeOverlays()
		} else if k.Name == fyne.KeyEscape {
			session.Dashboard = "app.messages"
			messages.Contact = ""
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutMessages())
			removeOverlays()
		} else if k.Name == fyne.KeyF5 {
			session.Window.SetContent(layoutPM())
		}
	})

	subContainer := container.NewStack(
		container.NewVBox(
			container.NewStack(
				container.NewHBox(
					layout.NewSpacer(),
					line1,
					layout.NewSpacer(),
					menuLabel,
					layout.NewSpacer(),
					line2,
					layout.NewSpacer(),
				),
			),
			rectSpacer,
			rectSpacer,
			container.NewCenter(
				layout.NewSpacer(),
				linkBack,
				layout.NewSpacer(),
			),
			rectSpacer,
			rectSpacer,
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

	layout := container.NewStack(
		frame,
		c,
	)

	return layout
}

func layoutCyberdeck() fyne.CanvasObject {
	resetResources()
	session.Domain = "app.cyberdeck"
	wSpacer := widget.NewLabel(" ")
	title := canvas.NewText("C Y B E R D E C K", colors.Gray)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 16

	heading := canvas.NewText("My Contacts", colors.Green)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(fyne.NewSize(ui.Width, 20))
	frame := &iframe{}
	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(10, 5))

	menuLabel := canvas.NewText("  M O R E   O P T I O N S  ", colors.Gray)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkBack := widget.NewHyperlinkWithStyle("Back to Dashboard", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkBack.OnTapped = func() {
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutDashboard())
		removeOverlays()
	}

	shardText := canvas.NewText(session.Username, colors.Green)
	shardText.TextStyle = fyne.TextStyle{Bold: true}
	shardText.TextSize = 22

	shortShard := canvas.NewText("APPLICATION  CONNECTIONS", colors.Gray)
	shortShard.TextStyle = fyne.TextStyle{Bold: true}
	shortShard.TextSize = 12

	linkColor := colors.Green

	if cyberdeck.server == nil {
		session.Link = "Blocked"
		linkColor = colors.Gray
	}

	cyberdeck.status = canvas.NewText(session.Link, linkColor)
	cyberdeck.status.TextSize = 22
	cyberdeck.status.TextStyle = fyne.TextStyle{Bold: true}

	serverStatus := canvas.NewText("APPLICATION  CONNECTIONS", colors.Gray)
	serverStatus.TextSize = 12
	serverStatus.Alignment = fyne.TextAlignCenter
	serverStatus.TextStyle = fyne.TextStyle{Bold: true}

	linkCenter := container.NewCenter(
		cyberdeck.status,
	)

	cyberdeck.userText = widget.NewEntry()
	cyberdeck.userText.PlaceHolder = "Username"
	cyberdeck.userText.OnChanged = func(s string) {
		if len(s) > 1 {
			cyberdeck.user = s
		}
	}

	cyberdeck.passText = widget.NewEntry()
	cyberdeck.passText.Password = true
	cyberdeck.passText.PlaceHolder = "Password"
	cyberdeck.passText.OnChanged = func(s string) {
		if len(s) > 1 {
			cyberdeck.pass = s
		}
	}

	cyberdeck.toggle = widget.NewButton("Turn On", nil)
	cyberdeck.toggle.OnTapped = func() {
		toggleCyberdeck()
	}

	if session.Offline {
		cyberdeck.toggle.Text = "Disabled in Offline Mode"
		cyberdeck.toggle.Disable()
	} else {
		if cyberdeck.server != nil {
			cyberdeck.status.Text = "Allowed"
			cyberdeck.status.Color = colors.Green
			cyberdeck.toggle.Text = "Turn Off"
			cyberdeck.userText.Disable()
			cyberdeck.passText.Disable()
		} else {
			cyberdeck.status.Text = "Blocked"
			cyberdeck.status.Color = colors.Gray
			cyberdeck.toggle.Text = "Turn On"
			cyberdeck.userText.Enable()
			cyberdeck.passText.Enable()
		}
	}

	cyberdeck.userText.SetText(cyberdeck.user)
	cyberdeck.passText.SetText(cyberdeck.pass)

	linkCopy := widget.NewHyperlinkWithStyle("Copy Credentials", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkCopy.OnTapped = func() {
		session.Window.Clipboard().SetContent(cyberdeck.user + ":" + cyberdeck.pass)
	}

	deckForm := container.NewVBox(
		rect,
		rectSpacer,
		container.NewCenter(container.NewVBox(title, rectSpacer)),
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
		wSpacer,
		container.NewHBox(
			layout.NewSpacer(),
			linkCopy,
			layout.NewSpacer(),
		),
		wSpacer,
	)

	gridItem1 := container.NewCenter(
		deckForm,
	)

	gridItem1.Hidden = false

	features := container.NewCenter(
		layout.NewSpacer(),
		gridItem1,
		layout.NewSpacer(),
	)

	session.Window.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
		if k.Name == fyne.KeyLeft {
			session.Dashboard = "main"
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutDashboard())
			removeOverlays()
		}
	})

	subContainer := container.NewStack(
		container.NewVBox(
			container.NewStack(
				container.NewHBox(
					layout.NewSpacer(),
					line1,
					layout.NewSpacer(),
					menuLabel,
					layout.NewSpacer(),
					line2,
					layout.NewSpacer(),
				),
			),
			rectSpacer,
			rectSpacer,
			container.NewCenter(
				layout.NewSpacer(),
				linkBack,
				layout.NewSpacer(),
			),
			rectSpacer,
			rectSpacer,
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

	layout := container.NewStack(
		frame,
		c,
	)

	return layout
}

func layoutIdentity() fyne.CanvasObject {
	resetResources()
	session.Domain = "app.Identity"
	title := canvas.NewText("I D E N T I T Y", colors.Gray)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 16

	heading := canvas.NewText("My Contacts", colors.Green)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	menuLabel := canvas.NewText("  M O R E   O P T I O N S  ", colors.Gray)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	frame := &iframe{}

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(10, 5))
	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(fyne.NewSize(ui.Width, 35))
	rectListBox := canvas.NewRectangle(color.Transparent)
	rectListBox.SetMinSize(fyne.NewSize(ui.Width, ui.Height*0.44))

	shortShard := canvas.NewText("PRIMARY  USERNAME", colors.Gray)
	shortShard.TextStyle = fyne.TextStyle{Bold: true}
	shortShard.TextSize = 12

	idCenter := container.NewCenter(
		shortShard,
	)

	linkBack := widget.NewHyperlinkWithStyle("Back to Dashboard", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkBack.OnTapped = func() {
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutDashboard())
		removeOverlays()
	}

	//entryReg := NewMobileEntry()
	entryReg := widget.NewEntry()
	entryReg.MultiLine = false
	entryReg.Wrapping = fyne.TextWrap(fyne.TextTruncateClip)

	userData, err := queryUsernames()
	if err != nil {
		userData, err = getUsernames()
		if err != nil {
			userData = nil
		}
	}

	userList := binding.BindStringList(&userData)

	btnReg := widget.NewButton(" Register ", nil)
	btnReg.Disable()
	btnReg.OnTapped = func() {
		if len(session.NewUser) > 5 {
			valid, _, _ := checkUsername(session.NewUser, -1)
			if !valid {
				btnReg.Text = "Confirming..."
				btnReg.Disable()
				btnReg.Refresh()
				err := registerUsername(session.NewUser)
				if err != nil {
					btnReg.Text = "Unable to register..."
					btnReg.Refresh()
					fmt.Printf("[Username] %s\n", err)

				} else {
					go func() {
						entryReg.Text = ""
						entryReg.Refresh()
						walletapi.WaitNewHeightBlock()
						var loop bool
						for !loop {
							if session.Domain == "app.Identity" {
								//vars, _, _, err := gnomon.Index.RPC.GetSCVariables("0000000000000000000000000000000000000000000000000000000000000001", engram.Disk.Get_Daemon_TopoHeight(), nil, []string{session.NewUser}, nil, false)
								usernames, err := queryUsernames()
								if err != nil {
									fmt.Printf("[Username] Error querying usernames: %s\n", err)
									return
								}

								for u := range usernames {
									if usernames[u] == session.NewUser {
										fmt.Printf("[Username] Successfully registered username: %s\n", session.NewUser)
										_ = tx
										btnReg.Text = "Registration successful!"
										btnReg.Refresh()
										session.NewUser = ""
										loop = true
										session.Window.SetContent(layoutIdentity())
										break
									}
								}
							} else {
								loop = true
							}

							time.Sleep(time.Second * 1)
						}
					}()
				}
			}
		}
	}

	entryReg.PlaceHolder = "New Username"
	entryReg.Validator = func(s string) error {
		btnReg.Text = " Register "
		btnReg.Enable()
		btnReg.Refresh()
		session.NewUser = s
		if len(s) > 5 {
			valid, _, _ := checkUsername(s, -1)
			if !valid {
				btnReg.Enable()
				btnReg.Refresh()
			} else {
				btnReg.Disable()
				err := errors.New("Username already exists")
				entryReg.SetValidationError(err)
				btnReg.Refresh()
				return err
			}
		} else {
			btnReg.Disable()
			err := errors.New("Username too short, need a minimum of six characters")
			entryReg.SetValidationError(err)
			btnReg.Refresh()
			return err
		}

		return nil
	}
	entryReg.OnChanged = func(s string) {
		entryReg.Validate()
	}

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

	err = getPrimaryUsername()
	if err != nil {
		session.Username = ""
	}

	textUsername := canvas.NewText(session.Username, colors.Green)
	textUsername.TextStyle = fyne.TextStyle{Bold: true}
	textUsername.TextSize = 22

	if session.Username == "" {
		textUsername.Text = "---"
		textUsername.Refresh()
	} else {
		for u := range userData {
			if userData[u] == session.Username {
				userBox.Select(u)
				userBox.ScrollTo(u)
			}
		}
	}

	userBox.OnSelected = func(id widget.ListItemID) {
		overlay := session.Window.Canvas().Overlays()
		overlay.Add(
			container.NewStack(
				&iframe{},
				canvas.NewRectangle(colors.DarkMatter),
			),
		)
		overlay.Add(layoutIdentityDetail(userData[id]))
		userBox.UnselectAll()
	}

	shardForm := container.NewVBox(
		rectSpacer,
		rectSpacer,
		container.NewCenter(
			container.NewVBox(
				title,
				rectSpacer,
			),
		),
		rectSpacer,
		container.NewStack(
			container.NewCenter(
				textUsername,
			),
		),
		rectSpacer,
		idCenter,
		rectSpacer,
		rectSpacer,
		container.NewStack(
			rectListBox,
			userBox,
		),
		rectSpacer,
		entryReg,
		rectSpacer,
		btnReg,
		rectSpacer,
		rectSpacer,
		rectSpacer,
		rectSpacer,
	)

	gridItem1 := container.NewCenter(
		shardForm,
	)

	features := container.NewCenter(
		layout.NewSpacer(),
		gridItem1,
		layout.NewSpacer(),
	)

	session.Window.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
		if k.Name == fyne.KeyRight {
			session.Dashboard = "main"

			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutDashboard())
			removeOverlays()
		} else if k.Name == fyne.KeyF5 {
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutIdentity())
			removeOverlays()
		}
	})

	subContainer := container.NewStack(
		container.NewVBox(
			container.NewStack(
				container.NewHBox(
					layout.NewSpacer(),
					line1,
					layout.NewSpacer(),
					menuLabel,
					layout.NewSpacer(),
					line2,
					layout.NewSpacer(),
				),
			),
			rectSpacer,
			rectSpacer,
			container.NewCenter(
				layout.NewSpacer(),
				linkBack,
				layout.NewSpacer(),
			),
			rectSpacer,
			rectSpacer,
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

	layout := container.NewStack(
		frame,
		c,
	)

	return layout
}

func layoutIdentityDetail(username string) fyne.CanvasObject {
	var address string
	var valid bool
	resetResources()

	wSpacer := widget.NewLabel(" ")

	rectWidth := canvas.NewRectangle(color.Transparent)
	rectWidth.SetMinSize(fyne.NewSize(ui.MaxWidth*0.99, 10))

	rectWidth90 := canvas.NewRectangle(color.Transparent)
	rectWidth90.SetMinSize(fyne.NewSize(ui.Width, 10))

	frame := &iframe{}

	heading := canvas.NewText("I D E N T I T Y    D E T A I L", colors.Gray)
	heading.TextSize = 16
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(6, 5))

	labelUsername := canvas.NewText("REGISTERED  USERNAME", colors.Gray)
	labelUsername.TextSize = 11
	labelUsername.Alignment = fyne.TextAlignCenter
	labelUsername.TextStyle = fyne.TextStyle{Bold: true}

	labelTransfer := canvas.NewText("  T R A N S F E R  ", colors.Gray)
	labelTransfer.TextSize = 11
	labelTransfer.Alignment = fyne.TextAlignCenter
	labelTransfer.TextStyle = fyne.TextStyle{Bold: true}

	menuLabel := canvas.NewText("  M O R E   O P T I O N S  ", colors.Gray)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkBack := widget.NewHyperlinkWithStyle("Back to Identity", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkBack.OnTapped = func() {
		removeOverlays()
	}

	valueUsername := canvas.NewText(username, colors.Green)
	valueUsername.TextSize = 22
	valueUsername.TextStyle = fyne.TextStyle{Bold: true}
	valueUsername.Alignment = fyne.TextAlignCenter

	btnSetPrimary := widget.NewButton("Set Primary Username", nil)
	btnSetPrimary.OnTapped = func() {
		setPrimaryUsername(username)
		session.Username = username
		session.Window.SetContent(layoutIdentity())
		removeOverlays()
	}

	btnSend := widget.NewButton("Transfer Username", nil)

	inputAddress := widget.NewEntry()
	inputAddress.PlaceHolder = "Receiver Username or Address"
	inputAddress.Validator = func(s string) error {
		btnSend.Text = "Transfer Username"
		btnSend.Enable()
		btnSend.Refresh()
		valid, address, _ = checkUsername(s, -1)
		if !valid {
			_, err := globals.ParseValidateAddress(s)
			if err != nil {
				btnSend.Disable()
				btnSend.Refresh()
				err := errors.New("address does not exist")
				inputAddress.SetValidationError(err)
				inputAddress.Refresh()
				return err
			} else {
				btnSend.Enable()
				btnSend.Refresh()
				address = s
			}
		} else {
			btnSend.Enable()
			btnSend.Refresh()
		}

		return nil
	}

	btnSend.OnTapped = func() {
		if address != "" && address != engram.Disk.GetAddress().String() {
			btnSend.Text = "Setting up transfer..."
			btnSend.Disable()
			btnSend.Refresh()
			inputAddress.Disable()
			inputAddress.Refresh()
			err := transferUsername(username, address)
			if err != nil {
				address = ""
				btnSend.Text = "Transfer failed..."
				btnSend.Disable()
				btnSend.Refresh()
				inputAddress.Enable()
				inputAddress.Refresh()
			} else {
				btnSend.Text = "Confirming..."
				btnSend.Refresh()
				go func() {
					walletapi.WaitNewHeightBlock()
					for {
						found := false
						if session.Domain == "app.Identity" {
							usernames, err := queryUsernames()
							if err != nil {
								fmt.Printf("[Username] Error querying usernames: %s\n", err)
								return
							}

							for u := range usernames {
								if usernames[u] == username {
									found = true
								}
							}

							if !found {
								fmt.Printf("[TransferOwnership] %s was successfully transfered to: %s\n", username, address)
								session.Window.SetContent(layoutTransition())
								session.Window.SetContent(layoutIdentity())
								removeOverlays()
								break
							}

						} else {
							break
						}

						time.Sleep(time.Second * 1)
					}
				}()
			}
		}
	}

	top := container.NewVBox(
		rectSpacer,
		rectSpacer,
		container.NewCenter(
			heading,
		),
		rectSpacer,
		rectSpacer,
	)

	center := container.NewStack(
		container.NewVScroll(
			container.NewStack(
				rectWidth,
				container.NewHBox(
					layout.NewSpacer(),
					container.NewVBox(
						rectSpacer,
						valueUsername,
						rectSpacer,
						labelUsername,
						wSpacer,
						container.NewHBox(
							layout.NewSpacer(),
							container.NewStack(
								rectWidth90,
								btnSetPrimary,
							),
							layout.NewSpacer(),
						),
						wSpacer,
						container.NewStack(
							rectWidth,
							container.NewHBox(
								layout.NewSpacer(),
								line1,
								layout.NewSpacer(),
								labelTransfer,
								layout.NewSpacer(),
								line2,
								layout.NewSpacer(),
							),
						),
						wSpacer,
						container.NewHBox(
							layout.NewSpacer(),
							container.NewStack(
								rectWidth90,
								inputAddress,
							),
							layout.NewSpacer(),
						),
						rectSpacer,
						rectSpacer,
						container.NewHBox(
							layout.NewSpacer(),
							container.NewStack(
								rectWidth90,
								btnSend,
							),
							layout.NewSpacer(),
						),
					),
					layout.NewSpacer(),
				),
			),
		),
	)

	bottom := container.NewStack(
		container.NewVBox(
			rectSpacer,
			rectSpacer,
			container.NewStack(
				container.NewHBox(
					layout.NewSpacer(),
					line1,
					layout.NewSpacer(),
					menuLabel,
					layout.NewSpacer(),
					line2,
					layout.NewSpacer(),
				),
			),
			rectSpacer,
			rectSpacer,
			container.NewCenter(
				layout.NewSpacer(),
				linkBack,
				layout.NewSpacer(),
			),
			rectSpacer,
			rectSpacer,
			rectSpacer,
			rectSpacer,
		),
	)

	layout := container.NewStack(
		frame,
		container.NewBorder(
			top,
			bottom,
			nil,
			center,
		),
	)

	return layout
}

func layoutWaiting(title *canvas.Text, heading *canvas.Text, sub *canvas.Text, link *widget.Hyperlink) fyne.CanvasObject {
	resetResources()

	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(fyne.NewSize(ui.Width*0.6, ui.Height*0.35))
	rect2 := canvas.NewRectangle(color.Transparent)
	rect2.SetMinSize(fyne.NewSize(ui.Width, 1))
	frame := canvas.NewRectangle(color.Transparent)
	frame.SetMinSize(fyne.NewSize(ui.Width, ui.Height))
	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(10, 5))
	label := canvas.NewText("PROOF-OF-WORK", colors.Gray)
	label.TextStyle = fyne.TextStyle{Bold: true}
	label.TextSize = 12
	hashes := canvas.NewText(fmt.Sprintf("%d", session.RegHashes), colors.Account)
	hashes.TextSize = 18

	go func() {
		for engram.Disk != nil {
			hashes.Text = fmt.Sprintf("%d", session.RegHashes)
			hashes.Refresh()
		}
	}()

	session.Gif, _ = x.NewAnimatedGifFromResource(resourceAnimation2Gif)
	session.Gif.SetMinSize(rect.MinSize())
	session.Gif.Resize(rect.MinSize())
	session.Gif.Start()

	waitForm := container.NewVBox(
		widget.NewLabel(""),
		container.NewHBox(
			layout.NewSpacer(),
			title,
			layout.NewSpacer(),
		),
		widget.NewLabel(""),
		heading,
		rectSpacer,
		sub,
		widget.NewLabel(""),
		container.NewStack(
			session.Gif,
		),
		widget.NewLabel(""),
		container.NewHBox(
			layout.NewSpacer(),
			container.NewVBox(
				container.NewCenter(
					rect2,
					hashes,
				),
				rectSpacer,
				container.NewCenter(
					rect2,
					label,
				),
			),
			layout.NewSpacer(),
		),
	)

	grid := container.NewHBox(
		layout.NewSpacer(),
		waitForm,
		layout.NewSpacer(),
	)

	footer := container.NewVBox(
		container.NewHBox(
			layout.NewSpacer(),
			link,
			layout.NewSpacer(),
		),
		widget.NewLabel(""),
	)

	c := container.NewBorder(
		grid,
		footer,
		nil,
		nil,
	)

	layout := container.NewStack(
		frame,
		c,
	)

	return layout
}

func layoutAlert(t int) fyne.CanvasObject {
	resetResources()

	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(fyne.NewSize(ui.Width*0.6, ui.Width*0.35))
	frame := &iframe{}
	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(10, 5))
	wSpacer := widget.NewLabel(" ")

	title := canvas.NewText("", colors.Gray)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 16
	title.Alignment = fyne.TextAlignCenter

	heading := canvas.NewText("", colors.Red)
	heading.TextStyle = fyne.TextStyle{Bold: true}
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter

	sub := widget.NewRichTextFromMarkdown("")
	sub.Wrapping = fyne.TextWrapWord

	labelSettings := widget.NewHyperlinkWithStyle("Review Settings", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	if t == 1 {
		title.Text = "E  R  R  O  R"
		heading.Text = "Connection Failure"
		sub.ParseMarkdown("Connection to " + session.Daemon + " has failed. Please review your settings and try again.")
		labelSettings.Text = "Review Settings"
		labelSettings.OnTapped = func() {
			session.Window.SetContent(layoutSettings())
		}
	} else if t == 2 {
		title.Text = "E  R  R  O  R"
		heading.Text = "Write Failure"
		sub.ParseMarkdown("Could not write data to disk, please check to make sure Engram has the proper permissions.")
		labelSettings.Text = "Review Settings"
		labelSettings.OnTapped = func() {
			session.Window.SetContent(layoutMain())
		}
	} else {
		title.Text = "E R R O R"
		heading.Text = "ID-10T Error Protocol"
		sub.ParseMarkdown("System malfunction... Please... Find... Help...")
		labelSettings.Text = "Review Settings"
		labelSettings.OnTapped = func() {
			session.Window.SetContent(layoutSettings())
		}
	}

	rectHeader := canvas.NewRectangle(color.Transparent)
	rectHeader.SetMinSize(fyne.NewSize(ui.Width, 1))

	session.Gif, _ = x.NewAnimatedGifFromResource(resourceAnimation2Gif)
	session.Gif.SetMinSize(rect.MinSize())
	session.Gif.Start()

	alertForm := container.NewVBox(
		wSpacer,
		wSpacer,
		rectHeader,
		container.NewStack(
			rect,
			res.red_alert,
		),
		heading,
		rectSpacer,
		sub,
		widget.NewLabel(""),
	)

	footer := container.NewVBox(
		container.NewHBox(
			layout.NewSpacer(),
			labelSettings,
			layout.NewSpacer(),
		),
		wSpacer,
	)

	features := container.NewCenter(
		layout.NewSpacer(),
		alertForm,
		layout.NewSpacer(),
	)

	c := container.NewBorder(
		features,
		footer,
		nil,
		nil,
	)

	layout := container.NewStack(
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

	frame := &iframe{}
	rectWidth := canvas.NewRectangle(color.Transparent)
	rectWidth.SetMinSize(fyne.NewSize(ui.MaxWidth, 10))
	rectWidth90 := canvas.NewRectangle(color.Transparent)
	rectWidth90.SetMinSize(fyne.NewSize(ui.Width, 10))

	heading := canvas.NewText("H I S T O R Y", colors.Gray)
	heading.TextSize = 16
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(fyne.NewSize(ui.Width*0.3, 35))

	rectMid := canvas.NewRectangle(color.Transparent)
	rectMid.SetMinSize(fyne.NewSize(ui.Width*0.35, 35))

	results := canvas.NewText("", colors.Green)
	results.TextSize = 13

	listData = binding.BindStringList(&data)
	listBox = widget.NewListWithData(listData,
		func() fyne.CanvasObject {
			return container.NewHBox(
				container.NewStack(
					rect,
					widget.NewLabel(""),
				),
				container.NewStack(
					rectMid,
					widget.NewLabel(""),
				),
				container.NewStack(
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

			split := strings.Split(str, ";;;")

			co.(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*widget.Label).SetText(split[0])
			co.(*fyne.Container).Objects[1].(*fyne.Container).Objects[1].(*widget.Label).SetText(split[1])
			co.(*fyne.Container).Objects[2].(*fyne.Container).Objects[1].(*widget.Label).SetText(split[3])
		})

	menu := widget.NewSelect([]string{"Normal", "Coinbase", "Messages"}, nil)
	menu.PlaceHolder = "(Select Transaction Type)"

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(10, 5))
	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(fyne.NewSize(ui.Width, ui.Height*0.60))

	menuLabel := canvas.NewText("  M O R E   O P T I O N S  ", colors.Gray)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkBack := widget.NewHyperlinkWithStyle("Back to Dashboard", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkBack.OnTapped = func() {
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutDashboard())
		removeOverlays()
	}

	label := canvas.NewText(view, colors.Account)
	label.TextSize = 15
	label.TextStyle = fyne.TextStyle{Bold: true}

	menu.OnChanged = func(s string) {
		switch s {
		case "Normal":
			listBox.UnselectAll()
			results.Text = "  Scanning..."
			results.Refresh()
			count := 0
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
							//stamp = string(timefmt.Format(time.RFC822))
							stamp = timefmt.Format("2006-01-02")
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

							count += 1
							data = append(data, direction+";;;"+amount+";;;"+height+";;;"+stamp+";;;"+txid)
						}
					}

					results.Text = fmt.Sprintf("  Results:  %d", count)
					results.Refresh()

					listData.Set(data)

					listBox.OnSelected = func(id widget.ListItemID) {
						//var zeroscid crypto.Hash
						split := strings.Split(data[id], ";;;")
						result := engram.Disk.Get_Payments_TXID(split[4])

						if result.TXID == "" {
							label.Text = "---"
						} else {
							label.Text = result.TXID
						}
						label.Refresh()

						overlay := session.Window.Canvas().Overlays()
						overlay.Add(
							container.NewStack(
								&iframe{},
								canvas.NewRectangle(colors.DarkMatter),
							),
						)
						overlay.Add(layoutHistoryDetail(split[4]))
						listBox.UnselectAll()
					}
					listBox.Refresh()
					listBox.ScrollToBottom()
				}()
			} else {
				results.Text = fmt.Sprintf("  Results:  %d", count)
				results.Refresh()
			}
		case "Coinbase":
			listBox.UnselectAll()
			results.Text = "  Scanning..."
			results.Refresh()
			count := 0
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
							direction = "Network"
							timefmt := entries[e].Time
							stamp = timefmt.Format("2006-01-02")
							height = strconv.FormatUint(entries[e].Height, 10)
							amount := globals.FormatMoney(entries[e].Amount)
							txid = entries[e].TXID

							count += 1
							data = append(data, direction+";;;"+amount+";;;"+height+";;;"+stamp+";;;"+txid)
						}
					}

					results.Text = fmt.Sprintf("  Results:  %d", count)
					results.Refresh()

					listData.Set(data)

					listBox.OnSelected = func(id widget.ListItemID) {
						listBox.UnselectAll()
					}
					listBox.Refresh()
					listBox.ScrollToBottom()
				}()
			} else {
				results.Text = fmt.Sprintf("  Results:  %d", count)
				results.Refresh()
			}
		case "Messages":
			listBox.UnselectAll()
			results.Text = "  Scanning..."
			results.Refresh()
			count := 0
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
						//stamp = string(timefmt.Format(time.RFC822))
						stamp = timefmt.Format("2006-01-02")

						temp := entries[e].Incoming
						if !temp {
							direction = "Sent    "
						} else {
							direction = "Received"
						}
						if entries[e].Payload_RPC.HasValue(rpc.RPC_COMMENT, rpc.DataString) {
							contact := ""
							username := ""
							if entries[e].Payload_RPC.HasValue(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString) {
								contact = entries[e].Payload_RPC.Value(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString).(string)
								if len(contact) > 10 {
									username = contact[0:10] + ".."
								} else {
									username = contact
								}
							}

							comment = entries[e].Payload_RPC.Value(rpc.RPC_COMMENT, rpc.DataString).(string)
							if len(comment) > 10 {
								comment = comment[0:10] + ".."
							}

							txid = entries[e].TXID
							count += 1
							data = append(data, direction+";;;"+username+";;;"+comment+";;;"+stamp+";;;"+txid+";;;"+contact)
						}
					}

					results.Text = fmt.Sprintf("  Results:  %d", count)
					results.Refresh()

					listData.Set(data)

					listBox.OnSelected = func(id widget.ListItemID) {
						split := strings.Split(data[id], ";;;")
						overlay := session.Window.Canvas().Overlays()
						overlay.Add(
							container.NewStack(
								&iframe{},
								canvas.NewRectangle(colors.DarkMatter),
							),
						)
						overlay.Add(layoutHistoryDetail(split[4]))
						listBox.UnselectAll()
						listBox.Refresh()
					}

					listBox.Refresh()
					listBox.ScrollToBottom()
				}()
			} else {
				results.Text = fmt.Sprintf("  Results:  %d", count)
				results.Refresh()
			}
		default:

		}
	}

	center := container.NewStack(
		rectWidth,
		container.NewHBox(
			layout.NewSpacer(),
			container.NewVBox(
				menu,
				rectSpacer,
				results,
				rectSpacer,
				rectSpacer,
				container.NewStack(
					rectList,
					listBox,
				),
			),
			layout.NewSpacer(),
		),
	)

	top := container.NewVBox(
		rectSpacer,
		rectSpacer,
		container.NewCenter(
			heading,
		),
		rectSpacer,
		rectSpacer,
		container.NewCenter(
			center,
		),
	)

	bottom := container.NewStack(
		container.NewVBox(
			container.NewStack(
				container.NewHBox(
					layout.NewSpacer(),
					line1,
					layout.NewSpacer(),
					menuLabel,
					layout.NewSpacer(),
					line2,
					layout.NewSpacer(),
				),
			),
			rectSpacer,
			rectSpacer,
			container.NewCenter(
				layout.NewSpacer(),
				linkBack,
				layout.NewSpacer(),
			),
			rectSpacer,
			rectSpacer,
			rectSpacer,
			rectSpacer,
		),
	)

	layout := container.NewStack(
		frame,
		container.NewBorder(
			top,
			bottom,
			nil,
			nil,
		),
	)

	return layout
}

func layoutHistoryDetail(txid string) fyne.CanvasObject {
	resetResources()

	wSpacer := widget.NewLabel(" ")

	rectWidth := canvas.NewRectangle(color.Transparent)
	rectWidth.SetMinSize(fyne.NewSize(ui.MaxWidth*0.99, 10))

	rectWidth90 := canvas.NewRectangle(color.Transparent)
	rectWidth90.SetMinSize(fyne.NewSize(ui.Width, 10))

	frame := &iframe{}

	heading := canvas.NewText("T R A N S A C T I O N    D E T A I L", colors.Gray)
	heading.TextSize = 16
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(6, 5))

	labelTXID := canvas.NewText("TRANSACTION  ID", colors.Gray)
	labelTXID.TextSize = 11
	labelTXID.Alignment = fyne.TextAlignLeading
	labelTXID.TextStyle = fyne.TextStyle{Bold: true}

	labelAmount := canvas.NewText("AMOUNT", colors.Gray)
	labelAmount.TextSize = 11
	labelAmount.Alignment = fyne.TextAlignLeading
	labelAmount.TextStyle = fyne.TextStyle{Bold: true}

	labelDirection := canvas.NewText("PAYMENT  DIRECTION", colors.Gray)
	labelDirection.TextSize = 11
	labelDirection.Alignment = fyne.TextAlignLeading
	labelDirection.TextStyle = fyne.TextStyle{Bold: true}

	labelMember := canvas.NewText("", colors.Gray)
	labelMember.TextSize = 11
	labelMember.Alignment = fyne.TextAlignLeading
	labelMember.TextStyle = fyne.TextStyle{Bold: true}

	labelProof := canvas.NewText("TRANSACTION  PROOF", colors.Gray)
	labelProof.TextSize = 11
	labelProof.Alignment = fyne.TextAlignLeading
	labelProof.TextStyle = fyne.TextStyle{Bold: true}

	labelDestPort := canvas.NewText("DESTINATION  PORT", colors.Gray)
	labelDestPort.TextSize = 11
	labelDestPort.TextStyle = fyne.TextStyle{Bold: true}

	labelSourcePort := canvas.NewText("SOURCE  PORT", colors.Gray)
	labelSourcePort.TextSize = 11
	labelSourcePort.TextStyle = fyne.TextStyle{Bold: true}

	labelFees := canvas.NewText("TRANSACTION  FEES", colors.Gray)
	labelFees.TextSize = 11
	labelFees.TextStyle = fyne.TextStyle{Bold: true}

	labelPayload := canvas.NewText("PAYLOAD", colors.Gray)
	labelPayload.TextSize = 11
	labelPayload.TextStyle = fyne.TextStyle{Bold: true}

	labelHeight := canvas.NewText("BLOCK  HEIGHT", colors.Gray)
	labelHeight.TextSize = 11
	labelHeight.TextStyle = fyne.TextStyle{Bold: true}

	labelReply := canvas.NewText("REPLY  ADDRESS", colors.Gray)
	labelReply.TextSize = 11
	labelReply.TextStyle = fyne.TextStyle{Bold: true}

	menuLabel := canvas.NewText("  M O R E   O P T I O N S  ", colors.Gray)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	details := engram.Disk.Get_Payments_TXID(txid)

	stamp := string(details.Time.Format(time.RFC822))
	height := strconv.FormatUint(details.Height, 10)

	valueMember := widget.NewRichTextFromMarkdown(" ")
	valueMember.Wrapping = fyne.TextWrapBreak

	valueReply := widget.NewRichTextFromMarkdown("--")
	valueReply.Wrapping = fyne.TextWrapBreak

	if details.Payload_RPC.HasValue(rpc.RPC_REPLYBACK_ADDRESS, rpc.DataAddress) {
		address := details.Payload_RPC.Value(rpc.RPC_REPLYBACK_ADDRESS, rpc.DataAddress).(rpc.Address)
		valueReply.ParseMarkdown("" + address.String())
	}

	valuePayload := widget.NewRichTextFromMarkdown("--")
	valuePayload.Wrapping = fyne.TextWrapBreak

	if details.Payload_RPC.HasValue(rpc.RPC_COMMENT, rpc.DataString) {
		if details.Payload_RPC.Value(rpc.RPC_COMMENT, rpc.DataString).(string) != "" {
			valuePayload.ParseMarkdown("" + details.Payload_RPC.Value(rpc.RPC_COMMENT, rpc.DataString).(string))
		}
	}

	valueDirection := canvas.NewText("", colors.Account)
	valueDirection.TextSize = 22
	valueDirection.TextStyle = fyne.TextStyle{Bold: true}
	if details.Incoming {
		valueDirection.Text = " Received"
		labelMember.Text = "SENDER  ADDRESS"
		if details.Sender == "" || details.Sender == engram.Disk.GetAddress().String() {
			valueMember.ParseMarkdown("--")
		} else {
			valueMember.ParseMarkdown("" + details.Sender)
		}
	} else {
		valueDirection.Text = " Sent"
		labelMember.Text = "RECEIVER  ADDRESS"
		valueMember.ParseMarkdown("" + details.Destination)
	}

	valueTime := canvas.NewText(stamp, colors.Account)
	valueTime.TextSize = 14
	valueTime.TextStyle = fyne.TextStyle{Bold: true}

	valueFees := canvas.NewText(" "+globals.FormatMoney(details.Fees), colors.Account)
	valueFees.TextSize = 22
	valueFees.TextStyle = fyne.TextStyle{Bold: true}

	valueHeight := canvas.NewText(" "+height, colors.Account)
	valueHeight.TextSize = 22
	valueHeight.TextStyle = fyne.TextStyle{Bold: true}

	valueTXID := widget.NewRichTextFromMarkdown("")
	valueTXID.Wrapping = fyne.TextWrapBreak
	valueTXID.ParseMarkdown("" + txid)

	valueAmount := canvas.NewText("", colors.Account)
	valueAmount.TextSize = 22
	valueAmount.TextStyle = fyne.TextStyle{Bold: true}
	valueAmount.Text = " " + globals.FormatMoney(details.Amount)

	valuePort := canvas.NewText("", colors.Account)
	valuePort.TextSize = 22
	valuePort.TextStyle = fyne.TextStyle{Bold: true}
	valuePort.Text = " " + strconv.FormatUint(details.DestinationPort, 10)

	valueSourcePort := canvas.NewText("", colors.Account)
	valueSourcePort.TextSize = 22
	valueSourcePort.TextStyle = fyne.TextStyle{Bold: true}
	valueSourcePort.Text = " " + strconv.FormatUint(details.SourcePort, 10)

	btnView := widget.NewButton("View in Explorer", nil)
	btnView.OnTapped = func() {
		if engram.Disk.GetNetwork() {
			link, _ := url.Parse("https://explorer.dero.io/tx/" + txid)
			_ = fyne.CurrentApp().OpenURL(link)
		} else {
			link, _ := url.Parse("https://testnetexplorer.dero.io/tx/" + txid)
			_ = fyne.CurrentApp().OpenURL(link)
		}
	}

	linkBack := widget.NewHyperlinkWithStyle("Back to History", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkBack.OnTapped = func() {
		overlay := session.Window.Canvas().Overlays()
		overlay.Top().Hide()
		overlay.Remove(overlay.Top())
		overlay.Remove(overlay.Top())
	}

	linkAddress := widget.NewHyperlinkWithStyle("Copy Address", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkAddress.OnTapped = func() {
		session.Window.Clipboard().SetContent(valueMember.String())
	}

	linkReplyAddress := widget.NewHyperlinkWithStyle("Copy Address", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkReplyAddress.OnTapped = func() {
		if _, ok := details.Payload_RPC.Value(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString).(string); ok {
			session.Window.Clipboard().SetContent(details.Payload_RPC.Value(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString).(string))
		}
	}

	linkTXID := widget.NewHyperlinkWithStyle("Copy Transaction ID", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkTXID.OnTapped = func() {
		session.Window.Clipboard().SetContent(txid)
	}

	linkProof := widget.NewHyperlinkWithStyle("Copy Transaction Proof", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkProof.OnTapped = func() {
		session.Window.Clipboard().SetContent(details.Proof)
	}

	linkPayload := widget.NewHyperlinkWithStyle("Copy Payload", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkPayload.OnTapped = func() {
		if _, ok := details.Payload_RPC.Value(rpc.RPC_COMMENT, rpc.DataString).(string); ok {
			session.Window.Clipboard().SetContent(details.Payload_RPC.Value(rpc.RPC_COMMENT, rpc.DataString).(string))
		}
	}

	top := container.NewVBox(
		rectSpacer,
		rectSpacer,
		container.NewCenter(
			heading,
		),
		rectSpacer,
		container.NewCenter(
			valueTime,
		),
		rectSpacer,
		rectSpacer,
	)

	center := container.NewStack(
		container.NewVScroll(
			container.NewStack(
				rectWidth,
				container.NewHBox(
					layout.NewSpacer(),
					container.NewVBox(
						rectSpacer,
						labelDirection,
						rectSpacer,
						valueDirection,
						rectSpacer,
						rectSpacer,
						labelAmount,
						rectSpacer,
						container.NewStack(
							rectWidth90,
							valueAmount,
						),
						rectSpacer,
						rectSpacer,
						labelTXID,
						rectSpacer,
						container.NewStack(
							rectWidth90,
							valueTXID,
						),
						container.NewVBox(
							container.NewHBox(
								linkTXID,
								layout.NewSpacer(),
							),
							container.NewHBox(
								linkProof,
								layout.NewSpacer(),
							),
						),
						rectSpacer,
						rectSpacer,
						labelMember,
						rectSpacer,
						valueMember,
						container.NewHBox(
							linkAddress,
							layout.NewSpacer(),
						),
						rectSpacer,
						rectSpacer,
						labelReply,
						rectSpacer,
						valueReply,
						container.NewHBox(
							linkReplyAddress,
							layout.NewSpacer(),
						),
						rectSpacer,
						rectSpacer,
						labelHeight,
						rectSpacer,
						container.NewStack(
							rectWidth90,
							valueHeight,
						),
						rectSpacer,
						rectSpacer,
						labelFees,
						rectSpacer,
						container.NewStack(
							rectWidth90,
							valueFees,
						),
						rectSpacer,
						rectSpacer,
						labelPayload,
						rectSpacer,
						container.NewStack(
							rectWidth90,
							valuePayload,
						),
						container.NewVBox(
							container.NewHBox(
								linkPayload,
								layout.NewSpacer(),
							),
						),
						rectSpacer,
						rectSpacer,
						labelDestPort,
						rectSpacer,
						container.NewStack(
							rectWidth90,
							valuePort,
						),
						rectSpacer,
						rectSpacer,
						labelSourcePort,
						rectSpacer,
						container.NewStack(
							rectWidth90,
							valueSourcePort,
						),
						wSpacer,
						btnView,
						wSpacer,
					),
					layout.NewSpacer(),
				),
			),
		),
	)

	bottom := container.NewStack(
		container.NewVBox(
			rectSpacer,
			rectSpacer,
			container.NewStack(
				container.NewHBox(
					layout.NewSpacer(),
					line1,
					layout.NewSpacer(),
					menuLabel,
					layout.NewSpacer(),
					line2,
					layout.NewSpacer(),
				),
			),
			rectSpacer,
			rectSpacer,
			container.NewCenter(
				layout.NewSpacer(),
				linkBack,
				layout.NewSpacer(),
			),
			rectSpacer,
			rectSpacer,
			rectSpacer,
			rectSpacer,
		),
	)

	layout := container.NewStack(
		frame,
		container.NewBorder(
			top,
			bottom,
			nil,
			center,
		),
	)

	return layout
}

func layoutDatapad() fyne.CanvasObject {
	resetResources()
	session.Domain = "app.datapad"
	title := canvas.NewText("D A T A P A D", colors.Gray)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 16

	heading := canvas.NewText("", colors.Green)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	menuLabel := canvas.NewText("  M O R E   O P T I O N S  ", colors.Gray)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	entryNewPad := widget.NewEntry()
	entryNewPad.MultiLine = false
	entryNewPad.Wrapping = fyne.TextWrap(fyne.TextTruncateClip)

	btnAdd := widget.NewButton(" Create ", nil)
	btnAdd.Disable()
	btnAdd.OnTapped = func() {
		err := StoreEncryptedValue("Datapads", []byte(entryNewPad.Text), []byte(""))
		if err != nil {
			btnAdd.Text = "Error creating new Datapad"
			btnAdd.Disable()
			btnAdd.Refresh()
		} else {
			session.Datapad = entryNewPad.Text
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutDatapad())
			removeOverlays()
		}
	}

	entryNewPad.PlaceHolder = "Datapad Name"
	entryNewPad.Validator = func(s string) error {
		session.Datapad = s
		if len(s) > 0 {
			_, err := GetEncryptedValue("Datapads", []byte(s))
			if err == nil {
				btnAdd.Text = "Datapad already exists"
				btnAdd.Disable()
				btnAdd.Refresh()
				err := errors.New("Username already exists")
				entryNewPad.SetValidationError(err)
				return err
			} else {
				btnAdd.Text = "Create"
				btnAdd.Enable()
				btnAdd.Refresh()
				return nil
			}
		} else {
			btnAdd.Text = "Create"
			btnAdd.Disable()
			err := errors.New("Please enter a Datapad name")
			entryNewPad.SetValidationError(err)
			btnAdd.Refresh()
			return err
		}
	}
	entryNewPad.OnChanged = func(s string) {
		entryNewPad.Validate()
	}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkBack := widget.NewHyperlinkWithStyle("Back to Dashboard", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkBack.OnTapped = func() {
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutDashboard())
		removeOverlays()
	}

	frame := &iframe{}

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(10, 5))
	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(fyne.NewSize(ui.Width, 35))
	rectListBox := canvas.NewRectangle(color.Transparent)
	rectListBox.SetMinSize(fyne.NewSize(ui.Width, 350))

	var padData []string

	shard, err := GetShard()
	if err != nil {
		padData = []string{}
	}

	store, err := graviton.NewDiskStore(shard)
	if err != nil {
		padData = []string{}
	}

	ss, err := store.LoadSnapshot(0)

	if err != nil {
		padData = []string{}
	}

	tree, err := ss.GetTree("Datapads")
	if err != nil {
		padData = []string{}
	}

	cursor := tree.Cursor()

	for k, _, err := cursor.First(); err == nil; k, _, err = cursor.Next() {
		if string(k) != "" {
			padData = append(padData, string(k))
		}
	}

	padList := binding.BindStringList(&padData)

	padBox := widget.NewListWithData(padList,
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

	padBox.OnSelected = func(id widget.ListItemID) {
		session.Datapad = padData[id]
		overlay := session.Window.Canvas().Overlays()
		overlay.Add(
			container.NewStack(
				&iframe{},
				canvas.NewRectangle(colors.DarkMatter),
			),
		)
		overlay.Add(
			container.NewStack(
				&iframe{},
				layoutPad(),
			),
		)
		overlay.Top().Show()
		padBox.UnselectAll()
		padBox.Refresh()
	}

	shardForm := container.NewVBox(
		rectSpacer,
		rectSpacer,
		rectSpacer,
		container.NewCenter(container.NewVBox(title, rectSpacer)),
		rectSpacer,
		rectSpacer,
		container.NewStack(
			rectListBox,
			padBox,
		),
		rectSpacer,
		entryNewPad,
		rectSpacer,
		btnAdd,
		rectSpacer,
		rectSpacer,
		rectSpacer,
		rectSpacer,
	)

	gridItem1 := container.NewCenter(
		shardForm,
	)

	features := container.NewCenter(
		layout.NewSpacer(),
		gridItem1,
		layout.NewSpacer(),
	)

	subContainer := container.NewStack(
		container.NewVBox(
			container.NewStack(
				container.NewHBox(
					layout.NewSpacer(),
					line1,
					layout.NewSpacer(),
					menuLabel,
					layout.NewSpacer(),
					line2,
					layout.NewSpacer(),
				),
			),
			rectSpacer,
			rectSpacer,
			container.NewCenter(
				layout.NewSpacer(),
				linkBack,
				layout.NewSpacer(),
			),
			rectSpacer,
			rectSpacer,
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

	layout := container.NewStack(
		frame,
		c,
	)

	return layout
}

func layoutPad() fyne.CanvasObject {
	resetResources()

	rectWidth := canvas.NewRectangle(color.Transparent)
	rectWidth.SetMinSize(fyne.NewSize(ui.MaxWidth, 10))

	rectWidth90 := canvas.NewRectangle(color.Transparent)
	rectWidth90.SetMinSize(fyne.NewSize(ui.Width, 10))

	rectEntry := canvas.NewRectangle(color.Transparent)
	rectEntry.SetMinSize(fyne.NewSize(ui.Width, ui.Height*0.56))

	heading := canvas.NewText(session.Datapad, colors.Green)
	heading.TextSize = 20
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(6, 5))

	menuLabel := canvas.NewText("  M O R E   O P T I O N S  ", colors.Gray)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	selectOptions := widget.NewSelect([]string{"Clear", "Export (Plaintext)", "Delete"}, nil)
	selectOptions.PlaceHolder = "Select an Option ..."

	data, err := GetEncryptedValue("Datapads", []byte(session.Datapad))
	if err != nil {
		data = nil
	}

	overlay := session.Window.Canvas().Overlays()

	btnSave := widget.NewButton("Save", nil)

	entryPad := widget.NewEntry()
	entryPad.Wrapping = fyne.TextWrapWord

	selectOptions.OnChanged = func(s string) {
		if s == "Clear" {
			header := canvas.NewText("DATAPAD  RESET  REQUESTED", colors.Gray)
			header.TextSize = 14
			header.Alignment = fyne.TextAlignCenter
			header.TextStyle = fyne.TextStyle{Bold: true}

			subHeader := canvas.NewText("Clear Datapad?", colors.Account)
			subHeader.TextSize = 22
			subHeader.Alignment = fyne.TextAlignCenter
			subHeader.TextStyle = fyne.TextStyle{Bold: true}

			linkClose := widget.NewHyperlinkWithStyle("Cancel", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
			linkClose.OnTapped = func() {
				overlay := session.Window.Canvas().Overlays()
				overlay.Top().Hide()
				overlay.Remove(overlay.Top())
				overlay.Remove(overlay.Top())
				selectOptions.Selected = "Select an Option ..."
				selectOptions.Refresh()
			}

			btnSubmit := widget.NewButton("Clear", nil)

			btnSubmit.OnTapped = func() {
				if session.Datapad != "" {
					err := StoreEncryptedValue("Datapads", []byte(session.Datapad), []byte(""))
					if err != nil {
						fmt.Printf("[Datapad] Err: %s\n", err)
						selectOptions.Selected = "Select an Option ..."
						selectOptions.Refresh()
						return
					}

					selectOptions.Selected = "Select an Option ..."
					selectOptions.Refresh()
					entryPad.Text = ""
					entryPad.Refresh()
				}

				overlay := session.Window.Canvas().Overlays()
				overlay.Top().Hide()
				overlay.Remove(overlay.Top())
				overlay.Remove(overlay.Top())
				selectOptions.Selected = "Select an Option ..."
				selectOptions.Refresh()
			}

			span := canvas.NewRectangle(color.Transparent)
			span.SetMinSize(fyne.NewSize(ui.Width, 10))

			overlay.Add(
				container.NewStack(
					&iframe{},
					canvas.NewRectangle(colors.DarkMatter),
				),
			)

			overlay.Add(
				container.NewStack(
					&iframe{},
					container.NewCenter(
						container.NewVBox(
							span,
							container.NewCenter(
								header,
							),
							rectSpacer,
							rectSpacer,
							subHeader,
							widget.NewLabel(""),
							btnSubmit,
							rectSpacer,
							rectSpacer,
							container.NewHBox(
								layout.NewSpacer(),
								linkClose,
								layout.NewSpacer(),
							),
							rectSpacer,
							rectSpacer,
						),
					),
				),
			)
		} else if s == "Export (Plaintext)" {
			data := []byte(entryPad.Text)
			err := os.WriteFile(AppPath()+string(filepath.Separator)+session.Datapad+".txt", data, 0644)
			if err != nil {
				fmt.Printf("[Datapad] Err: %s\n", err)
				selectOptions.Selected = "Select an Option ..."
				selectOptions.Refresh()
				return
			}

			selectOptions.Selected = "Select an Option ..."
			selectOptions.Refresh()
		} else if s == "Delete" {
			header := canvas.NewText("DATAPAD  DELETION  REQUESTED", colors.Gray)
			header.TextSize = 14
			header.Alignment = fyne.TextAlignCenter
			header.TextStyle = fyne.TextStyle{Bold: true}

			subHeader := canvas.NewText("Delete Datapad?", colors.Account)
			subHeader.TextSize = 22
			subHeader.Alignment = fyne.TextAlignCenter
			subHeader.TextStyle = fyne.TextStyle{Bold: true}

			linkClose := widget.NewHyperlinkWithStyle("Cancel", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
			linkClose.OnTapped = func() {
				overlay := session.Window.Canvas().Overlays()
				overlay.Top().Hide()
				overlay.Remove(overlay.Top())
				overlay.Remove(overlay.Top())
				selectOptions.Selected = "Select an Option ..."
				selectOptions.Refresh()
			}

			btnSubmit := widget.NewButton("Delete", nil)

			btnSubmit.OnTapped = func() {
				if session.Datapad != "" {
					err := DeleteKey("Datapads", []byte(session.Datapad))
					if err != nil {
						fmt.Printf("[Datapad] Err: %s\n", err)
						selectOptions.Selected = "Select an Option ..."
						selectOptions.Refresh()
						fmt.Printf("[Datapad] Error deleting %s: %s\n", session.Datapad, err)
					} else {
						session.Datapad = ""
						session.DatapadChanged = false
						removeOverlays()
						session.Window.SetContent(layoutTransition())
						session.Window.SetContent(layoutDatapad())
					}
				}
			}

			span := canvas.NewRectangle(color.Transparent)
			span.SetMinSize(fyne.NewSize(ui.Width, 10))

			overlay.Add(
				container.NewStack(
					&iframe{},
					canvas.NewRectangle(colors.DarkMatter),
				),
			)

			overlay.Add(
				container.NewStack(
					&iframe{},
					container.NewCenter(
						container.NewVBox(
							span,
							container.NewCenter(
								header,
							),
							rectSpacer,
							rectSpacer,
							subHeader,
							widget.NewLabel(""),
							btnSubmit,
							rectSpacer,
							rectSpacer,
							container.NewHBox(
								layout.NewSpacer(),
								linkClose,
								layout.NewSpacer(),
							),
							rectSpacer,
							rectSpacer,
						),
					),
				),
			)
		} else {
			session.Datapad = ""
			session.DatapadChanged = false
			overlay := session.Window.Canvas().Overlays()
			overlay.Top().Hide()
			overlay.Remove(overlay.Top())
			overlay.Remove(overlay.Top())
			selectOptions.Selected = "Select an Option ..."
			selectOptions.Refresh()
		}
	}

	btnSave.OnTapped = func() {
		err = StoreEncryptedValue("Datapads", []byte(session.Datapad), []byte(entryPad.Text))
		if err != nil {
			btnSave.Text = "Error saving Datapad"
			btnSave.Disable()
			btnSave.Refresh()
		} else {
			session.DatapadChanged = false
			btnSave.Text = "Save"
			btnSave.Disable()
			heading.Text = session.Datapad
			heading.Refresh()
		}
	}

	session.DatapadChanged = false

	btnSave.Text = "Save"
	btnSave.Disable()

	entryPad.MultiLine = true
	entryPad.Text = string(data)
	entryPad.OnChanged = func(s string) {
		session.DatapadChanged = true
		heading.Text = session.Datapad + "*"
		heading.Refresh()
		btnSave.Text = "Save"
		btnSave.Enable()
	}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkBack := widget.NewHyperlinkWithStyle("Back to Datapad", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkBack.OnTapped = func() {
		if session.DatapadChanged {
			header := canvas.NewText("DATAPAD  CHANGE  DETECTED", colors.Gray)
			header.TextSize = 14
			header.Alignment = fyne.TextAlignCenter
			header.TextStyle = fyne.TextStyle{Bold: true}

			subHeader := canvas.NewText("Save Datapad?", colors.Account)
			subHeader.TextSize = 22
			subHeader.Alignment = fyne.TextAlignCenter
			subHeader.TextStyle = fyne.TextStyle{Bold: true}

			linkClose := widget.NewHyperlinkWithStyle("Discard Changes", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
			linkClose.OnTapped = func() {
				session.Datapad = ""
				session.DatapadChanged = false
				removeOverlays()
			}

			btnSubmit := widget.NewButton("Save", nil)

			btnSubmit.OnTapped = func() {
				err = StoreEncryptedValue("Datapads", []byte(session.Datapad), []byte(entryPad.Text))
				if err != nil {
					btnSave.Text = "Error saving Datapad"
					btnSave.Disable()
					btnSave.Refresh()
				} else {
					session.Datapad = ""
					session.DatapadChanged = false
					btnSave.Text = "Save"
					removeOverlays()
				}
			}

			span := canvas.NewRectangle(color.Transparent)
			span.SetMinSize(fyne.NewSize(ui.Width, 10))

			overlay.Add(
				container.NewStack(
					&iframe{},
					canvas.NewRectangle(colors.DarkMatter),
				),
			)

			overlay.Add(
				container.NewStack(
					&iframe{},
					container.NewCenter(
						container.NewVBox(
							span,
							container.NewCenter(
								header,
							),
							rectSpacer,
							rectSpacer,
							subHeader,
							widget.NewLabel(""),
							btnSubmit,
							rectSpacer,
							rectSpacer,
							container.NewHBox(
								layout.NewSpacer(),
								linkClose,
								layout.NewSpacer(),
							),
							rectSpacer,
							rectSpacer,
						),
					),
				),
			)
		} else {
			session.Datapad = ""
			session.DatapadChanged = false
			overlay := session.Window.Canvas().Overlays()
			overlay.Top().Hide()
			overlay.Remove(overlay.Top())
			overlay.Remove(overlay.Top())
		}
	}

	top := container.NewVBox(
		rectSpacer,
		rectSpacer,
		container.NewCenter(
			heading,
		),
		rectSpacer,
		container.NewCenter(
			container.NewStack(
				rectWidth90,
				selectOptions,
			),
		),
		rectSpacer,
	)

	center := container.NewStack(
		rectWidth,
		container.NewCenter(
			container.NewVBox(
				container.NewStack(
					rectEntry,
					entryPad,
				),
				rectSpacer,
				btnSave,
				rectSpacer,
			),
		),
	)

	bottom := container.NewStack(
		container.NewVBox(
			rectSpacer,
			rectSpacer,
			container.NewStack(
				container.NewHBox(
					layout.NewSpacer(),
					line1,
					layout.NewSpacer(),
					menuLabel,
					layout.NewSpacer(),
					line2,
					layout.NewSpacer(),
				),
			),
			rectSpacer,
			rectSpacer,
			container.NewCenter(
				layout.NewSpacer(),
				linkBack,
				layout.NewSpacer(),
			),
			rectSpacer,
			rectSpacer,
			rectSpacer,
			rectSpacer,
		),
	)

	layout := container.NewBorder(
		top,
		bottom,
		nil,
		center,
	)

	return layout
}

func layoutAccount() fyne.CanvasObject {
	resetResources()

	rectWidth := canvas.NewRectangle(color.Transparent)
	rectWidth.SetMinSize(fyne.NewSize(ui.MaxWidth*0.99, 10))
	rectWidth90 := canvas.NewRectangle(color.Transparent)
	rectWidth90.SetMinSize(fyne.NewSize(ui.Width, 10))
	rectBox := canvas.NewRectangle(color.Transparent)
	rectBox.SetMinSize(fyne.NewSize(ui.MaxWidth*0.99, ui.MaxHeight*0.80))

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(10, 5))

	title := canvas.NewText("M Y    A C C O U N T", colors.Gray)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 16

	heading := canvas.NewText(engram.Disk.GetAddress().String()[0:5]+"..."+engram.Disk.GetAddress().String()[len(engram.Disk.GetAddress().String())-10:len(engram.Disk.GetAddress().String())], colors.Green)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	labelPassword := canvas.NewText("N E W    P A S S W O R D", colors.Gray)
	labelPassword.TextStyle = fyne.TextStyle{Bold: true}
	labelPassword.TextSize = 11
	labelPassword.Alignment = fyne.TextAlignCenter

	menuLabel := canvas.NewText("  M O R E    O P T I O N S  ", colors.Gray)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	labelRecovery := canvas.NewText("A C C O U N T    R E C O V E R Y", colors.Gray)
	labelRecovery.TextSize = 11
	labelRecovery.Alignment = fyne.TextAlignCenter
	labelRecovery.TextStyle = fyne.TextStyle{Bold: true}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkBack := widget.NewHyperlinkWithStyle("Back to Dashboard", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkBack.OnTapped = func() {
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutDashboard())
		removeOverlays()
	}

	linkCopyAddress := widget.NewHyperlinkWithStyle("Copy Address", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkCopyAddress.OnTapped = func() {
		session.Window.Clipboard().SetContent(engram.Disk.GetAddress().String())
	}

	btnClear := widget.NewButton("Delete Datashard", nil)
	btnClear.OnTapped = func() {
		header := canvas.NewText("DATASHARD  DELETION  REQUESTED", colors.Gray)
		header.TextSize = 14
		header.Alignment = fyne.TextAlignCenter
		header.TextStyle = fyne.TextStyle{Bold: true}

		subHeader := canvas.NewText("Are you sure?", colors.Account)
		subHeader.TextSize = 22
		subHeader.Alignment = fyne.TextAlignCenter
		subHeader.TextStyle = fyne.TextStyle{Bold: true}

		linkClose := widget.NewHyperlinkWithStyle("Cancel", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		linkClose.OnTapped = func() {
			session.Datapad = ""
			session.DatapadChanged = false
			removeOverlays()
		}

		btnSubmit := widget.NewButton("Delete Datashard", nil)

		btnSubmit.OnTapped = func() {
			err := cleanWalletData()
			if err != nil {
				btnSubmit.Text = "Error deleting datashard"
				btnSubmit.Disable()
				btnSubmit.Refresh()
			} else {
				btnSubmit.Text = "Deletion successful!"
				btnSubmit.Disable()
				btnSubmit.Refresh()
				removeOverlays()
			}
		}

		span := canvas.NewRectangle(color.Transparent)
		span.SetMinSize(fyne.NewSize(ui.Width, 10))

		overlay := session.Window.Canvas().Overlays()

		overlay.Add(
			container.NewStack(
				&iframe{},
				canvas.NewRectangle(colors.DarkMatter),
			),
		)

		overlay.Add(
			container.NewStack(
				&iframe{},
				container.NewCenter(
					container.NewVBox(
						span,
						container.NewCenter(
							header,
						),
						rectSpacer,
						rectSpacer,
						subHeader,
						widget.NewLabel(""),
						btnSubmit,
						rectSpacer,
						rectSpacer,
						container.NewHBox(
							layout.NewSpacer(),
							linkClose,
							layout.NewSpacer(),
						),
						rectSpacer,
						rectSpacer,
					),
				),
			),
		)
	}

	btnSeed := widget.NewButton("Access Recovery Words", nil)
	btnSeed.OnTapped = func() {
		overlay := session.Window.Canvas().Overlays()

		header := canvas.NewText("ACCOUNT  VERIFICATION  REQUIRED", colors.Gray)
		header.TextSize = 14
		header.Alignment = fyne.TextAlignCenter
		header.TextStyle = fyne.TextStyle{Bold: true}

		subHeader := canvas.NewText("Confirm Password", colors.Account)
		subHeader.TextSize = 22
		subHeader.Alignment = fyne.TextAlignCenter
		subHeader.TextStyle = fyne.TextStyle{Bold: true}

		linkClose := widget.NewHyperlinkWithStyle("Cancel", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		linkClose.OnTapped = func() {
			overlay := session.Window.Canvas().Overlays()
			overlay.Top().Hide()
			overlay.Remove(overlay.Top())
			overlay.Remove(overlay.Top())
		}

		btnSubmit := widget.NewButton("Submit", nil)

		entryPassword := widget.NewEntry()
		entryPassword.Password = true
		entryPassword.PlaceHolder = "Password"
		entryPassword.OnChanged = func(s string) {
			if s == "" {
				btnSubmit.Text = "Submit"
				btnSubmit.Disable()
				btnSubmit.Refresh()
			} else {
				btnSubmit.Text = "Submit"
				btnSubmit.Enable()
				btnSubmit.Refresh()
			}
		}

		btnSubmit.OnTapped = func() {
			if engram.Disk.Check_Password(entryPassword.Text) {
				overlay.Add(
					container.NewStack(
						&iframe{},
						canvas.NewRectangle(colors.DarkMatter),
					),
				)

				overlay.Add(
					layoutRecovery(),
				)
			} else {
				btnSubmit.Text = "Invalid Password..."
				btnSubmit.Disable()
				btnSubmit.Refresh()
			}
		}

		btnSubmit.Disable()

		span := canvas.NewRectangle(color.Transparent)
		span.SetMinSize(fyne.NewSize(ui.Width, 10))

		overlay.Add(
			container.NewStack(
				&iframe{},
				canvas.NewRectangle(colors.DarkMatter),
			),
		)

		overlay.Add(
			container.NewStack(
				&iframe{},
				container.NewCenter(
					container.NewVBox(
						span,
						container.NewCenter(
							header,
						),
						rectSpacer,
						rectSpacer,
						subHeader,
						widget.NewLabel(""),
						container.NewCenter(
							container.NewStack(
								span,
								entryPassword,
							),
						),
						rectSpacer,
						rectSpacer,
						btnSubmit,
						rectSpacer,
						rectSpacer,
						container.NewHBox(
							layout.NewSpacer(),
							linkClose,
							layout.NewSpacer(),
						),
						rectSpacer,
						rectSpacer,
					),
				),
			),
		)
	}

	btnChange := widget.NewButton("Submit", nil)
	btnChange.Disable()

	curPass := widget.NewEntry()
	curPass.Password = true
	curPass.PlaceHolder = "Current Password"
	curPass.OnChanged = func(s string) {
		btnChange.Text = "Submit"
		btnChange.Enable()
		btnChange.Refresh()
	}

	newPass := widget.NewEntry()
	newPass.Password = true
	newPass.PlaceHolder = "New Password"
	newPass.OnChanged = func(s string) {
		btnChange.Text = "Submit"
		btnChange.Enable()
		btnChange.Refresh()
	}

	confirm := widget.NewEntry()
	confirm.Password = true
	confirm.PlaceHolder = "Confirm Password"
	confirm.OnChanged = func(s string) {
		btnChange.Text = "Submit"
		btnChange.Enable()
		btnChange.Refresh()
	}

	btnChange.OnTapped = func() {
		if engram.Disk.Check_Password(curPass.Text) {
			if newPass.Text == confirm.Text && newPass.Text != "" {
				err := engram.Disk.Set_Encrypted_Wallet_Password(newPass.Text)
				if err != nil {
					btnChange.Text = "Error changing password"
					btnChange.Disable()
					btnChange.Refresh()
				} else {
					curPass.Text = ""
					curPass.Refresh()
					newPass.Text = ""
					newPass.Refresh()
					confirm.Text = ""
					confirm.Refresh()
					btnChange.Text = "Password Updated"
					btnChange.Disable()
					btnChange.Refresh()
					engram.Disk.Save_Wallet()
				}
			} else {
				btnChange.Text = "Passwords do not match"
				btnChange.Disable()
				btnChange.Refresh()
			}
		} else {
			btnChange.Text = "Incorrect password entered"
			btnChange.Disable()
			btnChange.Refresh()
		}
	}

	form := container.NewStack(
		rectWidth,
		container.NewHBox(
			layout.NewSpacer(),
			container.NewVBox(
				curPass,
				rectWidth90,
				newPass,
				confirm,
				rectSpacer,
				btnChange,
			),
			layout.NewSpacer(),
		),
	)

	features := container.NewStack(
		rectBox,
		container.NewVScroll(
			container.NewVBox(
				rectSpacer,
				rectSpacer,
				container.NewCenter(
					container.NewVBox(
						title,
						rectSpacer,
					),
				),
				rectSpacer,
				heading,
				container.NewHBox(
					layout.NewSpacer(),
					linkCopyAddress,
					layout.NewSpacer(),
				),
				rectSpacer,
				rectSpacer,
				container.NewStack(
					container.NewHBox(
						layout.NewSpacer(),
						container.NewStack(
							rectWidth90,
							btnClear,
						),
						layout.NewSpacer(),
					),
				),
				widget.NewLabel(""),
				container.NewStack(
					container.NewHBox(
						layout.NewSpacer(),
						line1,
						layout.NewSpacer(),
						labelRecovery,
						layout.NewSpacer(),
						line2,
						layout.NewSpacer(),
					),
				),
				widget.NewLabel(""),
				container.NewStack(
					container.NewHBox(
						layout.NewSpacer(),
						container.NewStack(
							rectWidth90,
							btnSeed,
						),
						layout.NewSpacer(),
					),
				),
				widget.NewLabel(""),
				container.NewStack(
					container.NewHBox(
						layout.NewSpacer(),
						line1,
						layout.NewSpacer(),
						labelPassword,
						layout.NewSpacer(),
						line2,
						layout.NewSpacer(),
					),
				),
				widget.NewLabel(""),
				form,
			),
		),
	)

	bottom := container.NewStack(
		container.NewVBox(
			container.NewStack(
				container.NewHBox(
					layout.NewSpacer(),
					line1,
					layout.NewSpacer(),
					menuLabel,
					layout.NewSpacer(),
					line2,
					layout.NewSpacer(),
				),
			),
			rectSpacer,
			rectSpacer,
			container.NewCenter(
				layout.NewSpacer(),
				linkBack,
				layout.NewSpacer(),
			),
			rectSpacer,
			rectSpacer,
			rectSpacer,
			rectSpacer,
		),
	)

	layout := container.NewBorder(
		features,
		bottom,
		nil,
		nil,
	)

	return layout
}

func layoutRecovery() fyne.CanvasObject {
	wSpacer := widget.NewLabel(" ")
	heading := canvas.NewText("Recovery Words", colors.Green)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	rectStatus := canvas.NewRectangle(color.Transparent)
	rectStatus.SetMinSize(fyne.NewSize(10, 10))

	rectHeader := canvas.NewRectangle(color.Transparent)
	rectHeader.SetMinSize(fyne.NewSize(ui.Width, 10))

	linkCancel := widget.NewHyperlinkWithStyle("Back to My Account", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkCancel.OnTapped = func() {
		removeOverlays()
	}

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(ui.Width, 5))

	grid := container.NewVBox()
	grid.Objects = nil

	header := container.NewVBox(
		rectSpacer,
		rectSpacer,
		heading,
		rectSpacer,
		rectSpacer,
	)

	footer := container.NewVBox(
		wSpacer,
		container.NewHBox(
			layout.NewSpacer(),
			linkCancel,
			layout.NewSpacer(),
		),
		wSpacer,
	)

	body := widget.NewLabel("Please save the following 25 recovery words in a safe place. Never share them with anyone.")
	body.Wrapping = fyne.TextWrapWord
	body.Alignment = fyne.TextAlignCenter
	body.TextStyle = fyne.TextStyle{Bold: true}

	btnCopySeed := widget.NewButton("Copy Recovery Words", nil)

	form := container.NewVBox(
		container.NewHBox(
			layout.NewSpacer(),
			container.NewStack(
				rectHeader,
				body,
			),
			layout.NewSpacer(),
		),
		wSpacer,
		container.NewCenter(grid),
		rectSpacer,
		rectSpacer,
		container.NewHBox(
			layout.NewSpacer(),
			container.NewStack(
				rectHeader,
				btnCopySeed,
			),
			layout.NewSpacer(),
		),
		rectSpacer,
	)

	scrollBox := container.NewVScroll(
		container.NewStack(
			form,
		),
	)
	scrollBox.SetMinSize(fyne.NewSize(ui.MaxWidth, ui.Height*0.74))

	formatted := strings.Split(engram.Disk.GetSeed(), " ")

	rect := canvas.NewRectangle(color.RGBA{19, 25, 34, 255})
	rect.SetMinSize(fyne.NewSize(ui.Width, 25))

	for i := 0; i < len(formatted); i++ {
		pos := fmt.Sprintf("%d", i+1)
		word := strings.ReplaceAll(formatted[i], " ", "")
		grid.Add(container.NewStack(
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
		session.Window.Clipboard().SetContent(engram.Disk.GetSeed())
	}

	layout := container.NewBorder(
		container.NewHBox(
			layout.NewSpacer(),
			container.NewVBox(
				header,
				scrollBox,
			),
			layout.NewSpacer(),
		),
		footer,
		nil,
		nil,
	)

	return layout
}

func layoutFrame() fyne.CanvasObject {
	entry := widget.NewEntry()
	layout := container.NewStack(entry)

	resizeWindow(ui.MaxWidth, ui.MaxHeight)
	session.Window.SetContent(layout)
	session.Window.SetFixedSize(false)

	go func() {
		time.Sleep(time.Second * 2)
		removeOverlays()

		ui.MaxWidth = entry.Size().Width
		ui.MaxHeight = entry.Size().Height

		ui.Width = ui.MaxWidth * 0.9
		ui.Height = ui.MaxHeight
		ui.Padding = ui.MaxWidth * 0.05

		resizeWindow(ui.MaxWidth, ui.MaxHeight)
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutMain())
	}()

	overlays := session.Window.Canvas().Overlays()
	overlays.Add(
		container.NewStack(
			canvas.NewRectangle(colors.DarkMatter),
		),
	)

	return layout
}
