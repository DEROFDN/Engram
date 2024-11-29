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
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
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
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	x "fyne.io/x/fyne/widget"
	"github.com/civilware/Gnomon/structures"
	"github.com/civilware/epoch"
	"github.com/civilware/tela"
	"github.com/civilware/tela/logger"
	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/dvm"
	"github.com/deroproject/derohe/globals"
	"github.com/deroproject/derohe/rpc"
	"github.com/deroproject/derohe/walletapi"
	"github.com/deroproject/derohe/walletapi/mnemonics"
	"github.com/deroproject/derohe/walletapi/xswd"
	"github.com/deroproject/graviton"
	qrcode "github.com/skip2/go-qrcode"
)

func layoutMain() fyne.CanvasObject {
	// Set theme
	a.Settings().SetTheme(themes.main)
	session.Domain = "app.main"
	session.Path = ""
	session.Password = ""

	// Define objects

	btnLogin := widget.NewButton("Connect", nil)

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
			if !session.Offline {
				btnLogin.Text = "Connect"
			} else {
				btnLogin.Text = "Decrypt"
			}
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
					if !session.Offline {
						btnLogin.Text = "Connect"
					} else {
						btnLogin.Text = "Decrypt"
					}
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
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutNewAccount())
		removeOverlays()
	}

	linkRecover := widget.NewHyperlinkWithStyle("Recover an existing account", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkRecover.OnTapped = func() {
		session.Domain = "app.restore"
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutRestore())
		removeOverlays()
	}

	linkSettings := widget.NewHyperlinkWithStyle("Settings", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkSettings.OnTapped = func() {
		session.Domain = "app.settings"
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutSettings())
		removeOverlays()
	}

	modeData := binding.BindBool(&session.Offline)
	mode := widget.NewCheckWithData(" Offline Mode", modeData)
	mode.OnChanged = func(b bool) {
		if b {
			session.Offline = true
			btnLogin.Text = "Decrypt"
			btnLogin.Refresh()
		} else {
			session.Offline = false
			btnLogin.Text = "Connect"
			btnLogin.Refresh()
		}
	}

	footer := canvas.NewText("© 2024  DERO FOUNDATION  |  VERSION  "+version.String(), colors.Gray)
	footer.TextSize = 10
	footer.Alignment = fyne.TextAlignCenter
	footer.TextStyle = fyne.TextStyle{Bold: true}

	wPassword := NewReturnEntry()
	wPassword.OnReturn = btnLogin.OnTapped
	wPassword.Password = true
	wPassword.OnChanged = func(s string) {
		session.Error = ""
		if !session.Offline {
			btnLogin.Text = "Connect"
		} else {
			btnLogin.Text = "Decrypt"
		}
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
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutAlert(2))
	}

	// Populate the accounts in dropdown menu
	wAccount := widget.NewSelect(list, nil)
	wAccount.PlaceHolder = "(Select Account)"
	wAccount.OnChanged = func(s string) {
		session.Error = ""
		if !session.Offline {
			btnLogin.Text = "Connect"
		} else {
			btnLogin.Text = "Decrypt"
		}
		btnLogin.Refresh()

		// OnChange set wallet path
		switch session.Network {
		case NETWORK_TESTNET:
			session.Path = filepath.Join(AppPath(), "testnet") + string(filepath.Separator) + s
		case NETWORK_SIMULATOR:
			session.Path = filepath.Join(AppPath(), "testnet_simulator") + string(filepath.Separator) + s
		default:
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

	headerBlock := canvas.NewRectangle(color.Transparent)
	headerBlock.SetMinSize(fyne.NewSize(ui.Width, ui.MaxHeight*0.2))

	headerBox := canvas.NewRectangle(color.Transparent)
	headerBox.SetMinSize(fyne.NewSize(ui.Width, 1))

	frame := &iframe{}

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(ui.Width, 5))

	status.Connection.FillColor = colors.Gray
	status.Cyberdeck.FillColor = colors.Gray
	status.Gnomon.FillColor = colors.Gray
	status.EPOCH.FillColor = colors.Gray
	status.Sync.FillColor = colors.Gray

	form := container.NewStack(
		res.mainBg,
		container.NewVBox(
			wSpacer,
			container.NewStack(
				headerBlock,
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

	return NewVScroll(layout)
}

func layoutDashboard() fyne.CanvasObject {
	resizeWindow(ui.MaxWidth, ui.MaxHeight)

	session.Dashboard = "main"
	session.Domain = "app.wallet"

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
	switch session.Network {
	case NETWORK_TESTNET:
		network = " T  E  S  T  N  E  T "
	case NETWORK_SIMULATOR:
		network = " S  I  M  U  L  A  T  O  R "
	default:
		network = " M  A  I  N  N  E  T "
	}

	rectStatus := canvas.NewRectangle(color.Transparent)
	rectStatus.SetMinSize(fyne.NewSize(10, 10))

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

	cyberdeckText := "CYBERDECK"
	if cyberdeck.WS.server != nil {
		cyberdeckText = "CYBERDECK (WS)"
	} else if cyberdeck.RPC.server != nil {
		cyberdeckText = "CYBERDECK (RPC)"
	} else {
		status.Cyberdeck.FillColor = colors.Gray
		status.Cyberdeck.Refresh()
	}

	cyberdeckLabel := canvas.NewText(cyberdeckText, colors.Gray)
	cyberdeckLabel.TextSize = 12
	cyberdeckLabel.Alignment = fyne.TextAlignTrailing
	cyberdeckLabel.TextStyle = fyne.TextStyle{Bold: false}

	gnomonLabel := canvas.NewText("GNOMON", colors.Gray)
	gnomonLabel.TextSize = 12
	gnomonLabel.Alignment = fyne.TextAlignCenter
	gnomonLabel.TextStyle = fyne.TextStyle{Bold: false}

	epochLabel := canvas.NewText("EPOCH", colors.Gray)
	epochLabel.TextSize = 12
	epochLabel.Alignment = fyne.TextAlignTrailing
	epochLabel.TextStyle = fyne.TextStyle{Bold: false}
	if !epoch.IsActive() {
		if cyberdeck.EPOCH.err != nil {
			status.EPOCH.FillColor = colors.Red
			status.EPOCH.Refresh()
		} else {
			status.EPOCH.FillColor = colors.Gray
			status.EPOCH.Refresh()
		}
	}

	telaLabel := canvas.NewText("TELA", colors.Gray)
	telaLabel.TextSize = 12
	telaLabel.Alignment = fyne.TextAlignCenter
	telaLabel.TextStyle = fyne.TextStyle{Bold: false}

	telaStatus := canvas.NewCircle(colors.Gray)
	if len(tela.GetServerInfo()) > 0 {
		telaStatus.FillColor = colors.Green
	}

	animationCanvas := canvas.NewCircle(color.Transparent)

	if !session.Offline {
		daemonLabel.Text = session.Daemon
		animationStatus := canvas.NewColorRGBAAnimation(
			color.Transparent,
			colors.Yellow,
			time.Second,
			func(c color.Color) {
				animationCanvas.FillColor = c
				animationCanvas.Refresh()
			})

		animationStatus.RepeatCount = fyne.AnimationRepeatForever
		animationStatus.AutoReverse = true
		animationStatus.Start()
	}

	session.WalletHeight = engram.Disk.Get_Height()
	session.StatusText = canvas.NewText(fmt.Sprintf("%d", session.WalletHeight), colors.Gray)
	session.StatusText.TextSize = 12
	session.StatusText.Alignment = fyne.TextAlignTrailing
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
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutHistory())
		removeOverlays()
	}

	menu := widget.NewSelect([]string{"Identity", "My Account", "Messages", "Transfers", "Asset Explorer", "Services", "Cyberdeck", "File Manager", "Contract Builder", "Datapad", "TELA", " "}, nil)
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
		} else if s == "File Manager" {
			session.Window.Canvas().SetContent(layoutTransition())
			session.Window.Canvas().SetContent(layoutFileManager())
			removeOverlays()
		} else if s == "Contract Builder" {
			session.Window.Canvas().SetContent(layoutTransition())
			session.Window.Canvas().SetContent(layoutContractBuilder(""))
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
		} else if s == "TELA" {
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutTELA())
			removeOverlays()
		} else {
			session.Window.Canvas().SetContent(layoutTransition())
			session.Window.Canvas().SetContent(layoutDashboard())
			removeOverlays()
		}

		session.LastDomain = session.Window.Content()
	}

	res.gram.SetMinSize(fyne.NewSize(ui.Width, 150))

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(10, 5))

	rectSquare := canvas.NewRectangle(color.Transparent)
	rectSquare.SetMinSize(fyne.NewSize(5, 5))

	rectOffset := canvas.NewRectangle(color.Transparent)
	rectOffset.SetMinSize(fyne.NewSize(81, 1))

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
		container.NewVBox(
			container.NewHBox(
				container.NewStack(
					rectStatus,
					status.Connection,
				),
				rectSquare,
				daemonLabel,
				layout.NewSpacer(),
				container.NewStack(
					rectOffset,
					session.StatusText,
				),
				rectSquare,
				container.NewStack(
					rectStatus,
					animationCanvas,
					status.Sync,
				),
			),
			rectOffset,
			container.NewHBox(
				container.NewStack(
					rectStatus,
					animationCanvas,
					status.Gnomon,
				),
				rectSquare,
				gnomonLabel,
				layout.NewSpacer(),
				container.NewStack(
					rectOffset,
					epochLabel,
				),
				rectSquare,
				container.NewStack(
					rectStatus,
					animationCanvas,
					status.EPOCH,
				),
			),
			rectOffset,
			container.NewHBox(
				container.NewStack(
					rectStatus,
					telaStatus,
				),
				rectSquare,
				telaLabel,
				layout.NewSpacer(),
				container.NewStack(
					rectOffset,
					cyberdeckLabel,
				),
				rectSquare,
				container.NewStack(
					rectStatus,
					status.Cyberdeck,
				),
			),
		),
	)

	grid := container.NewCenter(
		deroForm,
	)

	gramSend.OnTapped = func() {
		session.LastDomain = session.Window.Content()
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
			container.NewCenter(
				linkLogout,
			),
			rectSpacer,
			rectSpacer,
			rectSpacer,
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

	return NewVScroll(layout)
}

func layoutSend() fyne.CanvasObject {
	session.Domain = "app.send"

	wSpacer := widget.NewLabel(" ")
	frame := &iframe{}

	btnSend := widget.NewButton("Save", nil)

	wAmount := widget.NewEntry()
	wAmount.SetPlaceHolder("Amount")

	wMessage := widget.NewEntry()
	wMessage.SetValidationError(nil)
	wMessage.SetPlaceHolder("Message")
	wMessage.Validator = func(s string) error {
		bytes := []byte(s)
		if len(bytes) <= 130 {
			tx.Comment = s
			wMessage.SetValidationError(nil)
			return nil
		} else {
			err := errors.New("message too long")
			wMessage.SetValidationError(err)
			return err
		}
	}

	wPaymentID := widget.NewEntry()
	wPaymentID.Validator = func(s string) (err error) {
		tx.PaymentID, err = strconv.ParseUint(s, 10, 64)
		if err != nil {
			wPaymentID.SetValidationError(err)
			tx.PaymentID = 0
		}

		return
	}
	wPaymentID.SetPlaceHolder("Payment ID / Service Port")

	options := []string{"Anonymity Set:   2  (None)", "Anonymity Set:   4  (Low)", "Anonymity Set:   8  (Low)", "Anonymity Set:   16  (Recommended)", "Anonymity Set:   32  (Medium)", "Anonymity Set:   64  (High)", "Anonymity Set:   128  (High)"}
	wRings := widget.NewSelect(options, nil)

	wReceiver := widget.NewEntry()
	wReceiver.SetPlaceHolder("Receiver username or address")
	wReceiver.SetValidationError(nil)
	wReceiver.Validator = func(s string) error {
		address, err := globals.ParseValidateAddress(s)
		if err != nil {
			tx.Address = nil
			addr, _ := checkUsername(s, -1)
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

	/*
		// TODO
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
			wAmount.SetValidationError(errors.New("invalid transaction amount"))
			btnSend.Disable()
		} else {
			balance, _ := engram.Disk.Get_Balance()
			entry, err := globals.ParseAmount(s)
			if err != nil {
				tx.Amount = 0
				wAmount.SetValidationError(errors.New("invalid transaction amount"))
				btnSend.Disable()
				return errors.New("invalid transaction amount")
			}

			if entry == 0 {
				tx.Amount = 0
				wAmount.SetValidationError(errors.New("invalid transaction amount"))
				btnSend.Disable()
				return errors.New("invalid transaction amount")
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
				wAmount.SetValidationError(errors.New("insufficient funds"))
			}
			return nil
		}
		return errors.New("invalid transaction amount")
	}

	wAmount.SetValidationError(nil)

	wRings.PlaceHolder = "(Select Anonymity Set)"
	if tx.Ringsize < 2 {
		tx.Ringsize = 16
	} else if len(tx.Pending) > 0 {
		rsIndex := 3
		switch tx.Ringsize {
		case 2:
			rsIndex = 0
		case 4:
			rsIndex = 1
		case 8:
			rsIndex = 2
		case 16:
			rsIndex = 3
		case 32:
			rsIndex = 4
		case 64:
			rsIndex = 5
		case 128:
			rsIndex = 6
		}
		wRings.SetSelectedIndex(rsIndex)
	}

	wRings.OnChanged = func(s string) {
		var err error
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
					session.LastDomain = session.Window.Content()
					session.Window.SetContent(layoutTransition())
					session.Window.SetContent(layoutTransfers())
					removeOverlays()
				}
			} else {
				wReceiver.SetValidationError(errors.New("invalid address"))
				wReceiver.Refresh()
			}
		}
	}

	sendHeading := canvas.NewText("S E N D    M O N E Y", colors.Gray)
	sendHeading.TextSize = 16
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
		rectSpacer,
		rectSpacer,
		container.NewCenter(
			rect300,
			sendHeading,
		),
		rectSpacer,
		rectSpacer,
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
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutDashboard())
		removeOverlays()
		if len(tx.Pending) == 0 {
			tx = Transfers{}
		}
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

	return NewVScroll(layout)
}

func layoutServiceAddress() fyne.CanvasObject {
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
	wMessage.Validator = func(s string) (err error) {
		bytes := []byte(s)
		if len(bytes) <= 130 {
			tx.Comment = s
		} else {
			err = errors.New("message too long")
			wMessage.SetValidationError(err)
		}

		return
	}

	wAmount.Validator = func(s string) error {
		if s == "" {
			tx.Amount = 0
			wAmount.SetValidationError(errors.New("invalid transaction amount"))
			btnCreate.Disable()
		} else {
			amount, err := globals.ParseAmount(s)
			if err != nil {
				tx.Amount = 0
				wAmount.SetValidationError(errors.New("invalid transaction amount"))
				btnCreate.Disable()
				return errors.New("invalid transaction amount")
			}
			wAmount.SetValidationError(nil)
			tx.Amount = amount
			btnCreate.Enable()

			return nil
		}
		return errors.New("invalid transaction amount")
	}

	wAmount.SetValidationError(nil)

	wPaymentID.Validator = func(s string) (err error) {
		tx.PaymentID, err = strconv.ParseUint(s, 10, 64)
		if err != nil {
			tx.PaymentID = 0
			btnCreate.Disable()
			wPaymentID.SetValidationError(err)
			return
		} else {
			if wReceiver.Text != "" {
				btnCreate.Enable()
				wPaymentID.SetValidationError(nil)
				return
			} else {
				err = errors.New("empty payment id")
				wPaymentID.SetValidationError(err)
				return
			}
		}
	}
	wPaymentID.SetPlaceHolder("Payment ID / Service Port")

	sendHeading := canvas.NewText("S E R V I C E    A D D R E S S", colors.Gray)
	sendHeading.TextSize = 16
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
					logger.Errorf("[Service Address] Error: %s\n", err)
					subHeader.Text = "Error"
					subHeader.Refresh()
					btnCopy.Disable()
				} else {
					logger.Printf("[Service Address] New Integrated Address: %s\n", address.String())
					logger.Printf("[Service Address] Arguments: %s\n", address.Arguments)

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

				var imageQR *canvas.Image

				qr, err := qrcode.New(address.String(), qrcode.Highest)
				if err != nil {

				} else {
					qr.BackgroundColor = colors.DarkMatter
					qr.ForegroundColor = colors.Green
				}

				imageQR = canvas.NewImageFromImage(qr.Image(int(ui.Width * 0.65)))
				imageQR.SetMinSize(fyne.NewSize(ui.Width*0.65, ui.Width*0.65))

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
								container.NewHBox(
									layout.NewSpacer(),
									imageQR,
									layout.NewSpacer(),
								),
								widget.NewLabel(""),
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
				wReceiver.SetValidationError(errors.New("invalid address"))
				wReceiver.Refresh()
			}
		}
	}

	form := container.NewVBox(
		rectSpacer,
		rectSpacer,
		container.NewCenter(
			rect300,
			sendHeading,
		),
		rectSpacer,
		rectSpacer,
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
		session.LastDomain = session.Window.Content()
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

	return NewVScroll(layout)
}

func layoutNewAccount() fyne.CanvasObject {
	resizeWindow(ui.MaxWidth, ui.MaxHeight)
	a.Settings().SetTheme(themes.alt)

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
		session.LastDomain = session.Window.Content()
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

	wPassword := widget.NewEntry()
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

	wPasswordConfirm := widget.NewEntry()
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
			err = errors.New("account name is too long")
			wAccount.SetText(session.Name)
			wAccount.Refresh()
			return
		}

		err = checkDir()
		if err != nil {
			session.LastDomain = session.Window.Content()
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutAlert(2))
			return
		}

		switch getNetwork() {
		case NETWORK_TESTNET:
			session.Path = filepath.Join(AppPath(), "testnet", s+".db")
		case NETWORK_SIMULATOR:
			session.Path = filepath.Join(AppPath(), "testnet_simulator", s+".db")
		default:
			session.Path = filepath.Join(AppPath(), "mainnet", s+".db")
		}
		session.Name = s

		if findAccount() {
			err = errors.New("account name already exists")
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
	return NewVScroll(layout)
}

func layoutRestore() fyne.CanvasObject {
	resizeWindow(ui.MaxWidth, ui.MaxHeight)
	a.Settings().SetTheme(themes.alt)

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
		session.LastDomain = session.Window.Content()
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

		} else {
			btnCreate.Disable()
			btnCreate.Refresh()
		}
	}
	wPasswordConfirm.SetPlaceHolder("Confirm Password")
	wPasswordConfirm.Wrapping = fyne.TextWrap(fyne.TextTruncateClip)

	recoveryType := widget.NewSelect([]string{"Recovery Words", "Secret Hex Key", "Import File"}, nil)
	recoveryType.PlaceHolder = "(Recovery Type)"
	recoveryType.SetSelectedIndex(0)
	recoveryType.OnChanged = func(s string) {
		errorText.Text = ""
		errorText.Refresh()
	}

	wAccount := NewMobileEntry()
	wAccount.OnFocusGained = func() {
		scrollBox.Offset = fyne.NewPos(0, 0)
		scrollBox.Refresh()
	}

	wLanguage := widget.NewSelect(mnemonics.Language_List(), nil)
	wLanguage.OnChanged = func(s string) {
		index := wLanguage.SelectedIndex()
		session.Language = index
		session.Window.Canvas().Focus(wAccount)
		errorText.Text = ""
		errorText.Refresh()
	}
	wLanguage.PlaceHolder = "(Select Language)"
	wLanguage.Hide()

	wAccount.SetPlaceHolder("Account Name")
	wAccount.Wrapping = fyne.TextWrap(fyne.TextTruncateClip)
	wAccount.Validator = func(s string) (err error) {
		session.Error = ""
		errorText.Text = ""
		errorText.Refresh()

		if len(s) > 25 {
			err = errors.New("account name is too long")
			wAccount.SetText(session.Name)
			wAccount.Refresh()
			errorText.Text = err.Error()
			errorText.Color = colors.Red
			errorText.Refresh()
			return
		}

		err = checkDir()
		if err != nil {
			session.LastDomain = session.Window.Content()
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutAlert(2))
			return
		}

		switch getNetwork() {
		case NETWORK_TESTNET:
			session.Path = filepath.Join(AppPath(), "testnet") + string(filepath.Separator) + s + ".db"
		case NETWORK_SIMULATOR:
			session.Path = filepath.Join(AppPath(), "testnet_simulator") + string(filepath.Separator) + s + ".db"
		default:
			session.Path = filepath.Join(AppPath(), "mainnet") + string(filepath.Separator) + s + ".db"
		}
		session.Name = s

		if findAccount() {
			err = errors.New("account name already exists")
			errorText.Text = err.Error()
			errorText.Color = colors.Red
			errorText.Refresh()
			return
		}

		if len(session.Password) > 0 && session.Password == session.PasswordConfirm && session.Name != "" {

		} else {
			btnCreate.Disable()
			btnCreate.Refresh()
		}

		if s != "" {
			recoveryType.Disable()
		} else {
			recoveryType.Enable()
		}

		return nil
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

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(ui.Width, 5))

	status.Connection.FillColor = colors.Gray
	status.Cyberdeck.FillColor = colors.Gray
	status.Gnomon.FillColor = colors.Gray
	status.Sync.FillColor = colors.Gray

	grid := container.NewVBox()
	grid.Objects = nil

	word1 := NewMobileEntry()
	word1.PlaceHolder = "Seed Word 1"
	word1.Validator = func(s string) error {
		s = strings.TrimSpace(s)
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("invalid seed word")
		}
		seed[0] = s
		word1.Text = s
		return nil
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
		s = strings.TrimSpace(s)
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("invalid seed word")
		}
		seed[1] = s
		word2.Text = s

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
		s = strings.TrimSpace(s)
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("invalid seed word")
		}
		seed[2] = s
		word3.Text = s

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
		s = strings.TrimSpace(s)
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("invalid seed word")
		}
		seed[3] = s
		word4.Text = s

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
		s = strings.TrimSpace(s)
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("invalid seed word")
		}
		seed[4] = s
		word5.Text = s

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
		s = strings.TrimSpace(s)
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("invalid seed word")
		}
		seed[5] = s
		word6.Text = s

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
	word6.OnFocusGained = func() {
		offset := word6.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			logger.Debugf("[Engram] scrollBox - before: %f\n", scrollBox.Offset.Y)
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
			logger.Debugf("[Engram] scrollBox - after: %f\n", scrollBox.Offset.Y)
		}
		logger.Debugf("[Engram] offset: %f\n", offset)
	}

	word7 := NewMobileEntry()
	word7.PlaceHolder = "Seed Word 7"
	word7.Validator = func(s string) error {
		s = strings.TrimSpace(s)
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("invalid seed word")
		}
		seed[6] = s
		word7.Text = s

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
	word7.OnFocusGained = func() {
		offset := word7.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			logger.Debugf("[Engram] scrollBox - before: %f\n", scrollBox.Offset.Y)
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
			logger.Debugf("[Engram] scrollBox - after: %f\n", scrollBox.Offset.Y)
		}
		logger.Debugf("[Engram] offset: %f\n", offset)
	}

	word8 := NewMobileEntry()
	word8.PlaceHolder = "Seed Word 8"
	word8.Validator = func(s string) error {
		s = strings.TrimSpace(s)
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("invalid seed word")
		}
		seed[7] = s
		word8.Text = s

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
		s = strings.TrimSpace(s)
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("invalid seed word")
		}
		seed[8] = s
		word9.Text = s

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
		s = strings.TrimSpace(s)
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("invalid seed word")
		}
		seed[9] = s
		word10.Text = s

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
		s = strings.TrimSpace(s)
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("invalid seed word")
		}
		seed[10] = s
		word11.Text = s

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
		s = strings.TrimSpace(s)
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("invalid seed word")
		}
		seed[11] = s
		word12.Text = s

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
		s = strings.TrimSpace(s)
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("invalid seed word")
		}
		seed[12] = s
		word13.Text = s

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
		s = strings.TrimSpace(s)
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("invalid seed word")
		}
		seed[13] = s
		word14.Text = s

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
		s = strings.TrimSpace(s)
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("invalid seed word")
		}
		seed[14] = s
		word15.Text = s

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
		s = strings.TrimSpace(s)
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("invalid seed word")
		}
		seed[15] = s
		word16.Text = s

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
		s = strings.TrimSpace(s)
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("invalid seed word")
		}
		seed[16] = s
		word17.Text = s

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
		s = strings.TrimSpace(s)
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("invalid seed word")
		}
		seed[17] = s
		word18.Text = s

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
		s = strings.TrimSpace(s)
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("invalid seed word")
		}
		seed[18] = s
		word19.Text = s

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
		s = strings.TrimSpace(s)
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("invalid seed word")
		}
		seed[19] = s
		word20.Text = s

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
		s = strings.TrimSpace(s)
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("invalid seed word")
		}
		seed[20] = s
		word21.Text = s

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
		s = strings.TrimSpace(s)
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("invalid seed word")
		}
		seed[21] = s
		word22.Text = s

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
		s = strings.TrimSpace(s)
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("invalid seed word")
		}
		seed[22] = s
		word23.Text = s

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
		s = strings.TrimSpace(s)
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("invalid seed word")
		}
		seed[23] = s
		word24.Text = s

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
		s = strings.TrimSpace(s)
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New("invalid seed word")
		}
		seed[24] = s
		word25.Text = s

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
	word25.OnFocusGained = func() {
		offset := word25.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			logger.Debugf("[Engram] scrollBox - before: %f\n", scrollBox.Offset.Y)
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
			logger.Debugf("[Engram] scrollBox - after: %f\n", scrollBox.Offset.Y)
		}
	}

	hexEntry := widget.NewEntry()
	hexEntry.SetPlaceHolder("Secret Key (64 character hex)")
	hexEntry.Validator = func(s string) (err error) {
		_, err = hex.DecodeString(s)
		if len(s) > 64 || err != nil {
			err = errors.New("invalid hex key")
			errorText.Text = err.Error()
			errorText.Color = colors.Red
			errorText.Refresh()
			btnCreate.Disable()

			return
		}

		errorText.Text = ""
		errorText.Refresh()
		if s != "" {
			btnCreate.Enable()
		}

		return
	}

	hexSpacer := canvas.NewRectangle(color.Transparent)
	hexSpacer.SetMinSize(fyne.NewSize(ui.Width, 91))

	hexForm := container.NewVBox(
		rectSpacer,
		hexEntry,
		hexSpacer,
		errorText,
	)

	wordsForm := container.NewVBox(
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
	)

	// Create a new form for account/password inputs
	recoveryForm := container.NewVBox(
		wLanguage,
		rectSpacer,
		rectSpacer,
		wAccount,
		wPassword,
		wPasswordConfirm,
		rectSpacer,
		rectSpacer,
		wordsForm,
	)

	importFileText := canvas.NewText(" ", colors.Green)
	importFileText.TextSize = 12
	importFileText.Alignment = fyne.TextAlignCenter

	importFileForm := container.NewVBox(
		rectSpacer,
		rectSpacer,
		errorText,
		rectSpacer,
		rectSpacer,
		importFileText,
		rectSpacer,
		rectSpacer,
	)

	form := container.NewHBox(
		layout.NewSpacer(),
		container.NewVBox(
			recoveryType,
			rectSpacer,
			recoveryForm,
		),
		layout.NewSpacer(),
	)

	recoveryType.OnChanged = func(s string) {
		errorText.Text = ""
		errorText.Refresh()

		switch s {
		case "Secret Hex Key":
			wLanguage.Show()
			form.Objects[1].(*fyne.Container).Objects[2] = recoveryForm
			recoveryForm.Objects[8] = hexForm
		case "Recovery Words":
			wLanguage.Hide()
			form.Objects[1].(*fyne.Container).Objects[2] = recoveryForm
			recoveryForm.Objects[8] = wordsForm
		case "Import File":
			btnCreate.Disable()
			importFileText.Text = ""
			importFileText.Refresh()
			form.Objects[1].(*fyne.Container).Objects[2] = importFileForm
			dialogFileImport := dialog.NewFileOpen(func(uri fyne.URIReadCloser, err error) {
				if err != nil {
					logger.Errorf("[Engram] File dialog: %s\n", err)
					errorText.Text = "could not import wallet file"
					errorText.Color = colors.Red
					errorText.Refresh()
					return
				}

				if uri == nil {
					return // Canceled
				}

				fileName := uri.URI().String()
				if uri.URI().MimeType() != "text/plain" {
					logger.Errorf("[Engram] Cannot import file %s\n", fileName)
					errorText.Text = "cannot import file"
					errorText.Color = colors.Red
					errorText.Refresh()
					return
				}

				if a.Driver().Device().IsMobile() {
					fileName = uri.URI().Name()
				} else {
					fileName = filepath.Base(strings.Replace(fileName, "file://", "", -1))
				}

				if !strings.HasSuffix(fileName, ".db") {
					logger.Errorf("[Engram] Engram requires .db wallet file\n")
					errorText.Text = "invalid wallet file"
					errorText.Color = colors.Red
					errorText.Refresh()
					return
				}

				filedata, err := readFromURI(uri)
				if err != nil {
					logger.Errorf("[Engram] Cannot read URI file data for %s: %s\n", fileName, err)
					errorText.Text = "cannot read file data"
					errorText.Color = colors.Red
					errorText.Refresh()
					return
				}

				filePath := ""
				switch session.Network {
				case NETWORK_TESTNET:
					filePath = filepath.Join(AppPath(), "testnet", fileName)
				case NETWORK_SIMULATOR:
					filePath = filepath.Join(AppPath(), "testnet_simulator", fileName)
				default:
					filePath = filepath.Join(AppPath(), "mainnet", fileName)
				}

				if _, err = os.Stat(filePath); !os.IsNotExist(err) {
					logger.Errorf("[Engram] Wallet file %q already exists\n", fileName)
					errorText.Text = "wallet file already exists"
					errorText.Color = colors.Red
					errorText.Refresh()
					return
				}

				err = os.WriteFile(filePath, filedata, 0600)
				if err != nil {
					logger.Errorf("[Engram] Importing file %s: %s\n", fileName, err)
					errorText.Text = "error importing wallet file"
					errorText.Color = colors.Red
					errorText.Refresh()
					return
				}

				errorText.Text = fmt.Sprintf("%s wallet file imported successfully", strings.ToLower(session.Network))
				errorText.Color = colors.Green
				errorText.Refresh()

				if len(fileName) > 50 {
					fileName = fileName[0:50] + "..."
				}

				importFileText.Text = fileName
				importFileText.Color = colors.Green
				importFileText.Refresh()

			}, session.Window)

			if !a.Driver().Device().IsMobile() {
				// Open file browser in current directory
				uri, err := storage.ListerForURI(storage.NewFileURI(AppPath()))
				if err == nil {
					dialogFileImport.SetLocation(uri)
				} else {
					logger.Errorf("[Engram] Could not open current directory %s\n", err)
				}
			}

			dialogFileImport.SetFilter(storage.NewExtensionFileFilter([]string{".db"}))
			dialogFileImport.SetView(dialog.ListView)
			dialogFileImport.Resize(fyne.NewSize(ui.Width, ui.Height))
			dialogFileImport.Show()
		}
	}

	body := widget.NewLabel("Your account has been successfully recovered. ")
	body.Wrapping = fyne.TextWrapWord
	body.Alignment = fyne.TextAlignCenter
	body.TextStyle = fyne.TextStyle{Bold: true}

	formSuccess := container.NewHBox(
		layout.NewSpacer(),
		container.NewVBox(
			rectSpacer,
			rectSpacer,
			heading2,
			rectSpacer,
			body,
			rectSpacer,
			rectSpacer,
			container.NewCenter(grid),
			rectSpacer,
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
				container.NewVBox(
					form,
					formSuccess,
				),
				layout.NewSpacer(),
			),
		),
	)

	scrollBox.SetMinSize(fyne.NewSize(ui.Width, ui.Height*0.65))

	btnCreate.OnTapped = func() {
		if engram.Disk != nil {
			closeWallet()
		}

		var err error

		if findAccount() {
			err = errors.New("account name already exists")
			errorText.Text = err.Error()
			errorText.Color = colors.Red
			errorText.Refresh()
			return
		} else {
			errorText.Text = ""
			errorText.Refresh()
		}

		getNetwork()

		var language string
		var temp *walletapi.Wallet_Disk

		if recoveryType.SelectedIndex() == 1 {
			if wAccount.Text == "" {
				err = errors.New("enter account name")
				errorText.Text = err.Error()
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}

			if wLanguage.SelectedIndex() < 0 {
				err = errors.New("select seed language")
				errorText.Text = err.Error()
				errorText.Color = colors.Red
				errorText.Refresh()
				go func() {
					wLanguage.FocusGained()
					time.Sleep(time.Second)
					wLanguage.FocusLost()
				}()
				return
			}

			if wPassword.Text == "" {
				err = errors.New("enter and confirm a password")
				errorText.Text = err.Error()
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}

			if session.Password != session.PasswordConfirm {
				err = errors.New("passwords do not match")
				errorText.Text = err.Error()
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}

			if hexEntry.Text == "" {
				err = errors.New("enter a valid hex key")
				errorText.Text = err.Error()
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}

			if len(hexEntry.Text) > 64 {
				err = errors.New("key must be less than 65 chars")
				errorText.Text = err.Error()
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}

			hexKey, err := hex.DecodeString(hexEntry.Text)
			if err != nil {
				errorText.Text = err.Error()
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}

			temp, err = walletapi.Create_Encrypted_Wallet(session.Path, session.Password, new(crypto.BNRed).SetBytes(hexKey))
			if err != nil {
				errorText.Text = err.Error()
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}

			language = wLanguage.Selected

		} else {
			if wAccount.Text == "" {
				err = errors.New("enter account name")
				errorText.Text = err.Error()
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}

			if wPassword.Text == "" {
				err = errors.New("enter and confirm a password")
				errorText.Text = err.Error()
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}

			if session.Password != session.PasswordConfirm {
				err = errors.New("passwords do not match")
				errorText.Text = err.Error()
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}

			var words string

			for i := 0; i < 25; i++ {
				words += seed[i] + " "
			}

			language, _, err = mnemonics.Words_To_Key(words)
			if err != nil {
				errorText.Text = err.Error()
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}

			temp, err = walletapi.Create_Encrypted_Wallet_From_Recovery_Words(session.Path, session.Password, words)
			if err != nil {
				errorText.Text = err.Error()
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}
		}

		engram.Disk = temp

		if session.Network == NETWORK_MAINNET {
			engram.Disk.SetNetwork(true)
		} else {
			engram.Disk.SetNetwork(false)
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
		rectSpacer,
		rectSpacer,
		heading,
		rectSpacer,
		rectSpacer,
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
	return NewVScroll(layout)
}

func layoutAssetExplorer() fyne.CanvasObject {
	session.Domain = "app.explorer"

	var data []string
	var listData binding.StringList
	var listBox *widget.List

	frame := &iframe{}
	rectLeft := canvas.NewRectangle(color.Transparent)
	rectLeft.SetMinSize(fyne.NewSize(ui.Width*0.40, 35))
	rectRight := canvas.NewRectangle(color.Transparent)
	rectRight.SetMinSize(fyne.NewSize(ui.Width*0.58, 35))
	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(fyne.NewSize(ui.Width, ui.Height*0.45))
	rectWidth := canvas.NewRectangle(color.Transparent)
	rectWidth.SetMinSize(fyne.NewSize(ui.Width, 10))

	heading := canvas.NewText("A S S E T    E X P L O R E R", colors.Gray)
	heading.TextSize = 16
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(6, 5))

	results := canvas.NewText("", colors.Green)
	results.TextSize = 14

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
	entrySCID.Disable()

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
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutDashboard())
		removeOverlays()
	}

	linkClearHistory := widget.NewHyperlinkWithStyle("Clear All", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: false})
	linkClearHistory.OnTapped = func() {
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
			DeleteKey(tree.GetName(), k)
		}

		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutAssetExplorer())
	}

	btnMyAssets := widget.NewButton("My Assets", nil)
	btnMyAssets.OnTapped = func() {
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutMyAssets())
	}

	layoutExplorer := container.NewStack(
		rectWidth,
		container.NewHBox(
			layout.NewSpacer(),
			container.NewVBox(
				rectSpacer,
				container.NewHBox(
					results,
					layout.NewSpacer(),
					linkClearHistory,
				),
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
		results.Text = "  Disabled in offline mode."
		results.Color = colors.Gray
		results.Refresh()
	} else if gnomon.Index == nil {
		results.Text = "  Gnomon is inactive."
		results.Color = colors.Gray
		results.Refresh()
	}

	entrySCID.OnChanged = func(s string) {
		if entrySCID.Text != "" && len(s) == 64 {
			showLoadingOverlay()

			var result []*structures.SCIDVariable
			switch gnomon.Index.DBType {
			case "gravdb":
				result = gnomon.Index.GravDBBackend.GetSCIDVariableDetailsAtTopoheight(s, engram.Disk.Get_Daemon_TopoHeight())
			case "boltdb":
				result = gnomon.Index.BBSBackend.GetSCIDVariableDetailsAtTopoheight(s, engram.Disk.Get_Daemon_TopoHeight())
			}

			if len(result) == 0 {
				_, err := getTxData(s)
				if err != nil {
					return
				}
			}

			err := StoreEncryptedValue("Explorer History", []byte(s), []byte(""))
			if err != nil {
				logger.Errorf("[Asset Explorer] Error saving search result: %s\n", err)
				return
			}

			scid := crypto.HashHexToHash(s)

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

			/*
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
			*/

			entrySCID.SetText("")
			session.LastDomain = session.Window.Content()
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutAssetManager(s))
			removeOverlays()
		}
	}

	go func() {
		if engram.Disk != nil && gnomon.Index != nil {
			for gnomon.Index.LastIndexedHeight < int64(engram.Disk.Get_Daemon_Height()) {
				if session.Domain != "app.explorer" {
					break
				}
				entrySCID.Disable()
				results.Text = "  Gnomon is syncing..."
				results.Color = colors.Yellow
				results.Refresh()
				time.Sleep(time.Second * 1)
			}

			entrySCID.Enable()
			results.Text = "  Loading previous scan history..."
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
			/*
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
			*/

			listBox.UnselectAll()
			session.LastDomain = session.Window.Content()
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutAssetManager(split[4]))
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

	return NewVScroll(layout)
}

func layoutMyAssets() fyne.CanvasObject {
	var data []string
	var listData binding.StringList
	var listBox *widget.List

	frame := &iframe{}
	rectLeft := canvas.NewRectangle(color.Transparent)
	rectLeft.SetMinSize(fyne.NewSize(ui.Width*0.40, 35))
	rectRight := canvas.NewRectangle(color.Transparent)
	rectRight.SetMinSize(fyne.NewSize(ui.Width*0.59, 35))
	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(fyne.NewSize(ui.Width, ui.Height*0.56))
	rectWidth := canvas.NewRectangle(color.Transparent)
	rectWidth.SetMinSize(fyne.NewSize(ui.MaxWidth, 10))

	heading := canvas.NewText("M Y    A S S E T S", colors.Gray)
	heading.TextSize = 16
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(6, 5))

	results := canvas.NewText("", colors.Green)
	results.TextSize = 13

	labelLastScan := canvas.NewText("", colors.Green)
	labelLastScan.TextSize = 13

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
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutAssetExplorer())
		removeOverlays()
	}

	btnRescan := widget.NewButton("Rescan Blockchain", nil)
	btnRescan.Disable()

	layoutAssets := container.NewStack(
		rectWidth,
		container.NewHBox(
			layout.NewSpacer(),
			container.NewVBox(
				rectSpacer,
				container.NewHBox(
					results,
					layout.NewSpacer(),
					labelLastScan,
				),
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

			results.Text = "  Loading previous scan results..."
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
			}

			rescan := func() {
				btnRescan.Disable()
				assetTotal = 0
				assetCount = 0

				t := time.Now()
				timeNow := string(t.Format(time.RFC822))
				StoreEncryptedValue("Asset Scan", []byte("Last Scan"), []byte(timeNow))

				results.Text = "  Indexing..."
				results.Color = colors.Yellow
				results.Refresh()

				owned = 0

				assetData = []string{}
				listBox.UnselectAll()
				listData.Set(assetData)

				if gnomon.Index != nil {
					switch gnomon.Index.DBType {
					case "gravdb":
						assetList = gnomon.Index.GravDBBackend.GetAllOwnersAndSCIDs()
					case "boltdb":
						assetList = gnomon.Index.BBSBackend.GetAllOwnersAndSCIDs()
					}

					for len(assetList) < 5 {
						logger.Printf("[Gnomon] Asset Scan Status: [%d / %d / %d]\n", gnomon.Index.LastIndexedHeight, engram.Disk.Get_Daemon_Height(), len(assetList))
						results.Color = colors.Yellow
						switch gnomon.Index.DBType {
						case "gravdb":
							assetList = gnomon.Index.GravDBBackend.GetAllOwnersAndSCIDs()
						case "boltdb":
							assetList = gnomon.Index.BBSBackend.GetAllOwnersAndSCIDs()
						}
						time.Sleep(time.Second * 5)
					}
				}

				results.Text = "  Scanning results..."
				results.Color = colors.Yellow
				results.Refresh()

				if gnomon.Index != nil {
					switch gnomon.Index.DBType {
					case "gravdb":
						assetList = gnomon.Index.GravDBBackend.GetAllOwnersAndSCIDs()
					case "boltdb":
						assetList = gnomon.Index.BBSBackend.GetAllOwnersAndSCIDs()
					}
				}

				contracts := []crypto.Hash{}

				for sc := range assetList {
					scid := crypto.HashHexToHash(sc)

					if !scid.IsZero() {
						assetCount += 1
						contracts = append(contracts, scid)
					}
				}

				wg := sync.WaitGroup{}
				maxWorkers := 50
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

						results.Text = "  Scanning... " + fmt.Sprintf("%d / %d", assetTotal, assetCount)
						results.Color = colors.Yellow
						results.Refresh()

						bal, _, err := engram.Disk.GetDecryptedBalanceAtTopoHeight(scid, -1, engram.Disk.GetAddress().String())
						if err != nil {
							return
						} else {
							balance := globals.FormatMoney(bal)

							if bal != zerobal {
								err = StoreEncryptedValue("My Assets", []byte(scid.String()), []byte(balance))
								if err != nil {
									logger.Errorf("[History] Failed to store asset: %s\n", err)
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
								logger.Printf("[Assets] Found asset: %s\n", scid.String())
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

				labelLastScan.Text = fmt.Sprintf("  %s", timeNow)
				labelLastScan.Color = colors.Green
				labelLastScan.Refresh()

				listData.Set(assetData)
				btnRescan.Enable()
			}

			btnRescan.OnTapped = rescan

			lastScan, _ := GetEncryptedValue("Asset Scan", []byte("Last Scan"))

			if len(assetData) == 0 && len(lastScan) == 0 {
				rescan()
			}

			if len(lastScan) > 0 {
				results.Text = fmt.Sprintf("  Owned Assets:  %d", owned)
				labelLastScan.Text = fmt.Sprintf("  %s", lastScan)
			} else {
				results.Text = fmt.Sprintf("  Owned Assets:  %d", owned)
				labelLastScan.Text = ""
			}

			results.Color = colors.Green
			results.Refresh()

			labelLastScan.Refresh()

			listData.Set(assetData)

			listBox.OnSelected = func(id widget.ListItemID) {
				split := strings.Split(assetData[id], ";;;")

				/*
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
				*/

				listBox.UnselectAll()
				session.LastDomain = session.Window.Content()
				session.Window.SetContent(layoutTransition())
				session.Window.SetContent(layoutAssetManager(split[4]))
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

	return NewVScroll(layout)
}

func layoutAssetManager(scid string) fyne.CanvasObject {
	captureDomain := session.Domain
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

	labelSigner := canvas.NewText("   SMART  CONTRACT  AUTHOR", colors.Gray)
	labelSigner.TextSize = 14
	labelSigner.Alignment = fyne.TextAlignLeading
	labelSigner.TextStyle = fyne.TextStyle{Bold: true}

	labelOwner := canvas.NewText("   SMART  CONTRACT  OWNER", colors.Gray)
	labelOwner.TextSize = 14
	labelOwner.Alignment = fyne.TextAlignLeading
	labelOwner.TextStyle = fyne.TextStyle{Bold: true}

	labelSCID := canvas.NewText("   SMART  CONTRACT  ID", colors.Gray)
	labelSCID.TextSize = 14
	labelSCID.Alignment = fyne.TextAlignLeading
	labelSCID.TextStyle = fyne.TextStyle{Bold: true}

	labelBalance := canvas.NewText("   ASSET  BALANCE", colors.Gray)
	labelBalance.TextSize = 14
	labelBalance.Alignment = fyne.TextAlignLeading
	labelBalance.TextStyle = fyne.TextStyle{Bold: true}

	labelTransfer := canvas.NewText("   TRANSFER  ASSET", colors.Gray)
	labelTransfer.TextSize = 14
	labelTransfer.Alignment = fyne.TextAlignLeading
	labelTransfer.TextStyle = fyne.TextStyle{Bold: true}

	labelExecute := canvas.NewText("   EXECUTE  ACTION", colors.Gray)
	labelExecute.TextSize = 14
	labelExecute.Alignment = fyne.TextAlignLeading
	labelExecute.TextStyle = fyne.TextStyle{Bold: true}

	var ringsize uint64
	var err error

	options := []string{"Anonymity Set:   2  (None)", "Anonymity Set:   4  (Low)", "Anonymity Set:   8  (Low)", "Anonymity Set:   16  (Recommended)", "Anonymity Set:   32  (Medium)", "Anonymity Set:   64  (High)", "Anonymity Set:   128  (High)"}

	selectRingSize := widget.NewSelect(options, nil)
	selectRingSize.OnChanged = func(s string) {
		regex := regexp.MustCompile("[0-9]+")
		result := regex.FindAllString(selectRingSize.Selected, -1)
		ringsize, err = strconv.ParseUint(result[0], 10, 64)
		if err != nil {
			ringsize = 2
		}
	}

	selectRingSize.SetSelectedIndex(3)

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

	labelSeparator := widget.NewRichTextFromMarkdown("")
	labelSeparator.Wrapping = fyne.TextWrapOff
	labelSeparator.ParseMarkdown("---")

	labelSeparator2 := widget.NewRichTextFromMarkdown("")
	labelSeparator2.Wrapping = fyne.TextWrapOff
	labelSeparator2.ParseMarkdown("---")

	labelSeparator3 := widget.NewRichTextFromMarkdown("")
	labelSeparator3.Wrapping = fyne.TextWrapOff
	labelSeparator3.ParseMarkdown("---")

	labelSeparator4 := widget.NewRichTextFromMarkdown("")
	labelSeparator4.Wrapping = fyne.TextWrapOff
	labelSeparator4.ParseMarkdown("---")

	labelSeparator5 := widget.NewRichTextFromMarkdown("")
	labelSeparator5.Wrapping = fyne.TextWrapOff
	labelSeparator5.ParseMarkdown("---")

	labelSeparator6 := widget.NewRichTextFromMarkdown("")
	labelSeparator6.Wrapping = fyne.TextWrapOff
	labelSeparator6.ParseMarkdown("---")

	labelName := widget.NewRichTextFromMarkdown(name)
	labelName.Wrapping = fyne.TextWrapOff
	labelName.ParseMarkdown("## " + name)

	labelDesc := widget.NewRichText(&widget.TextSegment{
		Text: desc,
		Style: widget.RichTextStyle{
			Alignment: fyne.TextAlignCenter,
			ColorName: theme.ColorNameForeground,
			TextStyle: fyne.TextStyle{Bold: false},
		}})
	labelDesc.Wrapping = fyne.TextWrapWord

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
				exists, err := checkUsername(s, -1)
				if err != nil && exists == "" {
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

	balance := canvas.NewText(fmt.Sprintf("  %d", zerobal), colors.Green)
	balance.TextSize = 20
	balance.TextStyle = fyne.TextStyle{Bold: true}

	btnSend.OnTapped = func() {
		btnSend.Text = "Setting up transfer..."
		btnSend.Disable()
		btnSend.Refresh()
		entryAddress.Disable()
		entryAmount.Disable()
		selectRingSize.Disable()

		txid, err := transferAsset(hash, ringsize, entryAddress.Text, entryAmount.Text)
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
				sHeight := walletapi.Get_Daemon_Height()

				for session.Domain == "app.manager" {
					var zeroscid crypto.Hash
					_, result := engram.Disk.Get_Payments_TXID(zeroscid, txid.String())

					if result.TXID != txid.String() {
						time.Sleep(time.Second * 1)
					} else {
						break
					}
				}

				// If we go DEFAULT_CONFIRMATION_TIMEOUT blocks without exiting 'Confirming...' loop, display failed to transfer and break
				if walletapi.Get_Daemon_Height() > sHeight+int64(DEFAULT_CONFIRMATION_TIMEOUT) {
					entryAddress.Text = ""
					entryAddress.Refresh()
					entryAmount.Text = ""
					entryAmount.Refresh()
					btnSend.Text = "Transaction Failed..."
					btnSend.Disable()
					btnSend.Refresh()
					return
				}

				// If daemon height has incremented, print retry counters into button space
				if walletapi.Get_Daemon_Height()-sHeight > 0 {
					btnSend.Text = fmt.Sprintf("Confirming... (%d/%d)", walletapi.Get_Daemon_Height()-sHeight, DEFAULT_CONFIRMATION_TIMEOUT)
					btnSend.Refresh()
				}

				bal, _, err := engram.Disk.GetDecryptedBalanceAtTopoHeight(hash, -1, engram.Disk.GetAddress().String())
				if err == nil {
					err = StoreEncryptedValue("My Assets", []byte(hash.String()), []byte(globals.FormatMoney(bal)))
					if err != nil {
						logger.Errorf("[Asset] Error storing new asset balance for: %s\n", hash)
					}
					balance.Text = "  " + globals.FormatMoney(bal)
					balance.Refresh()
				}

				if bal != zerobal {
					btnSend.Text = "Send Asset"
					btnSend.Enable()
					btnSend.Refresh()
					entryAddress.Text = ""
					entryAddress.Enable()
					entryAddress.Refresh()
					entryAmount.Text = ""
					entryAmount.Enable()
					entryAmount.Refresh()
					selectRingSize.Enable()
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
		balance.Text = "  " + globals.FormatMoney(bal)
		balance.Refresh()

		if bal == zerobal {
			entryAddress.Disable()
			entryAmount.Disable()
			selectRingSize.Disable()
			btnSend.Text = "You do not own this asset"
			btnSend.Disable()
		}
	}

	if captureDomain == "app.manager" { // was already on manager and opened it again so go back option is to explorer
		captureDomain = "app.explorer"
	}

	linkBack := widget.NewHyperlinkWithStyle(fmt.Sprintf("Back to %s", sessionDomainToString(captureDomain)), nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkBack.OnTapped = func() {
		removeOverlays()
		capture := session.Window.Content()
		session.Window.SetContent(layoutTransition())
		if captureDomain == "app.explorer" {
			session.Window.SetContent(layoutAssetExplorer())
		} else {
			session.Window.SetContent(session.LastDomain)
			session.Domain = captureDomain
		}
		session.LastDomain = capture
	}

	image := canvas.NewImageFromResource(resourceBlankPng)
	image.SetMinSize(fyne.NewSize(ui.Width*0.3, ui.Width*0.3))
	image.FillMode = canvas.ImageFillContain

	if icon != "" {
		var path fyne.Resource
		path, err = fyne.LoadResourceFromURLString(icon)
		if err != nil {
			image.Resource = resourceBlankPng
		} else {
			image.Resource = path
		}

		image.SetMinSize(fyne.NewSize(ui.Width*0.3, ui.Width*0.3))
		image.FillMode = canvas.ImageFillContain
		image.Refresh()
	}

	if name == "" {
		labelName.ParseMarkdown("## --")
	}

	if desc == "" {
		labelDesc = widget.NewRichText(&widget.TextSegment{
			Text: "No description provided",
			Style: widget.RichTextStyle{
				Alignment: fyne.TextAlignCenter,
				ColorName: theme.ColorNameForeground,
				TextStyle: fyne.TextStyle{Italic: true},
			}})
		labelDesc.Wrapping = fyne.TextWrapWord
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
	var signerFunctions []string
	var deroFunctions []string
	var assetFunctions []string

	contract, _, err = dvm.ParseSmartContract(code)
	if err != nil {
		contract = dvm.SmartContract{}
	}

	data := []string{}

	for f := range contract.Functions {
		r, _ := utf8.DecodeRuneInString(contract.Functions[f].Name)

		if !unicode.IsUpper(r) {
			logger.Debugf("[DVM] Function %s is not an exported function - skipping it\n", contract.Functions[f].Name)
		} else if contract.Functions[f].Name == "Initialize" || contract.Functions[f].Name == "InitializePrivate" {
			logger.Debugf("[DVM] Function %s is an initialization function - skipping it\n", contract.Functions[f].Name)
		} else {
			data = append(data, contract.Functions[f].Name)
		}

		for l := range contract.Functions[f].Lines {
			for i := range contract.Functions[f].Lines[l] {
				if contract.Functions[f].Lines[l][i] == "SIGNER" && contract.Functions[f].Lines[l][i+1] == "(" {
					signerFunctions = append(signerFunctions, contract.Functions[f].Name)
				}

				if contract.Functions[f].Lines[l][i] == "DEROVALUE" && contract.Functions[f].Lines[l][i+1] == "(" {
					deroFunctions = append(deroFunctions, contract.Functions[f].Name)
				}

				if contract.Functions[f].Lines[l][i] == "ASSETVALUE" && contract.Functions[f].Lines[l][i+1] == "(" {
					assetFunctions = append(assetFunctions, contract.Functions[f].Name)
				}
			}
		}
	}

	sort.Strings(data)
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

		options := []string{"Anonymity Set:   2  (None)", "Anonymity Set:   4  (Low)", "Anonymity Set:   8  (Low)", "Anonymity Set:   16  (Recommended)", "Anonymity Set:   32  (Medium)", "Anonymity Set:   64  (High)", "Anonymity Set:   128  (High)"}

		var ringsize uint64

		signerRequired := false

		selectRingMembers := widget.NewSelect(options, nil)
		selectRingMembers.PlaceHolder = "(Select Anonymity Set)"

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

				existsDEROValue := false
				existsAssetValue := false

				// Scan code for ASSETVALUE and DEROVALUE
				for l := range contract.Functions[f].Lines {
					for i := range contract.Functions[f].Lines[l] {

						for v := range paramList {
							if paramList[v] == entryDEROValue {
								existsDEROValue = true
							} else if paramList[v] == entryAssetValue {
								existsAssetValue = true
							}
						}

						if contract.Functions[f].Lines[l][i] == "DEROVALUE" && contract.Functions[f].Lines[l][i+1] == "(" && !existsDEROValue {
							paramList = append(paramList, entryDEROValue)
							paramsContainer.Add(d)
							paramsContainer.Refresh()
							existsDEROValue = true
							logger.Debugf("[DVM] Added DEROVALUE: %s\n", contract.Functions[f].Lines[l][i])
						} else if len(deroFunctions) > 0 {
							for df := range deroFunctions {
								if contract.Functions[f].Lines[l][i] == deroFunctions[df] && contract.Functions[f].Lines[l][i+1] == "(" && !existsDEROValue {
									paramList = append(paramList, entryDEROValue)
									paramsContainer.Add(d)
									paramsContainer.Refresh()
									existsDEROValue = true
									logger.Debugf("[DVM] Added DEROVALUE: %s - Func: %s\n", contract.Functions[f].Lines[l][i], deroFunctions[df])
								}
							}
						}

						if contract.Functions[f].Lines[l][i] == "ASSETVALUE" && contract.Functions[f].Lines[l][i+1] == "(" && !existsAssetValue {
							paramList = append(paramList, entryAssetValue)
							paramsContainer.Add(a)
							paramsContainer.Refresh()
							existsAssetValue = true
							logger.Debugf("[DVM] Added ASSETVALUE: %s\n", contract.Functions[f].Lines[l][i])
						} else if len(assetFunctions) > 0 {
							for af := range assetFunctions {
								if contract.Functions[f].Lines[l][i] == assetFunctions[af] && contract.Functions[f].Lines[l][i+1] == "(" && !existsAssetValue {
									paramList = append(paramList, entryAssetValue)
									paramsContainer.Add(a)
									paramsContainer.Refresh()
									existsAssetValue = true
									logger.Debugf("[DVM] Added ASSETVALUE: %s\n", contract.Functions[f].Lines[l][i])
								}
							}
						}

						for si := range signerFunctions {
							if contract.Functions[f].Lines[l][i] == "SIGNER" && contract.Functions[f].Lines[l][i+1] == "(" {
								signerRequired = true
							} else if contract.Functions[f].Lines[l][i] == signerFunctions[si] && contract.Functions[f].Lines[l][i+1] == "(" {
								signerRequired = true
							}
						}
					}
				}

				selectRingMembers.OnChanged = func(s string) {
					if signerRequired {
						ringsize = 2
					} else {
						regex := regexp.MustCompile("[0-9]+")
						result := regex.FindAllString(selectRingMembers.Selected, -1)
						ringsize, err = strconv.ParseUint(result[0], 10, 64)
						if err != nil {
							ringsize = 2
						}
					}
				}

				if signerRequired {
					selectRingMembers.SetSelectedIndex(0)
				} else {
					selectRingMembers.SetSelectedIndex(3)
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
								selectRingMembers,
								rectSpacer,
								rectSpacer,
								paramsContainer,
								rectSpacer,
								rectSpacer,
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
									logger.Debugf("[%s] String: %s\n", params[p].Name, s)
									params[p].ValueString = s
								}
							} else if params[p].Type == 0x4 {
								if params[p].Name+" (Numbers Only)" == entry.PlaceHolder {
									amount, err := globals.ParseAmount(s)
									if err != nil {
										logger.Debugf("[%s] Param error: %s\n", params[p].Name, err)
										entry.SetValidationError(err)
										return err
									} else {
										logger.Debugf("[%s] Amount: %d\n", params[p].Name, amount)
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
					for f := range contract.Functions {
						if contract.Functions[f].Name == funcName.Text {
							params = contract.Functions[f].Params
						}
					}

					var err error

					if signerRequired {
						ringsize = 2
					} else {
						regex := regexp.MustCompile("[0-9]+")
						result := regex.FindAllString(selectRingMembers.Selected, -1)
						ringsize, err = strconv.ParseUint(result[0], 10, 64)
						if err != nil {
							ringsize = 2
							selectRingMembers.SetSelected(options[3])
						}
					}

					logger.Printf("[Engram] Ringsize: %d\n", ringsize)

					btnExecute.Text = "Executing..."
					btnExecute.Disable()
					btnExecute.Refresh()

					storage, err := executeContractFunction(hash, ringsize, dero_amount, asset_amount, funcName.Text, params)
					if err != nil {
						if strings.Contains(err.Error(), "somehow the tx could not be built") {
							btnExecute.Text = fmt.Sprintf("Insufficient Balance: Need %v", globals.FormatMoney(storage))
						} else if strings.Contains(err.Error(), "Discarded knowingly") {
							btnExecute.Text = "Error... discarded knowingly"
						} else if strings.Contains(err.Error(), "Recovered in function") {
							btnExecute.Text = "Error... invalid input"
						} else {
							btnExecute.Text = "Error executing function..."
						}
						btnExecute.Disable()
						btnExecute.Refresh()
					} else {
						btnExecute.Text = "Function executed successfully!"
						btnExecute.Disable()
						btnExecute.Refresh()
					}
				}

				if signerRequired {
					selectRingMembers.SetSelectedIndex(0)
					selectRingMembers.Disable()
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
						/*
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
						*/
						container.NewHBox(
							layout.NewSpacer(),
							image,
							layout.NewSpacer(),
						),
						rectSpacer,
						rectSpacer,
						container.NewHBox(
							layout.NewSpacer(),
							labelName,
							layout.NewSpacer(),
						),
						container.NewHBox(
							layout.NewSpacer(),
							container.NewStack(
								rectWidth90,
								labelDesc,
							),
							layout.NewSpacer(),
						),
						rectSpacer,
						rectSpacer,
						labelSeparator,
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
						labelSeparator2,
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
						labelSeparator3,
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
						labelSeparator4,
						rectSpacer,
						rectSpacer,
						labelBalance,
						rectSpacer,
						balance,
						rectSpacer,
						rectSpacer,
						labelSeparator5,
						rectSpacer,
						rectSpacer,
						labelTransfer,
						rectSpacer,
						rectSpacer,
						rectSpacer,
						selectRingSize,
						rectSpacer,
						entryAddress,
						rectSpacer,
						entryAmount,
						rectSpacer,
						btnSend,
						rectSpacer,
						rectSpacer,
						labelSeparator6,
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

	return NewVScroll(layout)
}

func layoutTransfers() fyne.CanvasObject {
	session.Domain = "app.transfers"

	wSpacer := widget.NewLabel(" ")

	sendTitle := canvas.NewText("T R A N S F E R S", colors.Gray)
	sendTitle.TextStyle = fyne.TextStyle{Bold: true}
	sendTitle.TextSize = 16

	sendDesc := canvas.NewText("", colors.Gray)
	sendDesc.TextSize = 18
	sendDesc.Alignment = fyne.TextAlignCenter
	sendDesc.TextStyle = fyne.TextStyle{Bold: true}

	sendHeading := canvas.NewText("S A V E D    T R A N S F E R S", colors.Gray)
	sendHeading.TextSize = 16
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
	rectListBox.SetMinSize(fyne.NewSize(ui.Width, ui.Height*0.53))

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
		session.LastDomain = session.Window.Content()
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
				return
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
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutTransfersDetail(id))
	}

	btnSend := widget.NewButton("Send Transfers", nil)

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

		entryPassword := NewReturnEntry()
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
						btnSend.Text = "Send Transfers"
						btnSend.Enable()
						btnSend.Refresh()
						return
					}

					go func() {
						btnClear.Disable()
						btnSend.Text = "Confirming..."
						btnSend.Refresh()

						walletapi.WaitNewHeightBlock()
						sHeight := walletapi.Get_Daemon_Height()

						for session.Domain == "app.transfers" {
							var zeroscid crypto.Hash
							_, result := engram.Disk.Get_Payments_TXID(zeroscid, txid.String())

							if result.TXID == txid.String() {
								btnSend.Text = "Transfer Successful!"
								btnSend.Refresh()

								break
							}

							// If we go DEFAULT_CONFIRMATION_TIMEOUT blocks without exiting 'Confirming...' loop, display failed to transfer and break
							if walletapi.Get_Daemon_Height() > sHeight+int64(DEFAULT_CONFIRMATION_TIMEOUT) {
								btnSend.Text = "Transfer failed..."
								btnSend.Disable()
								btnSend.Refresh()
								break
							}

							// If daemon height has incremented, print retry counters into button space
							if walletapi.Get_Daemon_Height()-sHeight > 0 {
								btnSend.Text = fmt.Sprintf("Confirming... (%d/%d)", walletapi.Get_Daemon_Height()-sHeight, DEFAULT_CONFIRMATION_TIMEOUT)
								btnSend.Refresh()
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

		entryPassword.OnReturn = btnSubmit.OnTapped

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

		session.Window.Canvas().Focus(entryPassword)
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
		rectSpacer,
		rectSpacer,
		sendHeading,
		rectSpacer,
		rectSpacer,
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

	return NewVScroll(layout)
}

func layoutTransfersDetail(index int) fyne.CanvasObject {
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

	labelDestination := canvas.NewText("   RECEIVER  ADDRESS", colors.Gray)
	labelDestination.TextSize = 14
	labelDestination.Alignment = fyne.TextAlignLeading
	labelDestination.TextStyle = fyne.TextStyle{Bold: true}

	labelAmount := canvas.NewText("   AMOUNT", colors.Gray)
	labelAmount.TextSize = 14
	labelAmount.Alignment = fyne.TextAlignLeading
	labelAmount.TextStyle = fyne.TextStyle{Bold: true}

	labelService := canvas.NewText("   SERVICE  ADDRESS", colors.Gray)
	labelService.TextSize = 14
	labelService.Alignment = fyne.TextAlignLeading
	labelService.TextStyle = fyne.TextStyle{Bold: true}

	labelDestPort := canvas.NewText("   DESTINATION  PORT", colors.Gray)
	labelDestPort.TextSize = 14
	labelDestPort.TextStyle = fyne.TextStyle{Bold: true}

	labelSourcePort := canvas.NewText("   SOURCE  PORT", colors.Gray)
	labelSourcePort.TextSize = 14
	labelSourcePort.TextStyle = fyne.TextStyle{Bold: true}

	labelFees := canvas.NewText("   TRANSACTION  FEES", colors.Gray)
	labelFees.TextSize = 14
	labelFees.TextStyle = fyne.TextStyle{Bold: true}

	labelPayload := canvas.NewText("   PAYLOAD", colors.Gray)
	labelPayload.TextSize = 14
	labelPayload.TextStyle = fyne.TextStyle{Bold: true}

	labelReply := canvas.NewText("   REPLY  ADDRESS", colors.Gray)
	labelReply.TextSize = 14
	labelReply.TextStyle = fyne.TextStyle{Bold: true}

	labelSeparator := widget.NewRichTextFromMarkdown("")
	labelSeparator.Wrapping = fyne.TextWrapOff
	labelSeparator.ParseMarkdown("---")

	labelSeparator2 := widget.NewRichTextFromMarkdown("")
	labelSeparator2.Wrapping = fyne.TextWrapOff
	labelSeparator2.ParseMarkdown("---")

	labelSeparator3 := widget.NewRichTextFromMarkdown("")
	labelSeparator3.Wrapping = fyne.TextWrapOff
	labelSeparator3.ParseMarkdown("---")

	labelSeparator4 := widget.NewRichTextFromMarkdown("")
	labelSeparator4.Wrapping = fyne.TextWrapOff
	labelSeparator4.ParseMarkdown("---")

	labelSeparator5 := widget.NewRichTextFromMarkdown("")
	labelSeparator5.Wrapping = fyne.TextWrapOff
	labelSeparator5.ParseMarkdown("---")

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
	valueAmount.Text = "  " + globals.FormatMoney(details.Amount)

	valueDestPort := canvas.NewText("", colors.Account)
	valueDestPort.TextSize = 22
	valueDestPort.TextStyle = fyne.TextStyle{Bold: true}

	if details.Payload_RPC.HasValue(rpc.RPC_DESTINATION_PORT, rpc.DataUint64) {
		port := fmt.Sprintf("%d", details.Payload_RPC.Value(rpc.RPC_DESTINATION_PORT, rpc.DataUint64))
		valueDestPort.Text = "  " + port
	} else {
		valueDestPort.Text = "  0"
	}

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
			tx.Pending = tx.Pending[:index]
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
						labelSeparator,
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
						labelSeparator2,
						rectSpacer,
						rectSpacer,
						labelReply,
						rectSpacer,
						valueReply,
						rectSpacer,
						rectSpacer,
						labelSeparator3,
						rectSpacer,
						rectSpacer,
						labelPayload,
						rectSpacer,
						container.NewStack(
							rectWidth90,
							valuePayload,
						),
						rectSpacer,
						rectSpacer,
						labelSeparator4,
						rectSpacer,
						rectSpacer,
						labelDestPort,
						rectSpacer,
						container.NewStack(
							rectWidth90,
							valueDestPort,
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

	return NewVScroll(layout)
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

	return NewVScroll(layout)
}

func layoutSettings() fyne.CanvasObject {
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

	entryAddress := widget.NewEntry()
	entryAddress.Validator = func(s string) (err error) {
		/*
			_, err := net.ResolveTCPAddr("tcp", s)
		*/
		regex := `^(?:[a-zA-Z0-9]{1,62}(?:[-\.][a-zA-Z0-9]{1,62})+)(:\d+)?$`
		test := regexp.MustCompile(regex)

		// Trim off http, https, wss, ws to validate regex on 'actual' uri for connection. If none match, s is just s as normal
		var ssplit string
		if strings.HasPrefix(s, "https") {
			ssplit = strings.TrimPrefix(strings.ToLower(s), "https://")
		} else if strings.HasPrefix(s, "http") {
			ssplit = strings.TrimPrefix(strings.ToLower(s), "http://")
		} else if strings.HasPrefix(s, "wss") {
			ssplit = strings.TrimPrefix(strings.ToLower(s), "wss://")
		} else if strings.HasPrefix(s, "ws") {
			ssplit = strings.TrimPrefix(strings.ToLower(s), "ws://")
		} else {
			// s is s
			ssplit = s
		}

		if test.MatchString(ssplit) {
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
	switch session.Network {
	case NETWORK_TESTNET:
		selectNodes.Options = []string{"testnetexplorer.dero.io:40402", "127.0.0.1:40402"}
	case NETWORK_SIMULATOR:
		selectNodes.Options = []string{"127.0.0.1:20000"}
		selectNodes.PlaceHolder = "Select Simulator Node ..."
	default:
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

	radioNetwork := widget.NewRadioGroup([]string{NETWORK_MAINNET, NETWORK_TESTNET, NETWORK_SIMULATOR}, nil)
	radioNetwork.Required = true
	radioNetwork.Horizontal = false
	radioNetwork.OnChanged = func(s string) {
		if s == NETWORK_TESTNET {
			setNetwork(s)
			selectNodes.Options = []string{"testnetexplorer.dero.io:40402", "127.0.0.1:40402"}
			selectNodes.PlaceHolder = "Select Public Node ..."
		} else if s == NETWORK_SIMULATOR {
			setNetwork(s)
			selectNodes.Options = []string{"127.0.0.1:20000"}
			selectNodes.PlaceHolder = "Select Simulator Node ..."
		} else {
			setNetwork(NETWORK_MAINNET)
			selectNodes.Options = []string{"node.derofoundation.org:11012", "127.0.0.1:10102"}
			selectNodes.PlaceHolder = "Select Public Node ..."
		}

		// Change globals.Config mainnet/testnet to match network
		globals.InitNetwork()

		selectNodes.Refresh()
	}

	net, _ := GetValue("settings", []byte("network"))

	if string(net) == NETWORK_TESTNET {
		radioNetwork.SetSelected(NETWORK_TESTNET)
	} else if string(net) == NETWORK_SIMULATOR {
		radioNetwork.SetSelected(NETWORK_SIMULATOR)
	} else {
		radioNetwork.SetSelected(NETWORK_MAINNET)
	}

	radioNetwork.Refresh()

	entryUser := widget.NewEntry()
	entryUser.PlaceHolder = "Username"
	entryUser.SetText(cyberdeck.RPC.user)

	entryPass := widget.NewEntry()
	entryPass.PlaceHolder = "Password"
	entryPass.Password = true
	entryPass.SetText(cyberdeck.RPC.pass)

	entryUser.OnChanged = func(s string) {
		cyberdeck.RPC.user = s
	}

	entryPass.OnChanged = func(s string) {
		cyberdeck.RPC.pass = s
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
		network := radioNetwork.Selected
		if network == NETWORK_TESTNET {
			setNetwork(network)
		} else if network == NETWORK_SIMULATOR {
			setNetwork(network)
		} else {
			setNetwork(NETWORK_MAINNET)
		}
		setDaemon(entryAddress.Text)

		initSettings()

		resizeWindow(ui.MaxWidth, ui.MaxHeight)
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutMain())
		removeOverlays()
	}

	btnRestore.OnTapped = func() {
		setNetwork(NETWORK_MAINNET)
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
			if parseError, ok := err.(*os.PathError); !ok {
				err = fmt.Errorf("error clearing local %s data", session.Network)
			} else {
				err = parseError.Err
			}

			statusText.Color = colors.Red
			statusText.Text = err.Error()
			statusText.Refresh()
			return
		}

		statusText.Color = colors.Green
		statusText.Text = fmt.Sprintf("Gnomon %s data successfully deleted.", strings.ToLower(session.Network))
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

	scrollBox := container.NewVScroll(
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

	return NewVScroll(layout)
}

func layoutMessages() fyne.CanvasObject {
	session.Domain = "app.messages"

	if !walletapi.Connected {
		session.Window.SetContent(layoutSettings())
	}

	title := canvas.NewText("M Y    C O N T A C T S", colors.Gray)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 16

	menuLabel := canvas.NewText("  M O R E   O P T I O N S  ", colors.Gray)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	checkLimit := widget.NewCheck(" Show only recent messages", nil)
	checkLimit.OnChanged = func(b bool) {
		if b {
			session.LimitMessages = true
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutMessages())
			removeOverlays()
		} else {
			session.LimitMessages = false
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutMessages())
			removeOverlays()
		}
	}

	if session.LimitMessages {
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
		session.LastDomain = session.Window.Content()
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
	rectListBox.SetMinSize(fyne.NewSize(ui.Width, ui.Height*0.43))

	messages.Data = nil

	var height uint64

	if session.LimitMessages {
		height = engram.Disk.Get_Height() - 1000000
	} else {
		height = 0
	}

	data := getMessages(height)
	temp := data

	list := binding.BindStringList(&data)

	msgbox.List = widget.NewListWithData(list,
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
			address := short[len(short)-DEFAULT_USERADDR_SHORTEN_LENGTH:]
			username := dataItem[1]
			// If a username is longer than what *would* be a 'short' address of ...xyzxyzxyzx (e.g. 13), then shorten as well to be similar sizing
			if len(username) > DEFAULT_USERADDR_SHORTEN_LENGTH+3 {
				username = "..." + username[len(username)-DEFAULT_USERADDR_SHORTEN_LENGTH:]
			}

			if username == "" {
				co.(*fyne.Container).Objects[0].(*widget.Label).SetText("..." + address)
			} else {
				co.(*fyne.Container).Objects[0].(*widget.Label).SetText(username)
			}
			co.(*fyne.Container).Objects[0].(*widget.Label).Wrapping = fyne.TextWrapWord
			co.(*fyne.Container).Objects[0].(*widget.Label).TextStyle.Bold = false
			co.(*fyne.Container).Objects[0].(*widget.Label).Alignment = fyne.TextAlignLeading
		})

	msgbox.List.OnSelected = func(id widget.ListItemID) {
		msgbox.List.UnselectAll()
		split := strings.Split(data[id], "~~~")
		if split[1] == "" {
			messages.Contact = split[0]
		} else {
			messages.Contact = split[1]
		}

		session.LastDomain = session.Window.Content()
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
			//_, err := engram.Disk.NameToAddress(messages.Contact)
			_, err := checkUsername(messages.Contact, -1)
			if err != nil {
				return
			}
		}

		session.LastDomain = session.Window.Content()
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
				//_, err := engram.Disk.NameToAddress(s)
				_, err = checkUsername(s, -1)
				if err != nil {
					btnSend.Disable()
					return errors.New("invalid username or address")
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

		return errors.New("invalid username or address")
	}

	messageForm := container.NewVBox(
		rectSpacer,
		rectSpacer,
		container.NewHBox(
			layout.NewSpacer(),
			title,
			layout.NewSpacer(),
		),
		rectSpacer,
		rectSpacer,
		entrySearch,
		rectSpacer,
		rectSpacer,
		container.NewStack(
			rectListBox,
			msgbox.List,
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

	return NewVScroll(layout)
}

func layoutPM() fyne.CanvasObject {
	session.Domain = "app.messages.contact"

	if !walletapi.Connected {
		session.Window.SetContent(layoutSettings())
	}

	getPrimaryUsername()

	contactAddress := ""

	// So message contact sizes are not overblown from UI
	_, err := globals.ParseValidateAddress(messages.Contact)
	if err != nil {
		//_, err := engram.Disk.NameToAddress(messages.Contact)
		_, err := checkUsername(messages.Contact, -1)
		if err == nil {
			contactAddress = messages.Contact
		}
	} /* else {
		short := messages.Contact[len(messages.Contact)-10:]
		contactAddress = "..." + short
	}*/

	// Safety, even though valid addresses are sized enough but usernames may not be
	if len(messages.Contact) > DEFAULT_USERADDR_SHORTEN_LENGTH+3 {
		short := messages.Contact[len(messages.Contact)-DEFAULT_USERADDR_SHORTEN_LENGTH:]
		contactAddress = "..." + short
	}

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
		session.LastDomain = session.Window.Content()
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
	subframe.SetMinSize(fyne.NewSize(ui.Width, ui.Height*0.51))
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
	var height uint64

	if session.LimitMessages {
		height = engram.Disk.Get_Height() - 1000000
	} else {
		height = 0
	}

	data := getMessagesFromUser(messages.Contact, height)

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

			//uname, err := engram.Disk.NameToAddress(split[0])
			uname, err := checkUsername(split[0], -1)
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
		//check, err := engram.Disk.NameToAddress(messages.Contact)
		check, err := checkUsername(messages.Contact, -1)
		if err == nil {
			contact = check
		}

		_, err = globals.ParseValidateAddress(contact)
		if err != nil {
			session.LastDomain = session.Window.Content()
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

	btnSend.OnTapped = func() {
		if messages.Message == "" {
			return
		}
		contact := ""
		_, err := globals.ParseValidateAddress(messages.Contact)
		if err != nil {
			//check, err := engram.Disk.NameToAddress(messages.Contact)
			check, err := checkUsername(messages.Contact, -1)
			if err != nil {
				logger.Errorf("[Message] Failed to send: %s\n", err)
				btnSend.Text = "Failed to verify address..."
				btnSend.Disable()
				btnSend.Refresh()
				return
			}
			contact = check
		} else {
			contact = messages.Contact
		}

		btnSend.Text = "Setting up transfer..."
		btnSend.Disable()
		btnSend.Refresh()

		txid, err := sendMessage(messages.Message, session.Username, contact)
		if err != nil {
			logger.Errorf("[Message] Failed to send: %s\n", err)
			btnSend.Text = "Failed to send message..."
			btnSend.Disable()
			btnSend.Refresh()
			return
		}

		logger.Printf("[Message] Dispatched transaction successfully to: %s\n", messages.Contact)
		btnSend.Text = "Confirming..."
		btnSend.Disable()
		btnSend.Refresh()
		messages.Message = ""
		entry.Text = ""
		entry.Refresh()

		go func() {
			walletapi.WaitNewHeightBlock()
			sHeight := walletapi.Get_Daemon_Height()
			var success bool
			for session.Domain == "app.messages.contact" {
				var zeroscid crypto.Hash
				_, result := engram.Disk.Get_Payments_TXID(zeroscid, txid.String())

				if result.TXID != txid.String() {
					time.Sleep(time.Second * 1)
				} else {
					success = true
				}

				// If we go DEFAULT_CONFIRMATION_TIMEOUT blocks without exiting 'Confirming...' loop, display failed to transfer and break
				if walletapi.Get_Daemon_Height() > sHeight+int64(DEFAULT_CONFIRMATION_TIMEOUT) {
					btnSend.Text = "Failed to send message..."
					btnSend.Disable()
					btnSend.Refresh()
					break
				}

				// If daemon height has incremented, print retry counters into button space
				if walletapi.Get_Daemon_Height()-sHeight > 0 {
					btnSend.Text = fmt.Sprintf("Confirming... (%d/%d)", walletapi.Get_Daemon_Height()-sHeight, DEFAULT_CONFIRMATION_TIMEOUT)
					btnSend.Refresh()
				}

				// If success, reload page w/ latest content. Otherwise retain the Failure message for UX relay
				if success {
					session.Window.SetContent(layoutTransition())
					session.Window.SetContent(layoutPM())
					break
				} else {
					time.Sleep(time.Second * 1)
				}
			}
		}()
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

	return NewVScroll(layout)
}

func layoutCyberdeck() fyne.CanvasObject {
	session.Domain = "app.cyberdeck"

	go refreshXSWDList()

	wSpacer := widget.NewLabel(" ")

	title := canvas.NewText("C Y B E R D E C K", colors.Gray)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 16

	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(fyne.NewSize(ui.Width, ui.Height*0.20))

	frame := &iframe{}

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(10, 5))

	rectWidth90 := canvas.NewRectangle(color.Transparent)
	rectWidth90.SetMinSize(fyne.NewSize(ui.Width, 0))

	rpcLabel := canvas.NewText("      C O N F I G U R A T I O N      ", colors.Gray)
	rpcLabel.TextSize = 11
	rpcLabel.Alignment = fyne.TextAlignCenter
	rpcLabel.TextStyle = fyne.TextStyle{Bold: true}

	wsLabel := canvas.NewText("      C O N F I G U R A T I O N      ", colors.Gray)
	wsLabel.TextSize = 11
	wsLabel.Alignment = fyne.TextAlignCenter
	wsLabel.TextStyle = fyne.TextStyle{Bold: true}

	labelConnections := canvas.NewText("  C O N N E C T I O N S  ", colors.Gray)
	labelConnections.TextSize = 11
	labelConnections.Alignment = fyne.TextAlignCenter
	labelConnections.TextStyle = fyne.TextStyle{Bold: true}

	menuLabel := canvas.NewText("  M O R E   O P T I O N S  ", colors.Gray)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	sep1 := canvas.NewRectangle(colors.Gray)
	sep1.SetMinSize(fyne.NewSize(ui.Width*0.2, 2))

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep1,
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
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutDashboard())
		removeOverlays()
	}

	shortShard := canvas.NewText("APPLICATION  CONNECTIONS", colors.Gray)
	shortShard.TextStyle = fyne.TextStyle{Bold: true}
	shortShard.TextSize = 12

	linkColor := colors.Green

	if cyberdeck.RPC.server == nil {
		session.Link = "Blocked"
		linkColor = colors.Gray
	}

	cyberdeck.RPC.status = canvas.NewText(session.Link, linkColor)
	cyberdeck.RPC.status.TextSize = 22
	cyberdeck.RPC.status.TextStyle = fyne.TextStyle{Bold: true}

	serverStatus := canvas.NewText("APPLICATION  CONNECTIONS", colors.Gray)
	serverStatus.TextSize = 12
	serverStatus.Alignment = fyne.TextAlignCenter
	serverStatus.TextStyle = fyne.TextStyle{Bold: true}

	linkCenter := container.NewCenter(
		cyberdeck.RPC.status,
	)

	cyberdeck.RPC.userText = widget.NewEntry()
	cyberdeck.RPC.userText.PlaceHolder = "Username"
	cyberdeck.RPC.userText.OnChanged = func(s string) {
		if len(s) > 1 {
			cyberdeck.RPC.user = s
		}
	}

	cyberdeck.RPC.passText = widget.NewEntry()
	cyberdeck.RPC.passText.Password = true
	cyberdeck.RPC.passText.PlaceHolder = "Password"
	cyberdeck.RPC.passText.OnChanged = func(s string) {
		if len(s) > 1 {
			cyberdeck.RPC.pass = s
		}
	}

	cyberdeck.RPC.portText = widget.NewEntry()
	cyberdeck.RPC.portText.PlaceHolder = "0.0.0.0:10103"
	cyberdeck.RPC.portText.Validator = func(s string) (err error) {
		regex := `^(?:[a-zA-Z0-9]{1,62}(?:[-\.][a-zA-Z0-9]{1,62})+)(:\d+)?$`
		test := regexp.MustCompile(regex)
		if test.MatchString(s) {
			cyberdeck.RPC.portText.SetValidationError(nil)
		} else {
			err = errors.New("invalid host name")
			cyberdeck.RPC.portText.SetValidationError(err)
		}

		return
	}
	cyberdeck.RPC.portText.SetText(getCyberdeck("RPC"))

	linkColor = colors.Green

	if cyberdeck.WS.server == nil {
		session.Link = "Blocked"
		linkColor = colors.Gray
	}

	cyberdeck.WS.status = canvas.NewText(session.Link, linkColor)
	cyberdeck.WS.status.TextSize = 22
	cyberdeck.WS.status.TextStyle = fyne.TextStyle{Bold: true}

	deckChoice := widget.NewSelect([]string{"Web Sockets (WS)", "Remote Procedure Calls (RPC)"}, nil)

	cyberdeck.RPC.toggle = widget.NewButton("Turn On", nil)
	cyberdeck.RPC.toggle.OnTapped = func() {
		switch session.Network {
		case NETWORK_TESTNET:
			if cyberdeck.RPC.portText.Validate() != nil {
				cyberdeck.RPC.port = fmt.Sprintf("127.0.0.1:%d", DEFAULT_TESTNET_WALLET_PORT)
				cyberdeck.RPC.portText.SetText(cyberdeck.RPC.port)
			} else {
				cyberdeck.RPC.port = cyberdeck.RPC.portText.Text
			}
		case NETWORK_SIMULATOR:
			if cyberdeck.RPC.portText.Validate() != nil {
				cyberdeck.RPC.port = fmt.Sprintf("127.0.0.1:%d", DEFAULT_SIMULATOR_WALLET_PORT)
				cyberdeck.RPC.portText.SetText(cyberdeck.RPC.port)
			} else {
				cyberdeck.RPC.port = cyberdeck.RPC.portText.Text
			}
		default:
			if cyberdeck.RPC.portText.Validate() != nil {
				cyberdeck.RPC.port = fmt.Sprintf("127.0.0.1:%d", DEFAULT_WALLET_PORT)
				cyberdeck.RPC.portText.SetText(cyberdeck.RPC.port)
			} else {
				cyberdeck.RPC.port = cyberdeck.RPC.portText.Text
			}
		}

		toggleRPCServer(cyberdeck.RPC.port)
		if cyberdeck.RPC.server != nil {
			setCyberdeck(cyberdeck.RPC.port, "RPC")
			deckChoice.Disable()
			cyberdeck.RPC.portText.Disable()
		} else {
			deckChoice.Enable()
			cyberdeck.RPC.portText.Enable()
		}
	}

	if cyberdeck.WS.portText == nil {
		cyberdeck.WS.portText = widget.NewEntry()
		cyberdeck.WS.portText.PlaceHolder = "0.0.0.0:44326"
		cyberdeck.WS.portText.Validator = func(s string) (err error) {
			regex := `^(?:[a-zA-Z0-9]{1,62}(?:[-\.][a-zA-Z0-9]{1,62})+)(:\d+)?$`
			test := regexp.MustCompile(regex)
			if test.MatchString(s) {
				cyberdeck.WS.portText.SetValidationError(nil)
			} else {
				err = errors.New("invalid host name")
				cyberdeck.WS.portText.SetValidationError(err)
			}

			return
		}
	}

	cyberdeck.WS.toggle = widget.NewButton("Turn On", nil)
	cyberdeck.WS.toggle.OnTapped = func() {
		if cyberdeck.WS.portText.Validate() != nil {
			cyberdeck.WS.port = fmt.Sprintf("127.0.0.1:%d", xswd.XSWD_PORT)
			cyberdeck.WS.portText.SetText(cyberdeck.WS.port)
		} else {
			_, err := net.ResolveTCPAddr("tcp", cyberdeck.WS.port)
			if err != nil {
				logger.Errorf("[Cyberdeck] XSWD port: %s\n", err)
				cyberdeck.WS.port = fmt.Sprintf("127.0.0.1:%d", xswd.XSWD_PORT)
				cyberdeck.WS.portText.SetText(cyberdeck.WS.port)
			} else {
				cyberdeck.WS.port = cyberdeck.WS.portText.Text
			}
		}

		cyberdeck.EPOCH.err = nil
		toggleXSWD(cyberdeck.WS.port)
		if cyberdeck.WS.server != nil {
			setCyberdeck(cyberdeck.WS.port, "WS")
			cyberdeck.WS.portText.Disable()
			deckChoice.Disable()
			if cyberdeck.EPOCH.enabled {
				if cyberdeck.EPOCH.allowWithAddress {
					// If address is defined by dApp, GetWork will be started and stopped upon each WS call
					logger.Printf("[EPOCH] dApp addresses are enabled\n")
					return
				}

				err := epoch.StartGetWork(engram.Disk.GetAddress().String(), session.Daemon)
				if err != nil {
					logger.Errorf("[EPOCH] Connecting: %s\n", err)
					cyberdeck.EPOCH.err = err
				} else {
					cyberdeck.EPOCH.err = nil
					setCyberdeck(epoch.GetPort(), "EPOCH")
				}
			}
		} else {
			stopEPOCH()
			cyberdeck.WS.portText.Enable()
			deckChoice.Enable()
		}
	}

	if session.Offline {
		cyberdeck.RPC.toggle.Text = "Disabled in Offline Mode"
		cyberdeck.RPC.toggle.Disable()
		cyberdeck.RPC.portText.Disable()
		cyberdeck.WS.toggle.Text = "Disabled in Offline Mode"
		cyberdeck.WS.toggle.Disable()
		cyberdeck.WS.portText.Disable()
	} else {
		if cyberdeck.RPC.server != nil {
			cyberdeck.RPC.status.Text = "Allowed"
			cyberdeck.RPC.status.Color = colors.Green
			cyberdeck.RPC.toggle.Text = "Turn Off"
			cyberdeck.RPC.userText.Disable()
			cyberdeck.RPC.passText.Disable()
			cyberdeck.RPC.portText.Disable()
			deckChoice.Disable()
		} else {
			cyberdeck.RPC.status.Text = "Blocked"
			cyberdeck.RPC.status.Color = colors.Gray
			cyberdeck.RPC.toggle.Text = "Turn On"
			cyberdeck.RPC.userText.Enable()
			cyberdeck.RPC.passText.Enable()
			cyberdeck.RPC.portText.Enable()
		}

		if cyberdeck.WS.server != nil {
			cyberdeck.WS.status.Text = "Allowed"
			cyberdeck.WS.status.Color = colors.Green
			cyberdeck.WS.toggle.Text = "Turn Off"
			cyberdeck.WS.portText.Disable()
			deckChoice.Disable()
		} else {
			cyberdeck.WS.status.Text = "Blocked"
			cyberdeck.WS.status.Color = colors.Gray
			cyberdeck.WS.toggle.Text = "Turn On"
			cyberdeck.WS.portText.Enable()
		}
	}

	cyberdeck.RPC.userText.SetText(cyberdeck.RPC.user)
	cyberdeck.RPC.passText.SetText(cyberdeck.RPC.pass)

	linkCopy := widget.NewHyperlinkWithStyle("Copy Credentials", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkCopy.OnTapped = func() {
		session.Window.Clipboard().SetContent(cyberdeck.RPC.user + ":" + cyberdeck.RPC.pass)
	}

	linkPermissions := widget.NewHyperlinkWithStyle("Settings", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkPermissions.OnTapped = func() {
		//if cyberdeck.WS.server != nil {
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutXSWDPermissions())
		removeOverlays()
		//}
	}

	/*
		linkApps := widget.NewHyperlinkWithStyle("View Connections", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		linkApps.OnTapped = func() {
			if cyberdeck.WS.server != nil {
				session.LastDomain = session.Window.Content()
				session.Window.SetContent(layoutTransition())
				session.Window.SetContent(layoutXSWDConnections())
				removeOverlays()
			}
		}
	*/

	cyberdeck.WS.list = widget.NewList(
		func() int {
			return len(cyberdeck.WS.apps)
		},
		func() fyne.CanvasObject {
			return container.NewVBox(
				widget.NewLabel(""),
				//widget.NewLabel(""),
			)
		},
		func(li widget.ListItemID, co fyne.CanvasObject) {
			app := cyberdeck.WS.apps[li]
			co.(*fyne.Container).Objects[0].(*widget.Label).SetText(app.Name)
			//co.(*fyne.Container).Objects[1].(*widget.Label).SetText(app.Id)
		},
	)

	cyberdeck.WS.list.OnSelected = func(id widget.ListItemID) {
		cyberdeck.WS.list.UnselectAll()
		cyberdeck.WS.list.FocusLost()
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutXSWDAppManager(&cyberdeck.WS.apps[id]))
		removeOverlays()
	}

	xswdForm := container.NewVBox(
		rectSpacer,
		rectSpacer,
		container.NewHBox(
			layout.NewSpacer(),
			line1,
			layout.NewSpacer(),
			wsLabel,
			layout.NewSpacer(),
			line2,
			layout.NewSpacer(),
		),
		container.NewCenter(
			layout.NewSpacer(),
			container.NewCenter(
				container.NewVBox(
					rectWidth90,
					rectSpacer,
					container.NewCenter(
						cyberdeck.WS.status,
					),
					rectSpacer,
					serverStatus,
					wSpacer,
					cyberdeck.WS.toggle,
					rectSpacer,
					container.NewHBox(
						layout.NewSpacer(),
						linkPermissions,
						layout.NewSpacer(),
					),
				),
			),
		),
		container.NewStack(
			rectWidth90,
			container.NewVBox(
				rectSpacer,
				rectSpacer,
				container.NewHBox(
					layout.NewSpacer(),
					line1,
					layout.NewSpacer(),
					labelConnections,
					layout.NewSpacer(),
					line2,
					layout.NewSpacer(),
				),
				rectSpacer,
				rectSpacer,
				container.NewCenter(
					container.NewStack(
						rect,
						cyberdeck.WS.list,
					),
				),
			),
		),
		layout.NewSpacer(),
	)

	rpcForm := container.NewVBox(
		rectSpacer,
		rectSpacer,
		container.NewHBox(
			layout.NewSpacer(),
			line1,
			layout.NewSpacer(),
			rpcLabel,
			layout.NewSpacer(),
			line2,
			layout.NewSpacer(),
		),
		container.NewCenter(
			layout.NewSpacer(),
			container.NewCenter(
				container.NewVBox(
					rectWidth90,
					rectSpacer,
					linkCenter,
					rectSpacer,
					serverStatus,
					wSpacer,
					cyberdeck.RPC.toggle,
					wSpacer,
					cyberdeck.RPC.portText,
					rectSpacer,
					cyberdeck.RPC.userText,
					rectSpacer,
					cyberdeck.RPC.passText,
					wSpacer,
					container.NewHBox(
						layout.NewSpacer(),
						linkCopy,
						layout.NewSpacer(),
					),
				),
			),
			layout.NewSpacer(),
		),
	)

	deckFeatures := container.NewStack()
	if cyberdeck.RPC.server != nil {
		deckFeatures.Add(rpcForm)
		deckChoice.SetSelectedIndex(1)
	} else {
		deckFeatures.Add(xswdForm)
		deckChoice.SetSelectedIndex(0)
	}

	deckChoice.OnChanged = func(s string) {
		if s == "Remote Procedure Calls (RPC)" {
			deckFeatures.Objects[0] = rpcForm
		} else {
			deckFeatures.Objects[0] = xswdForm
		}
	}

	deckForm := container.NewVScroll(
		container.NewStack(
			container.NewVBox(
				rectSpacer,
				rectSpacer,
				container.NewCenter(
					container.NewVBox(
						title,
					),
				),
				rectSpacer,
				rectSpacer,
				container.NewCenter(
					container.NewStack(
						rectWidth90,
						deckChoice,
					),
				),
				container.NewBorder(
					deckFeatures,
					nil,
					nil,
					nil,
				),
			),
		),
	)

	deckForm.SetMinSize(fyne.NewSize(ui.MaxWidth*0.99, ui.MaxHeight*0.80))

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
		deckForm,
		subContainer,
		nil,
		nil,
	)

	layout := container.NewStack(
		frame,
		c,
	)

	return NewVScroll(layout)
}

// Layout details of an app connected through web socket
func layoutXSWDAppManager(ad *xswd.ApplicationData) fyne.CanvasObject {
	session.Domain = "app.cyberdeck.manager"

	frame := &iframe{}

	rectBox := canvas.NewRectangle(color.Transparent)
	rectBox.SetMinSize(fyne.NewSize(ui.MaxWidth*0.99, ui.MaxHeight*0.58))

	rectWidth90 := canvas.NewRectangle(color.Transparent)
	rectWidth90.SetMinSize(fyne.NewSize(ui.Width, 10))

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(6, 5))

	labelName := widget.NewRichTextFromMarkdown(ad.Name)
	labelName.Wrapping = fyne.TextWrapOff
	labelName.ParseMarkdown("## " + ad.Name)

	labelDesc := widget.NewRichTextFromMarkdown(ad.Description)
	labelDesc.Wrapping = fyne.TextWrapWord

	labelID := canvas.NewText("   APP  ID", colors.Gray)
	labelID.TextSize = 14
	labelID.Alignment = fyne.TextAlignLeading
	labelID.TextStyle = fyne.TextStyle{Bold: true}

	textID := widget.NewRichTextFromMarkdown(ad.Id)
	textID.Wrapping = fyne.TextWrapWord

	labelSignature := canvas.NewText("   SIGNATURE", colors.Gray)
	labelSignature.TextSize = 14
	labelSignature.Alignment = fyne.TextAlignLeading
	labelSignature.TextStyle = fyne.TextStyle{Bold: true}

	textSignature := widget.NewRichTextFromMarkdown("")
	textSignature.Wrapping = fyne.TextWrapWord

	labelURL := canvas.NewText("   URL", colors.Gray)
	labelURL.TextSize = 14
	labelURL.Alignment = fyne.TextAlignLeading
	labelURL.TextStyle = fyne.TextStyle{Bold: true}

	textURL := widget.NewRichTextFromMarkdown(ad.Url)
	textURL.Wrapping = fyne.TextWrapWord

	labelPermissions := canvas.NewText("   PERMISSIONS", colors.Gray)
	labelPermissions.TextSize = 14
	labelPermissions.Alignment = fyne.TextAlignLeading
	labelPermissions.TextStyle = fyne.TextStyle{Bold: true}

	labelEvents := canvas.NewText("   EVENTS", colors.Gray)
	labelEvents.TextSize = 14
	labelEvents.Alignment = fyne.TextAlignLeading
	labelEvents.TextStyle = fyne.TextStyle{Bold: true}

	labelSeparator := widget.NewRichTextFromMarkdown("")
	labelSeparator.Wrapping = fyne.TextWrapOff
	labelSeparator.ParseMarkdown("---")
	labelSeparator2 := widget.NewRichTextFromMarkdown("")
	labelSeparator2.Wrapping = fyne.TextWrapOff
	labelSeparator2.ParseMarkdown("---")
	labelSeparator3 := widget.NewRichTextFromMarkdown("")
	labelSeparator3.Wrapping = fyne.TextWrapOff
	labelSeparator3.ParseMarkdown("---")
	labelSeparator4 := widget.NewRichTextFromMarkdown("")
	labelSeparator4.Wrapping = fyne.TextWrapOff
	labelSeparator4.ParseMarkdown("---")
	labelSeparator5 := widget.NewRichTextFromMarkdown("")
	labelSeparator5.Wrapping = fyne.TextWrapOff
	labelSeparator5.ParseMarkdown("---")
	labelSeparator6 := widget.NewRichTextFromMarkdown("")
	labelSeparator6.Wrapping = fyne.TextWrapOff
	labelSeparator6.ParseMarkdown("---")

	signatureItems := container.NewVBox(
		labelSeparator2,
		rectSpacer,
		rectSpacer,
		labelSignature,
		textSignature,
		rectSpacer,
		rectSpacer,
	)

	// Show signature result if one exists
	signatureItems.Hide()
	if len(ad.Signature) > 0 {
		signatureItems.Show()
		_, message, err := engram.Disk.CheckSignature(ad.Signature)
		if err != nil {
			textSignature.ParseMarkdown(err.Error())
		} else {
			textSignature.ParseMarkdown(strings.TrimSpace(string(message)))
		}
	}

	// Find Permissions for connected app and build UI object
	var methods []string
	for k := range ad.Permissions {
		methods = append(methods, k)
	}

	permissionItems := container.NewVBox()

	permissions := []string{
		xswd.Ask.String(),
		xswd.Allow.String(),
		xswd.Deny.String(),
		xswd.AlwaysAllow.String(),
		xswd.AlwaysDeny.String(),
	}

	if len(methods) > 0 {
		sort.Strings(methods)
		for _, name := range methods {
			permission := widget.NewSelect(permissions, nil)
			permission.SetSelected(ad.Permissions[name].String())
			permission.Disable()
			permissionItems.Add(container.NewBorder(nil, nil, widget.NewRichTextFromMarkdown("### "+name), permission))
		}
	} else {
		permissionItems.Add(container.NewBorder(nil, nil, widget.NewRichTextFromMarkdown("No Permissions"), nil))
	}

	// Find RegisteredEvents for connected app and build UI object
	var events []rpc.EventType
	for k := range ad.RegisteredEvents {
		events = append(events, k)
	}

	eventItems := container.NewVBox()

	if len(events) > 0 {
		sort.Slice(events, func(i, j int) bool { return events[i] < events[j] })
		for _, name := range events {
			event := widget.NewSelect([]string{"false", "true"}, nil)
			event.SetSelected(strconv.FormatBool(ad.RegisteredEvents[name]))
			event.Disable()
			eventItems.Add(container.NewBorder(nil, nil, widget.NewRichTextFromMarkdown(fmt.Sprintf("### %s", name)), event))
		}
	} else {
		eventItems.Add(container.NewBorder(nil, nil, widget.NewRichTextFromMarkdown("No Events"), nil))
	}

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

	linkBack := widget.NewHyperlinkWithStyle("Back to Cyberdeck", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkBack.OnTapped = func() {
		removeOverlays()
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutCyberdeck())
	}

	image := canvas.NewImageFromResource(resourceWebsocketPng)
	image.SetMinSize(fyne.NewSize(ui.Width*0.15, ui.Width*0.15))
	image.FillMode = canvas.ImageFillContain

	// if icon != "" {
	// 	var path fyne.Resource
	// 	path, err = fyne.LoadResourceFromURLString(icon)
	// 	if err != nil {
	// 		image.Resource = resourceBlockGrayPng
	// 	} else {
	// 		image.Resource = path
	// 	}

	// 	image.SetMinSize(fyne.NewSize(ui.Width*0.2, ui.Width*0.2))
	// 	image.FillMode = canvas.ImageFillContain
	// 	image.Refresh()
	// }

	// if name == "" {
	// 	labelName.ParseMarkdown("## No name provided")
	// }

	// if desc == "" {
	// 	labelDesc.ParseMarkdown("No description provided")
	// }

	linkURL := widget.NewHyperlinkWithStyle("Open in browser", nil, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	linkURL.OnTapped = func() {
		link, err := url.Parse(ad.Url)
		if err != nil {
			logger.Errorf("[Engram] Error parsing XSWD application URL: %s\n", err)
			return
		}
		_ = fyne.CurrentApp().OpenURL(link)
	}

	btnRemove := widget.NewButton("Remove", nil)
	btnRemove.OnTapped = func() {
		if cyberdeck.WS.server != nil && len(cyberdeck.WS.apps) > 0 {
			cyberdeck.WS.server.RemoveApplication(ad)
			removeOverlays()
			session.LastDomain = session.Window.Content()
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutCyberdeck())
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
						labelSeparator,
						rectSpacer,
						rectSpacer,
						labelID,
						textID,
						rectSpacer,
						rectSpacer,
						signatureItems,
						labelSeparator3,
						rectSpacer,
						rectSpacer,
						labelURL,
						rectSpacer,
						textURL,
						container.NewHBox(
							layout.NewSpacer(),
						),
						container.NewHBox(
							linkURL,
							layout.NewSpacer(),
						),
						rectSpacer,
						rectSpacer,
						labelSeparator4,
						rectSpacer,
						rectSpacer,
						labelPermissions,
						rectSpacer,
						container.NewHBox(
							layout.NewSpacer(),
						),
						permissionItems,
						rectSpacer,
						rectSpacer,
						labelSeparator5,
						rectSpacer,
						rectSpacer,
						labelEvents,
						rectSpacer,
						eventItems,
						container.NewStack(
							rectWidth90,
						),
						rectSpacer,
						rectSpacer,
						labelSeparator6,
						rectSpacer,
						rectSpacer,
						btnRemove,
						rectSpacer,
						rectSpacer,
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

	return NewVScroll(layout)
}

// Layout XSWD permissions settings
func layoutXSWDPermissions() fyne.CanvasObject {
	session.Domain = "app.cyberdeck.permissions"

	wSpacer := widget.NewLabel(" ")

	frame := &iframe{}

	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(fyne.NewSize(ui.Width, 20))

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(10, 5))

	rectWidth90 := canvas.NewRectangle(color.Transparent)
	rectWidth90.SetMinSize(fyne.NewSize(ui.Width, 0))

	title := canvas.NewText("G L O B A L   P E R M I S S I O N S", colors.Gray)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 16

	xswdLabel := canvas.NewText("W E B   S O C K E T S", colors.Gray)
	xswdLabel.TextSize = 11
	xswdLabel.Alignment = fyne.TextAlignCenter
	xswdLabel.TextStyle = fyne.TextStyle{Bold: true}

	labelMethods := canvas.NewText("  METHODS", colors.Gray)
	labelMethods.TextSize = 14
	labelMethods.Alignment = fyne.TextAlignLeading
	labelMethods.TextStyle = fyne.TextStyle{Bold: true}

	labelConnection := canvas.NewText("  CONNECTIONS", colors.Gray)
	labelConnection.TextSize = 14
	labelConnection.Alignment = fyne.TextAlignLeading
	labelConnection.TextStyle = fyne.TextStyle{Bold: true}

	labelEpoch := canvas.NewText("  EPOCH", colors.Gray)
	labelEpoch.TextSize = 14
	labelEpoch.Alignment = fyne.TextAlignLeading
	labelEpoch.TextStyle = fyne.TextStyle{Bold: true}

	permissionInfo := canvas.NewText("APPLY ON CONNECTION", colors.Gray)
	permissionInfo.TextSize = 12
	permissionInfo.Alignment = fyne.TextAlignCenter
	permissionInfo.TextStyle = fyne.TextStyle{Bold: true}

	btnDefaults := widget.NewButton("Restore Defaults", nil)

	wMode := widget.NewCheck("Restrictive Mode", nil)

	wConnection := widget.NewSelect([]string{xswd.Ask.String(), xswd.Allow.String()}, nil)

	wGlobalPermissions := widget.NewSelect([]string{"Off", "Apply"}, nil)

	wEpoch := widget.NewSelect([]string{xswd.Deny.String(), xswd.Allow.String()}, nil)

	wEpochAddress := widget.NewSelect([]string{"My Address", "dApp Chooses"}, nil)

	if cyberdeck.EPOCH.enabled {
		wEpoch.SetSelectedIndex(1)
	} else {
		wEpoch.SetSelectedIndex(0)
		wEpochAddress.Disable()
	}

	if cyberdeck.EPOCH.allowWithAddress {
		wEpochAddress.SetSelectedIndex(1)
	} else {
		wEpochAddress.SetSelectedIndex(0)
	}

	wEpoch.OnChanged = func(s string) {
		if s == xswd.Allow.String() {
			cyberdeck.EPOCH.enabled = true
			wEpochAddress.Enable()
			return
		}

		cyberdeck.EPOCH.enabled = false
		wEpochAddress.SetSelectedIndex(0)
		wEpochAddress.Disable()
	}

	wEpochAddress.OnChanged = func(s string) {
		if s == "dApp Chooses" {
			cyberdeck.EPOCH.allowWithAddress = true
			return
		}

		cyberdeck.EPOCH.allowWithAddress = false
	}

	spacerEpoch := canvas.NewRectangle(color.Transparent)
	spacerEpoch.SetMinSize(fyne.NewSize(140, 0))

	entryEpochWork := widget.NewEntry()
	entryEpochWork.SetPlaceHolder(":10100")
	entryEpochWork.SetText(epoch.GetPort())
	entryEpochWork.Validator = func(s string) (err error) {
		i, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("invalid port")
		}

		return epoch.SetPort(i)
	}

	entryEpochHash := widget.NewEntry()
	entryEpochHash.SetPlaceHolder("Max hashes")
	entryEpochHash.SetText(strconv.Itoa(epoch.GetMaxHashes()))
	entryEpochHash.Validator = func(s string) (err error) {
		i, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("invalid hash value")
		}

		return epoch.SetMaxHashes(i)
	}

	wEpochPower := widget.NewSelect([]string{"Less", "More"}, nil)
	wEpochPower.SetSelectedIndex(0)
	if epoch.GetMaxThreads() > 2 {
		wEpochPower.SetSelectedIndex(1)
	}

	wEpochPower.OnChanged = func(s string) {
		if s == "More" {
			half := runtime.NumCPU() / 2
			if half > epoch.DEFAULT_MAX_THREADS {
				epoch.SetMaxThreads(half)
			}

			return
		}

		epoch.SetMaxThreads(epoch.DEFAULT_MAX_THREADS)
	}

	if session.Offline {
		wMode.Disable()
		wEpoch.Disable()
		wEpochAddress.Disable()
		entryEpochWork.Disable()
		entryEpochHash.Disable()
		wEpochPower.Disable()
	} else if cyberdeck.WS.server != nil {
		wEpoch.Disable()
		wEpochAddress.Disable()
		entryEpochWork.Disable()
		entryEpochHash.Disable()
		wEpochPower.Disable()
	}

	if cyberdeck.WS.advanced {
		wMode.SetChecked(false)
		if cyberdeck.WS.global.enabled {
			wGlobalPermissions.SetSelectedIndex(1)
			if cyberdeck.WS.global.connect {
				wConnection.SetSelectedIndex(1)
			} else {
				wConnection.SetSelectedIndex(0)
			}
		} else {
			wGlobalPermissions.SetSelectedIndex(0)
			wConnection.SetSelectedIndex(0)
			wConnection.Disable()
			btnDefaults.Disable()
		}
	} else {
		wMode.SetChecked(true)
		wConnection.SetSelectedIndex(0)
		wConnection.Disable()
		wGlobalPermissions.SetSelectedIndex(0)
		wGlobalPermissions.Disable()
		btnDefaults.Disable()
	}

	wMode.OnChanged = func(b bool) {
		cyberdeck.WS.advanced = !b // inverse as check box is for restrictive mode on/off
		if cyberdeck.WS.advanced {
			wGlobalPermissions.Enable()
		} else {
			wGlobalPermissions.SetSelectedIndex(0) // calling this here resets and disables wConnection
			wGlobalPermissions.Disable()
		}
	}

	wConnection.OnChanged = func(s string) {
		if s == xswd.Allow.String() {
			cyberdeck.WS.global.connect = true
		} else {
			cyberdeck.WS.global.connect = false
		}
	}

	formItems := container.NewVBox()

	permissions := []string{
		xswd.Ask.String(),
		xswd.AlwaysAllow.String(),
		xswd.AlwaysDeny.String(),
	}

	noStorePermissions := []string{
		xswd.Ask.String(),
		xswd.AlwaysDeny.String(),
	}

	// Permissions select on changed func
	onChanged := func(n string) func(s string) {
		return func(s string) {
			cyberdeck.WS.Lock()
			defer cyberdeck.WS.Unlock()

			switch s {
			case xswd.Ask.String():
				cyberdeck.WS.global.permissions[n] = xswd.Ask
			case xswd.AlwaysAllow.String():
				cyberdeck.WS.global.permissions[n] = xswd.AlwaysAllow
			case xswd.AlwaysDeny.String():
				cyberdeck.WS.global.permissions[n] = xswd.AlwaysDeny
			default:
				cyberdeck.WS.global.permissions[n] = xswd.Ask
			}
		}
	}

	stored, methods := getPermissions()
	for _, name := range methods {
		n := name
		permission := widget.NewSelect([]string{}, nil)
		if engramCanStoreMethod(n) {
			permission.SetOptions(permissions)
		} else {
			permission.SetOptions(noStorePermissions)
		}

		if cyberdeck.WS.global.enabled {
			permission.SetSelected(stored[n].String())
			permission.OnChanged = onChanged(n)
		} else {
			permission.SetSelectedIndex(0)
			permission.Disable()
		}
		formItems.Add(container.NewBorder(nil, nil, widget.NewRichTextFromMarkdown("### "+n), permission))
	}

	statusText := "Disabled"
	statusColor := colors.Gray
	if cyberdeck.WS.global.enabled {
		statusText = "Enabled"
		statusColor = colors.Green
	}

	cyberdeck.WS.global.status = canvas.NewText(statusText, statusColor)
	cyberdeck.WS.global.status.TextSize = 22
	cyberdeck.WS.global.status.TextStyle = fyne.TextStyle{Bold: true}

	btnDefaults.OnTapped = func() {
		if !cyberdeck.WS.global.enabled {
			return
		}

		header := canvas.NewText("RESTORE  DEFAULT  PERMISSIONS", colors.Gray)
		header.TextSize = 14
		header.Alignment = fyne.TextAlignCenter
		header.TextStyle = fyne.TextStyle{Bold: true}

		subHeader := canvas.NewText("Are you sure?", colors.Account)
		subHeader.TextSize = 22
		subHeader.Alignment = fyne.TextAlignCenter
		subHeader.TextStyle = fyne.TextStyle{Bold: true}

		linkCancel := widget.NewHyperlinkWithStyle("Cancel", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		linkCancel.OnTapped = func() {
			removeOverlays()
		}

		btnSubmit := widget.NewButton("Restore Defaults", nil)
		btnSubmit.OnTapped = func() {
			wConnection.SetSelectedIndex(0)
			for _, obj := range formItems.Objects {
				obj.(*fyne.Container).Objects[1].(*widget.Select).SetSelectedIndex(0)
			}
			removeOverlays()
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
							linkCancel,
							layout.NewSpacer(),
						),
						rectSpacer,
						rectSpacer,
					),
				),
			),
		)
	}

	wGlobalPermissions.OnChanged = func(s string) {
		if s != "Apply" {
			setPermissions()
			btnDefaults.Disable()
			cyberdeck.WS.global.status.Text = "Disabled"
			cyberdeck.WS.global.status.Color = colors.Gray
			cyberdeck.WS.global.status.Refresh()
			cyberdeck.WS.global.enabled = false
			wConnection.SetSelectedIndex(0)
			wConnection.Disable()
			for _, obj := range formItems.Objects {
				obj.(*fyne.Container).Objects[1].(*widget.Select).OnChanged = nil
				obj.(*fyne.Container).Objects[1].(*widget.Select).SetSelectedIndex(0)
				obj.(*fyne.Container).Objects[1].(*widget.Select).Disable()
			}
		} else {
			cyberdeck.WS.global.status.Text = "Enabled"
			cyberdeck.WS.global.status.Color = colors.Green
			cyberdeck.WS.global.status.Refresh()
			cyberdeck.WS.global.enabled = true
			wConnection.Enable()
			btnDefaults.Enable()
			go func() {
				stored, _ := getPermissions()
				for _, obj := range formItems.Objects {
					name := obj.(*fyne.Container).Objects[0].(*widget.RichText).String()
					obj.(*fyne.Container).Objects[1].(*widget.Select).SetSelected(stored[name].String())
					obj.(*fyne.Container).Objects[1].(*widget.Select).Enable()
					obj.(*fyne.Container).Objects[1].(*widget.Select).OnChanged = onChanged(name)
				}
			}()
		}
	}

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

	linkBack := widget.NewHyperlinkWithStyle("Back to Cyberdeck", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkBack.OnTapped = func() {
		setPermissions()
		removeOverlays()
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutCyberdeck())
	}

	// Initialized in layoutCyberdeck()
	cyberdeck.WS.portText.SetText(getCyberdeck("WS"))

	center := container.NewVScroll(
		container.NewStack(
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
				container.NewHBox(
					layout.NewSpacer(),
					line1,
					layout.NewSpacer(),
					xswdLabel,
					layout.NewSpacer(),
					line2,
					layout.NewSpacer(),
				),
				container.NewCenter(
					container.NewVBox(
						rectWidth90,
						rectSpacer,
						container.NewCenter(
							cyberdeck.WS.global.status,
						),
						rectSpacer,
						container.NewCenter(
							permissionInfo,
						),
					),
				),
				rectSpacer,
				container.NewHBox(
					layout.NewSpacer(),
					container.NewCenter(
						container.NewVBox(
							container.NewBorder(
								nil,
								nil,
								nil,
								nil,
								container.NewCenter(wMode),
							),
							rectSpacer,
							cyberdeck.WS.portText,
							rectSpacer,
							labelConnection,
							rectSpacer,
							container.NewBorder(
								nil,
								nil,
								widget.NewRichTextFromMarkdown("### Type"),
								wConnection,
							),
							container.NewBorder(
								nil,
								nil,
								widget.NewRichTextFromMarkdown("### Global Permissions"),
								wGlobalPermissions,
							),
							wSpacer,
							labelEpoch,
							rectSpacer,
							container.NewBorder(
								nil,
								nil,
								widget.NewRichTextFromMarkdown("### Preference"),
								wEpoch,
							),
							container.NewBorder(
								nil,
								nil,
								widget.NewRichTextFromMarkdown("### Reward Address"),
								wEpochAddress,
							),
							container.NewBorder(
								nil,
								nil,
								widget.NewRichTextFromMarkdown("### Get Work"),
								container.NewHBox(
									layout.NewSpacer(),
									container.NewStack(
										spacerEpoch,
										entryEpochWork,
									),
								),
							),
							container.NewBorder(
								nil,
								nil,
								widget.NewRichTextFromMarkdown("### Max Hashes"),
								container.NewHBox(
									layout.NewSpacer(),
									container.NewStack(
										spacerEpoch,
										entryEpochHash,
									),
								),
							),
							container.NewBorder(
								nil,
								nil,
								widget.NewRichTextFromMarkdown("### Power"),
								wEpochPower,
							),
							wSpacer,
							labelMethods,
							rectSpacer,
							container.NewCenter(
								formItems,
							),
							wSpacer,
						),
					),
					layout.NewSpacer(),
				),
				container.NewCenter(
					container.NewVBox(
						btnDefaults,
						rectWidth90,
					),
				),
				wSpacer,
			),
		),
	)
	center.SetMinSize(fyne.NewSize(ui.MaxWidth*0.99, ui.MaxHeight*0.80))

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
			center,
			bottom,
			nil,
			nil,
		),
	)

	return NewVScroll(layout)
}

func layoutIdentity() fyne.CanvasObject {
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
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutDashboard())
		removeOverlays()
	}

	//entryReg := NewMobileEntry()
	entryReg := widget.NewEntry()
	entryReg.MultiLine = false
	entryReg.Wrapping = fyne.TextWrap(fyne.TextTruncateClip)

	userData, err := queryUsernames(engram.Disk.GetAddress().String())
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
			valid, _ := checkUsername(session.NewUser, -1)
			if valid == "" {
				btnReg.Text = "Confirming..."
				btnReg.Disable()
				btnReg.Refresh()
				entryReg.Disable()
				storage, err := registerUsername(session.NewUser)
				if err != nil {
					if strings.Contains(err.Error(), "somehow the tx could not be built") {
						btnReg.Text = fmt.Sprintf("Insufficient Balance: Need %v", globals.FormatMoney(storage))
					} else {
						btnReg.Text = "Unable to register..."
					}
					btnReg.Refresh()
					logger.Errorf("[Username] %s\n", err)
				} else {
					go func() {
						entryReg.Text = ""
						entryReg.Refresh()
						walletapi.WaitNewHeightBlock()
						sHeight := walletapi.Get_Daemon_Height()

						for {
							if session.Domain == "app.Identity" {
								//vars, _, _, err := gnomon.Index.RPC.GetSCVariables("0000000000000000000000000000000000000000000000000000000000000001", engram.Disk.Get_Daemon_TopoHeight(), nil, []string{session.NewUser}, nil, false)
								usernames, err := queryUsernames(engram.Disk.GetAddress().String())
								if err != nil {
									logger.Errorf("[Username] Error querying usernames: %s\n", err)
									btnReg.Text = "Error querying usernames"
									btnReg.Refresh()
									return
								}

								for u := range usernames {
									if usernames[u] == session.NewUser {
										logger.Printf("[Username] Successfully registered username: %s\n", session.NewUser)
										_ = tx
										btnReg.Text = "Registration successful!"
										btnReg.Refresh()
										session.NewUser = ""
										session.Window.SetContent(layoutIdentity())
										return
									}
								}

								// If we go DEFAULT_CONFIRMATION_TIMEOUT blocks without exiting 'Confirming...' loop, display failed to transfer and break
								if walletapi.Get_Daemon_Height() > sHeight+int64(DEFAULT_CONFIRMATION_TIMEOUT) {
									btnReg.Text = "Unable to register..."
									btnReg.Refresh()
									break
								}

								// If daemon height has incremented, print retry counters into button space
								if walletapi.Get_Daemon_Height()-sHeight > 0 {
									btnReg.Text = fmt.Sprintf("Confirming... (%d/%d)", walletapi.Get_Daemon_Height()-sHeight, DEFAULT_CONFIRMATION_TIMEOUT)
									btnReg.Refresh()
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
	}

	entryReg.PlaceHolder = "New Username"
	entryReg.Validator = func(s string) error {
		btnReg.Text = " Register "
		btnReg.Enable()
		btnReg.Refresh()
		session.NewUser = s
		// Name Service SCID Logic
		//	15  IF STRLEN(name) >= 64 THEN GOTO 50 // skip names misuse
		//	20  IF STRLEN(name) >= 6 THEN GOTO 40
		if len(s) > 5 && len(s) < 64 {
			valid, _ := checkUsername(s, -1)
			if valid == "" {
				btnReg.Enable()
				btnReg.Refresh()
			} else {
				btnReg.Disable()
				err := errors.New("username already exists")
				entryReg.SetValidationError(err)
				btnReg.Refresh()
				return err
			}
		} else {
			btnReg.Disable()
			err := errors.New("username too short need a minimum of six characters")
			entryReg.SetValidationError(err)
			btnReg.Refresh()
			return err
		}

		return nil
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

			if len(str) > DEFAULT_USERADDR_SHORTEN_LENGTH+3 {
				str = "..." + str[len(str)-DEFAULT_USERADDR_SHORTEN_LENGTH:]
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

	dispUsername := session.Username
	if len(session.Username) > DEFAULT_USERADDR_SHORTEN_LENGTH+3 {
		dispUsername = "..." + dispUsername[len(dispUsername)-DEFAULT_USERADDR_SHORTEN_LENGTH:]
	}

	textUsername := canvas.NewText(dispUsername, colors.Green)
	textUsername.TextStyle = fyne.TextStyle{Bold: true}
	textUsername.TextSize = 22

	if session.Username == "" {
		textUsername.Text = "---"
		textUsername.Refresh()
	} /* else {
		for u := range userData {
			if userData[u] == session.Username {
				userBox.Select(u)
				userBox.ScrollTo(u)
			}
		}
	}*/

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

	return NewVScroll(layout)
}

func layoutIdentityDetail(username string) fyne.CanvasObject {
	var address string

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
		//session.Window.SetContent(layoutIdentity())
		removeOverlays()
	}

	btnSend := widget.NewButton("Transfer Username", nil)

	inputAddress := widget.NewEntry()
	inputAddress.PlaceHolder = "Receiver Username or Address"
	inputAddress.Validator = func(s string) error {
		btnSend.Text = "Transfer Username"
		btnSend.Enable()
		btnSend.Refresh()
		address, _ = checkUsername(s, -1)
		if address == "" {
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
			btnSetPrimary.Disable()
			storage, err := transferUsername(username, address)
			if err != nil {
				address = ""
				if strings.Contains(err.Error(), "somehow the tx could not be built") {
					btnSend.Text = fmt.Sprintf("Insufficient Balance: Need %v", globals.FormatMoney(storage))
				} else {
					btnSend.Text = "Transfer failed..."
				}
				btnSend.Disable()
				btnSend.Refresh()
				inputAddress.Enable()
				inputAddress.Refresh()
				btnSetPrimary.Enable()
			} else {
				btnSend.Text = "Confirming..."
				btnSend.Refresh()
				go func() {
					walletapi.WaitNewHeightBlock()
					sHeight := walletapi.Get_Daemon_Height()

					for {
						found := false
						if session.Domain == "app.Identity" {
							usernames, err := queryUsernames(engram.Disk.GetAddress().String())
							if err != nil {
								logger.Errorf("[Username] Error querying usernames: %s\n", err)
								btnSend.Text = "Error querying usernames"
								btnSend.Refresh()
								btnSetPrimary.Enable()
								return
							}

							for u := range usernames {
								if usernames[u] == username {
									found = true
								}
							}

							if !found {
								logger.Printf("[TransferOwnership] %s was successfully transferred to: %s\n", username, address)
								session.Window.SetContent(layoutTransition())
								session.Window.SetContent(layoutIdentity())
								removeOverlays()
								break
							}

							// If we go DEFAULT_CONFIRMATION_TIMEOUT blocks without exiting 'Confirming...' loop, display failed to transfer and break
							if walletapi.Get_Daemon_Height() > sHeight+int64(DEFAULT_CONFIRMATION_TIMEOUT) {
								logger.Errorf("[TransferOwnership] %s was unsuccessful in transferring to: %s\n", username, address)
								btnSend.Text = "Unable to transfer..."
								btnSend.Refresh()
								btnSetPrimary.Enable()
								break
							}

							// If daemon height has incremented, print retry counters into button space
							if walletapi.Get_Daemon_Height()-sHeight > 0 {
								btnSend.Text = fmt.Sprintf("Confirming... (%d/%d)", walletapi.Get_Daemon_Height()-sHeight, DEFAULT_CONFIRMATION_TIMEOUT)
								btnSend.Refresh()
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

	return NewVScroll(layout)
}

func layoutAlert(t int) fyne.CanvasObject {
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
		sub.ParseMarkdown("Could not write data to disk, please check to make sure Engram has the proper permissions and/or you have unzipped the contents.")
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

	return NewVScroll(layout)
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
		session.LastDomain = session.Window.Content()
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
						var zeroscid crypto.Hash
						_, result := engram.Disk.Get_Payments_TXID(zeroscid, split[4])

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

	return NewVScroll(layout)
}

func layoutHistoryDetail(txid string) fyne.CanvasObject {
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

	labelTXID := canvas.NewText("   TRANSACTION  ID", colors.Gray)
	labelTXID.TextSize = 14
	labelTXID.Alignment = fyne.TextAlignLeading
	labelTXID.TextStyle = fyne.TextStyle{Bold: true}

	labelAmount := canvas.NewText("   AMOUNT", colors.Gray)
	labelAmount.TextSize = 14
	labelAmount.Alignment = fyne.TextAlignLeading
	labelAmount.TextStyle = fyne.TextStyle{Bold: true}

	labelDirection := canvas.NewText("   PAYMENT  DIRECTION", colors.Gray)
	labelDirection.TextSize = 14
	labelDirection.Alignment = fyne.TextAlignLeading
	labelDirection.TextStyle = fyne.TextStyle{Bold: true}

	labelMember := canvas.NewText("", colors.Gray)
	labelMember.TextSize = 14
	labelMember.Alignment = fyne.TextAlignLeading
	labelMember.TextStyle = fyne.TextStyle{Bold: true}

	labeliMember := canvas.NewText("", colors.Gray)
	labeliMember.TextSize = 14
	labeliMember.Alignment = fyne.TextAlignLeading
	labeliMember.TextStyle = fyne.TextStyle{Bold: true}

	labelProof := canvas.NewText("   TRANSACTION  PROOF", colors.Gray)
	labelProof.TextSize = 14
	labelProof.Alignment = fyne.TextAlignLeading
	labelProof.TextStyle = fyne.TextStyle{Bold: true}

	labelDestPort := canvas.NewText("   DESTINATION  PORT", colors.Gray)
	labelDestPort.TextSize = 14
	labelDestPort.TextStyle = fyne.TextStyle{Bold: true}

	labelSourcePort := canvas.NewText("   SOURCE  PORT", colors.Gray)
	labelSourcePort.TextSize = 14
	labelSourcePort.TextStyle = fyne.TextStyle{Bold: true}

	labelFees := canvas.NewText("   TRANSACTION  FEES", colors.Gray)
	labelFees.TextSize = 14
	labelFees.TextStyle = fyne.TextStyle{Bold: true}

	labelPayload := canvas.NewText("   PAYLOAD", colors.Gray)
	labelPayload.TextSize = 14
	labelPayload.TextStyle = fyne.TextStyle{Bold: true}

	labelHeight := canvas.NewText("   BLOCK  HEIGHT", colors.Gray)
	labelHeight.TextSize = 14
	labelHeight.TextStyle = fyne.TextStyle{Bold: true}

	labelReply := canvas.NewText("   REPLY  ADDRESS", colors.Gray)
	labelReply.TextSize = 14
	labelReply.TextStyle = fyne.TextStyle{Bold: true}

	labelSeparator := widget.NewRichTextFromMarkdown("")
	labelSeparator.Wrapping = fyne.TextWrapOff
	labelSeparator.ParseMarkdown("---")

	labelSeparator2 := widget.NewRichTextFromMarkdown("")
	labelSeparator2.Wrapping = fyne.TextWrapOff
	labelSeparator2.ParseMarkdown("---")

	labelSeparator3 := widget.NewRichTextFromMarkdown("")
	labelSeparator3.Wrapping = fyne.TextWrapOff
	labelSeparator3.ParseMarkdown("---")

	labelSeparator4 := widget.NewRichTextFromMarkdown("")
	labelSeparator4.Wrapping = fyne.TextWrapOff
	labelSeparator4.ParseMarkdown("---")

	labelSeparator5 := widget.NewRichTextFromMarkdown("")
	labelSeparator5.Wrapping = fyne.TextWrapOff
	labelSeparator5.ParseMarkdown("---")

	labelSeparator6 := widget.NewRichTextFromMarkdown("")
	labelSeparator6.Wrapping = fyne.TextWrapOff
	labelSeparator6.ParseMarkdown("---")

	labelSeparator7 := widget.NewRichTextFromMarkdown("")
	labelSeparator7.Wrapping = fyne.TextWrapOff
	labelSeparator7.ParseMarkdown("---")

	labelSeparator8 := widget.NewRichTextFromMarkdown("")
	labelSeparator8.Wrapping = fyne.TextWrapOff
	labelSeparator8.ParseMarkdown("---")

	labelSeparator9 := widget.NewRichTextFromMarkdown("")
	labelSeparator9.Wrapping = fyne.TextWrapOff
	labelSeparator9.ParseMarkdown("---")

	labelSeparator10 := widget.NewRichTextFromMarkdown("")
	labelSeparator10.Wrapping = fyne.TextWrapOff
	labelSeparator10.ParseMarkdown("---")

	labelSeparator11 := widget.NewRichTextFromMarkdown("")
	labelSeparator11.Wrapping = fyne.TextWrapOff
	labelSeparator11.ParseMarkdown("---")

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

	var zeroscid crypto.Hash
	_, details := engram.Disk.Get_Payments_TXID(zeroscid, txid)

	stamp := string(details.Time.Format(time.RFC822))
	height := strconv.FormatUint(details.Height, 10)

	valueMember := widget.NewRichTextFromMarkdown(" ")
	valueMember.Wrapping = fyne.TextWrapBreak

	valueiMember := widget.NewRichTextFromMarkdown("--")
	valueiMember.Wrapping = fyne.TextWrapBreak

	valueReply := widget.NewRichTextFromMarkdown("--")
	valueReply.Wrapping = fyne.TextWrapBreak

	if details.Payload_RPC.HasValue(rpc.RPC_REPLYBACK_ADDRESS, rpc.DataAddress) {
		address := details.Payload_RPC.Value(rpc.RPC_REPLYBACK_ADDRESS, rpc.DataAddress).(rpc.Address)
		valueReply.ParseMarkdown("" + address.String())
	} else if details.Payload_RPC.HasValue(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString) && details.DestinationPort == 1337 {
		address := details.Payload_RPC.Value(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataString).(string)
		valueReply.ParseMarkdown("" + address)
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

	valueDirection := canvas.NewText("", colors.Account)
	valueDirection.TextSize = 22
	valueDirection.TextStyle = fyne.TextStyle{Bold: true}
	if details.Incoming {
		valueDirection.Text = "  Received"
		labelMember.Text = "  SENDER  ADDRESS"
		if details.Sender == "" || details.Sender == engram.Disk.GetAddress().String() {
			valueMember.ParseMarkdown("--")
		} else {
			valueMember.ParseMarkdown("" + details.Sender)
		}

		if details.Amount == 0 {
			valueAmount.Color = colors.Account
			valueAmount.Text = "  0.00000"
		} else {
			valueAmount.Color = colors.Green
			valueAmount.Text = "  + " + globals.FormatMoney(details.Amount)
		}
	} else {
		valueDirection.Text = "  Sent"
		labelMember.Text = "  RECEIVER  ADDRESS"
		valueMember.ParseMarkdown("" + details.Destination)

		if details.Amount == 0 {
			valueAmount.Color = colors.Account
			valueAmount.Text = "  0.00000"
		} else {
			valueAmount.Color = colors.Account
			valueAmount.Text = "  - " + globals.FormatMoney(details.Amount)
		}
	}

	labeliMember.Text = "  INTEGRATED  ADDRESS"
	var idest string
	if details.Destination == "" {
		// We are the recipient
		idest = engram.Disk.GetAddress().String()
	} else {
		idest = details.Destination
	}
	iaddr, _ := rpc.NewAddress(idest)
	if iaddr != nil {
		var iargs rpc.Arguments
		for _, v := range details.Payload_RPC {
			if !iargs.HasValue(v.Name, v.DataType) {
				// Skip the reply back addr that was injected, but 'reverse' this to be what the original payload was which requests the reply addr
				if v.Name == rpc.RPC_REPLYBACK_ADDRESS {
					iargs = append(iargs, rpc.Argument{Name: rpc.RPC_NEEDS_REPLYBACK_ADDRESS, DataType: rpc.DataUint64, Value: uint64(1)})
				} else {
					iargs = append(iargs, rpc.Argument{Name: v.Name, DataType: v.DataType, Value: v.Value})
				}
			}
		}

		// If value transfer 'V' doesn't exist, we add it here.
		if !iargs.HasValue(rpc.RPC_VALUE_TRANSFER, rpc.DataUint64) {
			iargs = append(iargs, rpc.Argument{Name: rpc.RPC_VALUE_TRANSFER, DataType: rpc.DataUint64, Value: details.Amount})
		}

		iaddr.Arguments = iargs

		// Check to see if integrated addr creation makes an actual integrated addr
		if iaddr.String() != details.Destination && iaddr.IsIntegratedAddress() {
			valueiMember.ParseMarkdown("" + iaddr.String())
		}
	}

	valueTime := canvas.NewText(stamp, colors.Account)
	valueTime.TextSize = 14
	valueTime.TextStyle = fyne.TextStyle{Bold: true}

	valueFees := canvas.NewText("  "+globals.FormatMoney(details.Fees), colors.Account)
	valueFees.TextSize = 22
	valueFees.TextStyle = fyne.TextStyle{Bold: true}

	valueHeight := canvas.NewText("  "+height, colors.Account)
	valueHeight.TextSize = 22
	valueHeight.TextStyle = fyne.TextStyle{Bold: true}

	valueTXID := widget.NewRichTextFromMarkdown("")
	valueTXID.Wrapping = fyne.TextWrapBreak
	valueTXID.ParseMarkdown("" + txid)

	valuePort := canvas.NewText("", colors.Account)
	valuePort.TextSize = 22
	valuePort.TextStyle = fyne.TextStyle{Bold: true}
	valuePort.Text = "  " + strconv.FormatUint(details.DestinationPort, 10)

	valueSourcePort := canvas.NewText("", colors.Account)
	valueSourcePort.TextSize = 22
	valueSourcePort.TextStyle = fyne.TextStyle{Bold: true}
	valueSourcePort.Text = "  " + strconv.FormatUint(details.SourcePort, 10)

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

	linkiAddress := widget.NewHyperlinkWithStyle("Copy Address", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkiAddress.OnTapped = func() {
		session.Window.Clipboard().SetContent(valueiMember.String())
	}

	linkReplyAddress := widget.NewHyperlinkWithStyle("Copy Address", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkReplyAddress.OnTapped = func() {
		if replyAddress, ok := details.Payload_RPC.Value(rpc.RPC_REPLYBACK_ADDRESS, rpc.DataAddress).(rpc.Address); ok {
			session.Window.Clipboard().SetContent(replyAddress.String())
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
						labelSeparator,
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
						labelSeparator2,
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
						labelSeparator3,
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
						labelSeparator4,
						rectSpacer,
						rectSpacer,
						labeliMember,
						rectSpacer,
						valueiMember,
						container.NewHBox(
							linkiAddress,
							layout.NewSpacer(),
						),
						rectSpacer,
						rectSpacer,
						labelSeparator5,
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
						labelSeparator6,
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
						labelSeparator7,
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
						labelSeparator8,
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
						labelSeparator9,
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
						labelSeparator10,
						rectSpacer,
						rectSpacer,
						labelSourcePort,
						rectSpacer,
						container.NewStack(
							rectWidth90,
							valueSourcePort,
						),
						rectSpacer,
						rectSpacer,
						labelSeparator11,
						rectSpacer,
						rectSpacer,
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
				err := errors.New("username already exists")
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
			err := errors.New("please enter a datapad name")
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
		session.LastDomain = session.Window.Content()
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

	return NewVScroll(layout)
}

func layoutPad() fyne.CanvasObject {
	rectWidth := canvas.NewRectangle(color.Transparent)
	rectWidth.SetMinSize(fyne.NewSize(ui.MaxWidth, 10))

	rectWidth90 := canvas.NewRectangle(color.Transparent)
	rectWidth90.SetMinSize(fyne.NewSize(ui.Width, 10))

	rectEntry := canvas.NewRectangle(color.Transparent)
	rectEntry.SetMinSize(fyne.NewSize(ui.Width, ui.Height*0.52))

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

	selectOptions := widget.NewSelect([]string{"Clear", "Export (Plaintext)", "Import From File", "Delete"}, nil)
	selectOptions.PlaceHolder = "Select an Option ..."

	data, err := GetEncryptedValue("Datapads", []byte(session.Datapad))
	if err != nil {
		data = nil
	}

	overlay := session.Window.Canvas().Overlays()

	btnSave := widget.NewButton("Save", nil)

	entryPad := widget.NewEntry()
	entryPad.Wrapping = fyne.TextWrapWord

	errorText := canvas.NewText(" ", colors.Green)
	errorText.TextSize = 12
	errorText.Alignment = fyne.TextAlignCenter

	selectOptions.OnChanged = func(s string) {
		errorText.Text = ""
		errorText.Refresh()

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
						logger.Errorf("[Datapad] Err: %s\n", err)
						selectOptions.Selected = "Select an Option ..."
						selectOptions.Refresh()
						return
					}

					selectOptions.Selected = "Select an Option ..."
					selectOptions.Refresh()
					entryPad.Text = ""
					entryPad.Refresh()
				}

				errorText.Text = "datapad cleared"
				errorText.Color = colors.Green
				errorText.Refresh()

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
			selectOptions.Selected = "Select an Option ..."
			selectOptions.Refresh()

			dialogFileSave := dialog.NewFileSave(func(uri fyne.URIWriteCloser, err error) {
				if err != nil {
					logger.Errorf("[Engram] File dialog: %s\n", err)
					errorText.Text = "could not export datapad"
					errorText.Color = colors.Red
					errorText.Refresh()
					return
				}

				if uri == nil {
					return // Canceled
				}

				data := []byte(entryPad.Text)
				_, err = writeToURI(data, uri)
				if err != nil {
					logger.Errorf("[Engram] Exporting datapad %s: %s\n", session.Datapad, err)
					errorText.Text = "error exporting datapad"
					errorText.Color = colors.Red
					errorText.Refresh()
					return
				}

				errorText.Text = "exported datapad successfully"
				errorText.Color = colors.Green
				errorText.Refresh()

			}, session.Window)

			if !a.Driver().Device().IsMobile() {
				// Open file browser in current directory
				uri, err := storage.ListerForURI(storage.NewFileURI(AppPath()))
				if err == nil {
					dialogFileSave.SetLocation(uri)
				} else {
					logger.Errorf("[Engram] Could not open current directory %s\n", err)
				}
			}

			// dialogFileSave.SetFilter(storage.NewMimeTypeFileFilter([]string{"text/*"}))
			dialogFileSave.SetView(dialog.ListView)
			dialogFileSave.SetFileName(fmt.Sprintf("%s.txt", session.Datapad))
			dialogFileSave.Resize(fyne.NewSize(ui.Width, ui.Height))
			dialogFileSave.Show()
		} else if s == "Import From File" {
			selectOptions.Selected = "Select an Option ..."
			selectOptions.Refresh()

			dialogFileImport := dialog.NewFileOpen(func(uri fyne.URIReadCloser, err error) {
				if err != nil {
					logger.Errorf("[Engram] File dialog: %s\n", err)
					errorText.Text = "could not import file"
					errorText.Color = colors.Red
					errorText.Refresh()
					return
				}

				if uri == nil {
					return // Canceled
				}

				fileName := uri.URI().String()
				if !strings.Contains(uri.URI().MimeType(), "text/") {
					logger.Errorf("[Engram] Cannot import file %s\n", fileName)
					errorText.Text = "cannot import file"
					errorText.Color = colors.Red
					errorText.Refresh()
					return
				}

				if a.Driver().Device().IsMobile() {
					fileName = uri.URI().Name()
				} else {
					fileName = filepath.Base(strings.Replace(fileName, "file://", "", -1))
				}

				filedata, err := readFromURI(uri)
				if err != nil {
					logger.Errorf("[Engram] Cannot read URI file data for %s: %s\n", fileName, err)
					errorText.Text = "cannot read file data"
					errorText.Color = colors.Red
					errorText.Refresh()
					return
				}

				if !isASCII(string(filedata)) {
					errorText.Text = "invalid file data"
					errorText.Color = colors.Red
					errorText.Refresh()
					return
				}

				if entryPad.Text == "" {
					entryPad.SetText(string(filedata))
				} else {
					entryPad.SetText(fmt.Sprintf("%s\n\n%s", entryPad.Text, string(filedata)))
				}

				errorText.Text = "file data imported successfully"
				errorText.Color = colors.Green
				errorText.Refresh()

			}, session.Window)

			if !a.Driver().Device().IsMobile() {
				// Open file browser in current directory
				uri, err := storage.ListerForURI(storage.NewFileURI(AppPath()))
				if err == nil {
					dialogFileImport.SetLocation(uri)
				} else {
					logger.Errorf("[Engram] Could not open current directory %s\n", err)
				}
			}

			// dialogFileSave.SetFilter(storage.NewMimeTypeFileFilter([]string{"text/*"}))
			dialogFileImport.SetView(dialog.ListView)
			dialogFileImport.Resize(fyne.NewSize(ui.Width, ui.Height))
			dialogFileImport.Show()
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
						selectOptions.Selected = "Select an Option ..."
						selectOptions.Refresh()
						logger.Errorf("[Datapad] Error deleting %s: %s\n", session.Datapad, err)
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
			btnSave.Disable()
			errorText.Text = "error saving datapad"
			errorText.Color = colors.Red
			errorText.Refresh()
		} else {
			session.DatapadChanged = false
			btnSave.Disable()
			heading.Text = session.Datapad
			heading.Refresh()
			errorText.Text = "datapad saved successfully"
			errorText.Color = colors.Green
			errorText.Refresh()
		}
	}

	session.DatapadChanged = false

	btnSave.Disable()

	entryPad.MultiLine = true
	entryPad.Text = string(data)
	entryPad.OnChanged = func(s string) {
		errorText.Text = ""
		errorText.Refresh()
		session.DatapadChanged = true
		heading.Text = session.Datapad + "*"
		heading.Refresh()
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
					btnSave.Disable()
					errorText.Text = "error saving datapad"
					errorText.Color = colors.Red
					errorText.Refresh()
					overlay.Remove(overlay.Top())
					overlay.Remove(overlay.Top())
				} else {
					session.Datapad = ""
					session.DatapadChanged = false
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
				errorText,
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
		nil,
		center,
	)

	return NewVScroll(layout)
}

func layoutAccount() fyne.CanvasObject {
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

	labelDatashard := canvas.NewText("D A T A S H A R D", colors.Gray)
	labelDatashard.TextStyle = fyne.TextStyle{Bold: true}
	labelDatashard.TextSize = 11
	labelDatashard.Alignment = fyne.TextAlignCenter

	headerDatashard := canvas.NewText("DATASHARD  ID", colors.Gray)
	headerDatashard.TextSize = 16
	headerDatashard.Alignment = fyne.TextAlignCenter
	headerDatashard.TextStyle = fyne.TextStyle{Bold: true}

	address := engram.Disk.GetAddress().String()
	shardID := fmt.Sprintf("%x", sha1.Sum([]byte(address)))

	textDatashard := widget.NewRichTextFromMarkdown("### " + shardID)
	textDatashard.Wrapping = fyne.TextWrapWord

	textDatashardDesc := widget.NewRichTextFromMarkdown("Datashards hold encrypted data and stores it locally on your device. Each datashard is unique and can only be decrypted by the account it is associated with. Examples of data stored include:")
	textDatashardDesc.Wrapping = fyne.TextWrapWord

	textDatashardDesc2 := widget.NewRichTextFromMarkdown("* Datapad entries\n* Saved search history\n* Asset scan results\n* Account settings")
	textDatashardDesc2.Wrapping = fyne.TextWrapWord

	menuLabel := canvas.NewText("  M O R E    O P T I O N S  ", colors.Gray)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	labelRecovery := canvas.NewText("A C C O U N T    O P T I O N S", colors.Gray)
	labelRecovery.TextSize = 11
	labelRecovery.Alignment = fyne.TextAlignCenter
	labelRecovery.TextStyle = fyne.TextStyle{Bold: true}

	labelEpoch := canvas.NewText("E P O C H", colors.Gray)
	labelEpoch.TextSize = 11
	labelEpoch.Alignment = fyne.TextAlignCenter
	labelEpoch.TextStyle = fyne.TextStyle{Bold: true}

	spacerEpoch := canvas.NewRectangle(color.Transparent)
	spacerEpoch.SetMinSize(fyne.NewSize(140, 0))

	wEpoch := widget.NewSelect([]string{"Session", "Total"}, nil)
	wEpoch.SetSelected("Session")

	epochSession, _ := epoch.GetSession(time.Second * 4)

	labelEpochHashes := widget.NewRichTextFromMarkdown("### Hashes")
	labelEpochHashes.Wrapping = fyne.TextWrapWord

	epochHashes := fmt.Sprintf("%.1fK", float64(epochSession.Hashes)/1000)
	textEpochHashes := widget.NewRichTextFromMarkdown(epochHashes)
	textEpochHashes.Wrapping = fyne.TextWrapWord

	labelEpochBlocks := widget.NewRichTextFromMarkdown("### Miniblocks")
	labelEpochBlocks.Wrapping = fyne.TextWrapWord

	epochBlocks := fmt.Sprintf("%d", epochSession.MiniBlocks)
	textEpochBlocks := widget.NewRichTextFromMarkdown(epochBlocks)
	textEpochBlocks.Wrapping = fyne.TextWrapWord

	wEpoch.OnChanged = func(s string) {
		epochSession, _ := epoch.GetSession(time.Second * 4)
		if s == "Total" {
			total := epoch.GetSessionEPOCH_Result{
				Hashes:     cyberdeck.EPOCH.total.Hashes,
				MiniBlocks: cyberdeck.EPOCH.total.MiniBlocks,
			}

			if epoch.IsActive() {
				total.Hashes += epochSession.Hashes
				total.MiniBlocks += epochSession.MiniBlocks
			}

			textEpochHashes.ParseMarkdown(epoch.HashesToString(total.Hashes))
			textEpochBlocks.ParseMarkdown(fmt.Sprintf("%d", total.MiniBlocks))

			return
		}

		textEpochHashes.ParseMarkdown(epoch.HashesToString(epochSession.Hashes))
		textEpochBlocks.ParseMarkdown(fmt.Sprintf("%d", epochSession.MiniBlocks))
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

	formEpoch := container.NewVBox(
		rectSpacer,
		rectSpacer,
		container.NewStack(
			container.NewHBox(
				layout.NewSpacer(),
				line1,
				layout.NewSpacer(),
				labelEpoch,
				layout.NewSpacer(),
				line2,
				layout.NewSpacer(),
			),
		),
		container.NewStack(
			container.NewHBox(
				layout.NewSpacer(),
				container.NewStack(
					rectWidth90,
					container.NewVBox(
						rectSpacer,
						wEpoch,
						container.NewHBox(
							container.NewStack(
								spacerEpoch,
								labelEpochHashes,
							),
							container.NewStack(
								spacerEpoch,
								textEpochHashes,
							),
						),
						container.NewHBox(
							container.NewStack(
								spacerEpoch,
								labelEpochBlocks,
							),
							container.NewStack(
								spacerEpoch,
								textEpochBlocks,
							),
						),
					),
				),
				layout.NewSpacer(),
			),
		),
	)

	if session.Offline {
		formEpoch.Hide()
	}

	linkBack := widget.NewHyperlinkWithStyle("Back to Dashboard", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkBack.OnTapped = func() {
		session.LastDomain = session.Window.Content()
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

	errorText := canvas.NewText(" ", colors.Green)
	errorText.TextSize = 12
	errorText.Alignment = fyne.TextAlignCenter

	optionsList := []string{"Recovery Words (Seed)", "Recovery Hex Keys", "Change Password", "Export Wallet File"}
	selectOptions := widget.NewSelect(optionsList, nil)
	selectOptions.PlaceHolder = "(Select one)"

	selectOptions.OnChanged = func(s string) {
		errorText.Text = ""
		errorText.Refresh()

		if s == "Recovery Words (Seed)" {
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
				selectOptions.ClearSelected()
			}

			btnConfirm := widget.NewButton("Submit", nil)

			entryPassword := NewReturnEntry()
			entryPassword.Password = true
			entryPassword.PlaceHolder = "Password"
			entryPassword.OnChanged = func(s string) {
				if s == "" {
					btnConfirm.Text = "Submit"
					btnConfirm.Disable()
					btnConfirm.Refresh()
				} else {
					btnConfirm.Text = "Submit"
					btnConfirm.Enable()
					btnConfirm.Refresh()
				}
			}

			btnConfirm.OnTapped = func() {
				selectOptions.ClearSelected()
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
					btnConfirm.Text = "Invalid Password..."
					btnConfirm.Disable()
					btnConfirm.Refresh()
				}
			}

			entryPassword.OnReturn = btnConfirm.OnTapped

			btnConfirm.Disable()

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
							btnConfirm,
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

			session.Window.Canvas().Focus(entryPassword)

		} else if s == "Recovery Hex Keys" {
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
				selectOptions.SetSelected("(Select One)")
			}

			btnConfirm := widget.NewButton("Submit", nil)

			entryPassword := NewReturnEntry()
			entryPassword.Password = true
			entryPassword.PlaceHolder = "Password"
			entryPassword.OnChanged = func(s string) {
				if s == "" {
					btnConfirm.Text = "Submit"
					btnConfirm.Disable()
					btnConfirm.Refresh()
				} else {
					btnConfirm.Text = "Submit"
					btnConfirm.Enable()
					btnConfirm.Refresh()
				}
			}

			btnConfirm.OnTapped = func() {
				selectOptions.ClearSelected()
				if engram.Disk.Check_Password(entryPassword.Text) {
					overlay.Add(
						container.NewStack(
							&iframe{},
							canvas.NewRectangle(colors.DarkMatter),
						),
					)

					overlay.Add(
						layoutRecoveryHex(),
					)
				} else {
					btnConfirm.Text = "Invalid Password..."
					btnConfirm.Disable()
					btnConfirm.Refresh()
				}
			}

			entryPassword.OnReturn = btnConfirm.OnTapped

			btnConfirm.Disable()

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
							btnConfirm,
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

			session.Window.Canvas().Focus(entryPassword)

		} else if s == "Change Password" {
			overlay := session.Window.Canvas().Overlays()

			header := canvas.NewText("ACCOUNT  AUTHORIZATION  REQUEST", colors.Gray)
			header.TextSize = 14
			header.Alignment = fyne.TextAlignCenter
			header.TextStyle = fyne.TextStyle{Bold: true}

			subHeader := canvas.NewText("Change Password", colors.Account)
			subHeader.TextSize = 22
			subHeader.Alignment = fyne.TextAlignCenter
			subHeader.TextStyle = fyne.TextStyle{Bold: true}

			linkClose := widget.NewHyperlinkWithStyle("Close", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
			linkClose.OnTapped = func() {
				overlay := session.Window.Canvas().Overlays()
				overlay.Top().Hide()
				overlay.Remove(overlay.Top())
				overlay.Remove(overlay.Top())
				selectOptions.ClearSelected()
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
									curPass,
								),
							),
							widget.NewLabel(""),
							widget.NewSeparator(),
							widget.NewLabel(""),
							newPass,
							rectSpacer,
							confirm,
							rectSpacer,
							rectSpacer,
							btnChange,
							widget.NewLabel(""),
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

		} else if s == "Export Wallet File" {
			verificationOverlay(
				true,
				"",
				"",
				"",
				func(b bool) {
					if b {
						go func() {
							dialogFileSave := dialog.NewFileSave(func(uri fyne.URIWriteCloser, err error) {
								if err != nil {
									logger.Errorf("[Engram] File dialog: %s\n", err)
									errorText.Text = "could not export wallet file"
									errorText.Color = colors.Red
									errorText.Refresh()
									return
								}

								if uri == nil {
									return // Canceled
								}

								data, err := os.ReadFile(session.Path)
								if err != nil {
									logger.Errorf("[Engram] Reading wallet file %s: %s\n", session.Path, err)
									errorText.Text = "error reading wallet file"
									errorText.Color = colors.Red
									errorText.Refresh()
									return
								}

								_, err = writeToURI(data, uri)
								if err != nil {
									logger.Errorf("[Engram] Exporting %s: %s\n", session.Path, err)
									errorText.Text = "error exporting wallet file"
									errorText.Color = colors.Red
									errorText.Refresh()
									return
								}

								errorText.Text = "exported wallet file successfully"
								errorText.Color = colors.Green
								errorText.Refresh()

							}, session.Window)

							if !a.Driver().Device().IsMobile() {
								// Open file browser in current directory
								uri, err := storage.ListerForURI(storage.NewFileURI(AppPath()))
								if err == nil {
									dialogFileSave.SetLocation(uri)
								} else {
									logger.Errorf("[Engram] Could not open current directory %s\n", err)
								}
							}

							dialogFileSave.SetFilter(storage.NewExtensionFileFilter([]string{".db"}))
							dialogFileSave.SetView(dialog.ListView)
							dialogFileSave.SetFileName(filepath.Base(session.Path))
							dialogFileSave.Resize(fyne.NewSize(ui.Width, ui.Height))
							dialogFileSave.Show()
						}()
					}
				},
			)
		}
	}

	var imageQR *canvas.Image

	qr, err := qrcode.New(engram.Disk.GetAddress().String(), qrcode.Highest)
	if err != nil {

	} else {
		qr.BackgroundColor = colors.DarkMatter
		qr.ForegroundColor = colors.Green
	}

	imageQR = canvas.NewImageFromImage(qr.Image(int(ui.Width * 0.65)))
	imageQR.SetMinSize(fyne.NewSize(ui.Width*0.65, ui.Width*0.65))

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
					container.NewCenter(
						imageQR,
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
				rectSpacer,
				rectSpacer,
				container.NewStack(
					container.NewHBox(
						layout.NewSpacer(),
						container.NewVBox(
							selectOptions,
							rectWidth90,
							errorText,
						),
						layout.NewSpacer(),
					),
				),
				formEpoch,
				rectSpacer,
				rectSpacer,
				container.NewStack(
					container.NewHBox(
						layout.NewSpacer(),
						line1,
						layout.NewSpacer(),
						labelDatashard,
						layout.NewSpacer(),
						line2,
						layout.NewSpacer(),
					),
				),
				rectSpacer,
				rectSpacer,
				container.NewStack(
					container.NewHBox(
						layout.NewSpacer(),
						container.NewStack(
							rectWidth90,
							container.NewVBox(
								container.NewStack(
									layout.NewSpacer(),
									headerDatashard,
									layout.NewSpacer(),
								),
								rectSpacer,
								container.NewStack(
									layout.NewSpacer(),
									textDatashard,
									layout.NewSpacer(),
								),
								rectSpacer,
								rectSpacer,
								widget.NewSeparator(),
								rectSpacer,
								rectSpacer,
								container.NewStack(
									layout.NewSpacer(),
									textDatashardDesc,
									layout.NewSpacer(),
								),
								rectSpacer,
								container.NewStack(
									layout.NewSpacer(),
									textDatashardDesc2,
									layout.NewSpacer(),
								),
							),
						),
						layout.NewSpacer(),
					),
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

	return NewVScroll(layout)
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

	return NewVScroll(layout)
}

func layoutRecoveryHex() fyne.CanvasObject {
	wSpacer := widget.NewLabel(" ")
	heading := canvas.NewText("Recovery Hex Keys", colors.Green)
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

	body := widget.NewLabel("Please save the following hex secret key in a safe place. Never share your secret key with anyone.")
	body.Wrapping = fyne.TextWrapWord
	body.Alignment = fyne.TextAlignCenter
	body.TextStyle = fyne.TextStyle{Bold: true}

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

	keys := engram.Disk.Get_Keys()
	key := fmt.Sprintf("0000000000000000000000000000000000000000000000%s", keys.Secret.Text(16))
	secret := key[len(key)-64:]
	public := keys.Public.StringHex()

	textSecret := widget.NewRichTextFromMarkdown(secret)
	textSecret.Wrapping = fyne.TextWrapWord

	linkCopySecret := widget.NewHyperlinkWithStyle("Copy Secret Key", nil, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	textPublic := widget.NewRichTextFromMarkdown(public)
	textPublic.Wrapping = fyne.TextWrapWord

	linkCopyPublic := widget.NewHyperlinkWithStyle("Copy Public Key", nil, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	labelSecret := canvas.NewText("   SECRET  KEY", colors.Gray)
	labelSecret.TextSize = 14
	labelSecret.Alignment = fyne.TextAlignLeading
	labelSecret.TextStyle = fyne.TextStyle{Bold: true}

	labelPublic := canvas.NewText("   PUBLIC  KEY", colors.Gray)
	labelPublic.TextSize = 14
	labelPublic.Alignment = fyne.TextAlignLeading
	labelPublic.TextStyle = fyne.TextStyle{Bold: true}

	labelSeparator := widget.NewRichTextFromMarkdown("")
	labelSeparator.Wrapping = fyne.TextWrapOff
	labelSeparator.ParseMarkdown("---")

	grid.Add(container.NewVBox(
		labelSecret,
		rectSpacer,
		textSecret,
		rectSpacer,
		container.NewHBox(
			linkCopySecret,
		),
		rectSpacer,
		rectSpacer,
		labelSeparator,
		rectSpacer,
		rectSpacer,
		rectSpacer,
	))

	grid.Add(container.NewVBox(
		labelPublic,
		rectSpacer,
		textPublic,
		rectSpacer,
		container.NewHBox(
			linkCopyPublic,
		),
	))

	linkCopySecret.OnTapped = func() {
		session.Window.Clipboard().SetContent(secret)
	}

	linkCopyPublic.OnTapped = func() {
		session.Window.Clipboard().SetContent(public)
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

	return NewVScroll(layout)
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
		lastOrientation := a.Driver().Device().Orientation()
		initialOrientationVertical := fyne.IsVertical(lastOrientation)

		ui.Width = ui.MaxWidth * 0.9
		ui.Height = ui.MaxHeight
		ui.Padding = ui.MaxWidth * 0.05
		if fyne.IsHorizontal(lastOrientation) {
			// Smaller if horizontal for swipe scroll
			ui.MaxWidth = ui.MaxWidth * 0.7
			ui.Width = ui.MaxWidth * 0.7
			ui.Padding = ui.MaxWidth * 0.15
		}

		resizeWindow(ui.MaxWidth, ui.MaxHeight)
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutMain())

		frameWidth := ui.MaxWidth
		frameHeight := ui.MaxHeight

		// Mobile loop checking if orientation has changed
		for a.Driver() != nil {
			currentOrientation := a.Driver().Device().Orientation()
			if lastOrientation != currentOrientation {
				if initialOrientationVertical {
					if fyne.IsVertical(lastOrientation) && !fyne.IsVertical(currentOrientation) {
						ui.MaxWidth = frameHeight
						ui.MaxHeight = frameWidth
					} else {
						ui.MaxWidth = frameWidth
						ui.MaxHeight = frameHeight
					}
				} else {
					if fyne.IsHorizontal(lastOrientation) && !fyne.IsHorizontal(currentOrientation) {
						ui.MaxWidth = frameHeight
						ui.MaxHeight = frameWidth
					} else {
						ui.MaxWidth = frameWidth
						ui.MaxHeight = frameHeight
					}
				}

				ui.Width = ui.MaxWidth * 0.9
				ui.Height = ui.MaxHeight
				ui.Padding = ui.MaxWidth * 0.05
				if fyne.IsHorizontal(currentOrientation) {
					ui.MaxWidth = ui.MaxWidth * 0.7
					ui.Width = ui.MaxWidth * 0.7
					ui.Padding = ui.MaxWidth * 0.15
				}

				lastOrientation = currentOrientation
				resizeWindow(ui.MaxWidth, ui.MaxHeight)
			}
			time.Sleep(time.Second / 2)
		}
	}()

	overlays := session.Window.Canvas().Overlays()
	overlays.Add(
		container.NewStack(
			canvas.NewRectangle(colors.DarkMatter),
		),
	)

	return container.NewVScroll(layout)
}

func layoutFileManager() fyne.CanvasObject {
	session.Domain = "app.sign"

	frame := &iframe{}

	rectBox := canvas.NewRectangle(color.Transparent)
	rectBox.SetMinSize(fyne.NewSize(ui.MaxWidth*0.9, ui.MaxHeight*0.34))
	rectWidth100 := canvas.NewRectangle(color.Transparent)
	rectWidth100.SetMinSize(fyne.NewSize(ui.Width*0.99, 10))
	rectWidth90 := canvas.NewRectangle(color.Transparent)
	rectWidth90.SetMinSize(fyne.NewSize(ui.Width*0.9, 10))
	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(6, 5))

	heading := canvas.NewText("F I L E    M A N A G E R", colors.Gray)
	heading.TextSize = 16
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	labelResults := canvas.NewText("   RESULTS", colors.Gray)
	labelResults.TextSize = 14
	labelResults.Alignment = fyne.TextAlignLeading
	labelResults.TextStyle = fyne.TextStyle{Bold: true}

	signedResults := []string{}
	signedData := binding.BindStringList(&signedResults)
	signedList := widget.NewListWithData(signedData,
		func() fyne.CanvasObject {
			return container.NewStack(
				container.NewHBox(
					container.NewStack(
						rectWidth90,
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

			split := strings.Split(str, "/")
			pos := len(split) - 1
			name := strings.Split(split[pos], ";;;")

			co.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*widget.Label).SetText(name[0])
		},
	)

	verifiedResults := []string{}
	verifiedData := binding.BindStringList(&verifiedResults)
	verifiedList := widget.NewListWithData(verifiedData,
		func() fyne.CanvasObject {
			return container.NewStack(
				container.NewHBox(
					container.NewStack(
						rectWidth90,
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

			split := strings.Split(str, "/")
			pos := len(split) - 1
			name := strings.Split(split[pos], ";;;")

			co.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*widget.Label).SetText(name[0])
		},
	)

	errorText := canvas.NewText(" ", colors.Green)
	errorText.TextSize = 12
	errorText.Alignment = fyne.TextAlignCenter

	dialogBrowse := dialog.NewFileOpen(func(uc fyne.URIReadCloser, err error) {
		if err != nil {
			logger.Errorf("[Engram] Open file dialog: %s\n", err)
			errorText.Text = "could not open file"
			errorText.Color = colors.Red
			errorText.Refresh()
			return
		}

		if uc == nil {
			return
		}

		if session.Domain == "app.sign" {
			inputFileName := uc.URI().Name()
			outputFileName := inputFileName + ".signed"

			go func() {
				dialogFileSign := dialog.NewFileSave(func(uri fyne.URIWriteCloser, err error) {
					if err != nil {
						logger.Errorf("[Engram] Save file dialog: %s\n", err)
						errorText.Text = "could not open signed file"
						errorText.Color = colors.Red
						errorText.Refresh()
						return
					}

					if uri == nil {
						return // Canceled
					}

					filedata, err := readFromURI(uc)
					if err != nil {
						logger.Errorf("[Engram] Cannot read file data for %s: %s\n", inputFileName, err)
						errorText.Text = "could not read file"
						errorText.Color = colors.Red
						errorText.Refresh()
						return
					}

					_, err = writeToURI(engram.Disk.SignData(filedata), uri)
					if err != nil {
						logger.Errorf("[Engram] Cannot sign %s: %s\n", inputFileName, err)
						errorText.Text = "could not write signed file"
						errorText.Color = colors.Red
						errorText.Refresh()
						return
					}

					outputFile := uri.URI().Name()
					if a.Driver().Device().IsMobile() {
						// Mobile uses content access name on save dialog
						outputFile = outputFileName
					}

					logger.Printf("[Engram] Successfully signed file: %s\n", outputFile)

					errorText.Text = "signed file successfully"
					errorText.Color = colors.Green
					errorText.Refresh()

					signedResults = append(signedResults, outputFile)
					signedData.Set(signedResults)
					signedList.Refresh()

					signedLen := len(signedResults)
					labelResults.Text = fmt.Sprintf("   RESULTS  (%d / %d)", signedLen, signedLen)
					labelResults.Refresh()

				}, session.Window)

				if !a.Driver().Device().IsMobile() {
					// Open file browser in current directory
					uri, err := storage.ListerForURI(storage.NewFileURI(AppPath()))
					if err == nil {
						dialogFileSign.SetLocation(uri)
					} else {
						logger.Errorf("[Engram] Could not open current directory %s\n", err)
					}
				}

				dialogFileSign.SetFilter(storage.NewExtensionFileFilter([]string{".signed"}))
				dialogFileSign.SetView(dialog.ListView)
				dialogFileSign.SetFileName(outputFileName)
				dialogFileSign.Resize(fyne.NewSize(ui.Width, ui.Height))
				dialogFileSign.SetConfirmText("Save Sign")
				dialogFileSign.Show()
			}()
		} else {
			fileName := uc.URI().Name()
			if !strings.HasSuffix(fileName, ".signed") {
				errorText.Text = "verifying requires a .signed file"
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}

			filedata, err := readFromURI(uc)
			if err != nil {
				logger.Errorf("[Engram] Cannot read file data for %s: %s\n", fileName, err)
				errorText.Text = "could not read file"
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}

			// Trim off .signed from file because engram.Disk.CheckFileSignature() adds it back on anyways - https://github.com/deroproject/derohe/blob/main/walletapi/wallet.go#L709
			fileName = strings.TrimSuffix(fileName, ".signed")
			signer, message, err := engram.Disk.CheckSignature(filedata)
			if err != nil {
				logger.Errorf("[Engram] Signature verification failed for %s: %s\n", fileName, err)
				errorText.Text = "signature verification failed"
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}

			logger.Printf("[Engram] %s signed by: %s\n", fileName, signer.String())
			if isASCII(string(message)) {
				fmt.Println(string(message))
			}

			errorText.Text = "verified file successfully"
			errorText.Color = colors.Green
			errorText.Refresh()

			verifiedResults = append(verifiedResults, fileName+";;;"+signer.String())
			verifiedData.Set(verifiedResults)
			verifiedList.Refresh()

			verifiedLen := len(verifiedResults)
			labelResults.Text = fmt.Sprintf("   RESULTS  (%d / %d)", verifiedLen, verifiedLen)
			labelResults.Refresh()
		}
	}, session.Window)

	if !a.Driver().Device().IsMobile() {
		// Open file browser in current directory
		uri, err := storage.ListerForURI(storage.NewFileURI(AppPath()))
		if err == nil {
			dialogBrowse.SetLocation(uri)
		} else {
			logger.Errorf("[Engram] Could not open current directory %s\n", err)
		}
	}

	dialogBrowse.Resize(fyne.NewSize(ui.Width, ui.Height))
	dialogBrowse.SetView(dialog.ListView)

	signedList.OnSelected = func(id widget.ListItemID) {
		errorText.Text = ""
		errorText.Refresh()
	}

	verifiedList.OnSelected = func(id widget.ListItemID) {
		errorText.Text = ""
		errorText.Refresh()

		if session.Domain == "app.verify" {
			split := strings.Split(verifiedResults[id], ";;;")
			filepath := strings.Split(split[0], "/")
			filename := filepath[len(filepath)-1]
			filename = strings.Replace(filename, ".signed", "", -1)

			rectSpan := canvas.NewRectangle(color.Transparent)
			rectSpan.SetMinSize(fyne.NewSize(ui.Width*0.99, 10))

			header := canvas.NewText("S I G N A T U R E    D E T A I L", colors.Gray)
			header.TextSize = 16
			header.Alignment = fyne.TextAlignCenter
			header.TextStyle = fyne.TextStyle{Bold: true}

			labelStatus := canvas.NewText("   VERIFICATION   STATUS", colors.Gray)
			labelStatus.TextSize = 12
			labelStatus.TextStyle = fyne.TextStyle{Bold: true}
			labelStatus.Alignment = fyne.TextAlignCenter

			valueStatus := canvas.NewText("   Verified", colors.Green)
			valueStatus.TextSize = 22
			valueStatus.TextStyle = fyne.TextStyle{Bold: true}
			valueStatus.Alignment = fyne.TextAlignCenter

			labelFilename := canvas.NewText("   FILENAME", colors.Gray)
			labelFilename.TextSize = 14
			labelFilename.TextStyle = fyne.TextStyle{Bold: true}

			valueFilename := widget.NewRichTextFromMarkdown(filename)
			valueFilename.Wrapping = fyne.TextWrapBreak

			labelSigner := canvas.NewText("   SIGNER   ADDRESS", colors.Gray)
			labelSigner.TextSize = 14
			labelSigner.TextStyle = fyne.TextStyle{Bold: true}

			valueSigner := widget.NewRichTextFromMarkdown(split[1])
			valueSigner.Wrapping = fyne.TextWrapBreak

			labelSeparator := widget.NewRichTextFromMarkdown("")
			labelSeparator.Wrapping = fyne.TextWrapOff
			labelSeparator.ParseMarkdown("---")

			labelSeparator2 := widget.NewRichTextFromMarkdown("")
			labelSeparator2.Wrapping = fyne.TextWrapOff
			labelSeparator2.ParseMarkdown("---")

			labelSeparator3 := widget.NewRichTextFromMarkdown("")
			labelSeparator3.Wrapping = fyne.TextWrapOff
			labelSeparator3.ParseMarkdown("---")

			linkBack := widget.NewHyperlinkWithStyle("Hide Details", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
			linkBack.OnTapped = func() {
				removeOverlays()
			}

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
					container.NewHBox(
						layout.NewSpacer(),
						container.NewVBox(
							rectSpan,
							rectSpacer,
							header,
							rectSpacer,
							rectSpacer,
							container.NewHBox(
								layout.NewSpacer(),
								container.NewVBox(
									valueStatus,
									rectSpacer,
									labelStatus,
								),
								layout.NewSpacer(),
							),
							rectSpacer,
							rectSpacer,
							labelSeparator,
							rectSpacer,
							rectSpacer,
							labelFilename,
							rectSpacer,
							valueFilename,
							rectSpacer,
							rectSpacer,
							labelSeparator2,
							rectSpacer,
							rectSpacer,
							labelSigner,
							rectSpacer,
							valueSigner,
							rectSpacer,
							rectSpacer,
							labelSeparator3,
							rectSpacer,
							rectSpacer,
							container.NewHBox(
								layout.NewSpacer(),
								linkBack,
								layout.NewSpacer(),
							),
						),
						layout.NewSpacer(),
					),
				),
			)
			overlay.Top().Show()

			verifiedList.UnselectAll()
		}
	}

	btnBrowse := widget.NewButton("Browse Files", nil)
	btnBrowse.OnTapped = func() {
		errorText.Text = ""
		errorText.Refresh()
		if session.Domain == "app.sign" {
			dialogBrowse.SetFilter(nil)
			dialogBrowse.SetConfirmText("Open")
		} else {
			dialogBrowse.SetFilter(storage.NewExtensionFileFilter([]string{".signed"}))
			dialogBrowse.SetConfirmText("Verify")
		}

		dialogBrowse.Show()
	}

	labelAction := canvas.NewText("( DRAG-AND-DROP ENABLED )", colors.Gray)
	labelAction.TextSize = 12
	labelAction.Alignment = fyne.TextAlignLeading
	labelAction.TextStyle = fyne.TextStyle{Bold: true}

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

	labelSeparator := widget.NewRichTextFromMarkdown("")
	labelSeparator.Wrapping = fyne.TextWrapOff
	labelSeparator.ParseMarkdown("---")

	labelSeparator2 := widget.NewRichTextFromMarkdown("")
	labelSeparator2.Wrapping = fyne.TextWrapOff
	labelSeparator2.ParseMarkdown("---")

	labelSeparator3 := widget.NewRichTextFromMarkdown("")
	labelSeparator3.Wrapping = fyne.TextWrapOff
	labelSeparator3.ParseMarkdown("---")

	labelSeparator4 := widget.NewRichTextFromMarkdown("")
	labelSeparator4.Wrapping = fyne.TextWrapOff
	labelSeparator4.ParseMarkdown("---")

	linkBack := widget.NewHyperlinkWithStyle("Back to Dashboard", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkBack.OnTapped = func() {
		removeOverlays()
		capture := session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutDashboard())
		session.Domain = "app.wallet"
		session.LastDomain = capture
	}

	selectType := widget.NewSelect([]string{"Sign Files", "Verify Signed Files"}, nil)
	selectType.SetSelected("Sign Files")

	// Handle drag & drop files for file signing/verifying
	session.Window.SetOnDropped(func(p fyne.Position, files []fyne.URI) {
		errorText.Text = ""
		errorText.Refresh()

		if session.Domain == "app.sign" {
			if a.Driver().Device().IsMobile() {
				if len(files) > 1 {
					errorText.Text = "single file only"
					errorText.Color = colors.Red
					errorText.Refresh()
					return
				}

				inputFileName := files[0].Name()

				dialogFileSign := dialog.NewFileSave(func(uri fyne.URIWriteCloser, err error) {
					if err != nil {
						logger.Errorf("[Engram] File dialog: %s\n", err)
						errorText.Text = "could not open signed file"
						errorText.Color = colors.Red
						errorText.Refresh()
						return
					}

					if uri == nil {
						return // Canceled
					}

					uc, err := storage.Reader(files[0])
					if err != nil {
						logger.Errorf("[Engram] Cannot create reader for %s: %s\n", inputFileName, err)
						errorText.Text = "could not access file"
						errorText.Color = colors.Red
						errorText.Refresh()
						return
					}

					filedata, err := readFromURI(uc)
					if err != nil {
						logger.Errorf("[Engram] Cannot read file data for %s: %s\n", inputFileName, err)
						errorText.Text = "could not read file"
						errorText.Color = colors.Red
						errorText.Refresh()
						return
					}

					_, err = writeToURI(engram.Disk.SignData(filedata), uri)
					if err != nil {
						logger.Errorf("[Engram] Cannot sign %s: %s\n", inputFileName, err)
						errorText.Text = "could not write signed file"
						errorText.Color = colors.Red
						errorText.Refresh()
						return
					}

					// Mobile uses content access name on save dialog
					outputFile := inputFileName + ".signed"

					logger.Printf("[Engram] Successfully signed file: %s\n", outputFile)

					errorText.Text = "signed file successfully"
					errorText.Color = colors.Green
					errorText.Refresh()

					signedResults = append(signedResults, outputFile)
					signedData.Set(signedResults)
					signedList.Refresh()

					signedLen := len(signedResults)
					labelResults.Text = fmt.Sprintf("   RESULTS  (%d / %d)", signedLen, signedLen)
					labelResults.Refresh()

				}, session.Window)

				dialogFileSign.SetFilter(storage.NewExtensionFileFilter([]string{".signed"}))
				dialogFileSign.SetView(dialog.ListView)
				dialogFileSign.SetFileName(inputFileName)
				dialogFileSign.Resize(fyne.NewSize(ui.Width, ui.Height))
				dialogFileSign.SetConfirmText("Save Sign")
				dialogFileSign.Show()
			} else {
				singedLen := len(signedResults)
				count := 1 + singedLen

				for i, f := range files {
					inputFileName := f.Name()

					uc, err := storage.Reader(f)
					if err != nil {
						logger.Errorf("[Engram] Cannot create reader for %s: %s\n", inputFileName, err)
						errorText.Text = fmt.Sprintf("could not access file %d", i)
						errorText.Color = colors.Red
						errorText.Refresh()
						continue
					}

					filedata, err := readFromURI(uc)
					if err != nil {
						logger.Errorf("[Engram] Cannot read file data for %s: %s\n", inputFileName, err)
						errorText.Text = fmt.Sprintf("could not read file %d", i)
						errorText.Color = colors.Red
						errorText.Refresh()
						continue
					}

					outputfile := inputFileName + ".signed"

					if err := os.WriteFile(outputfile, engram.Disk.SignData(filedata), 0600); err != nil {
						logger.Errorf("[Engram] Cannot sign %s: %s\n", inputFileName, err)
						errorText.Text = fmt.Sprintf("cannot sign file %d", i)
						errorText.Color = colors.Red
						errorText.Refresh()
					} else {
						logger.Printf("[Engram] Successfully signed file: %s\n", outputfile)
						labelResults.Text = fmt.Sprintf("   RESULTS  (%d / %d)", count, len(files)+singedLen)
						labelResults.Refresh()
						signedResults = append(signedResults, outputfile)
						count += 1
					}
				}

				signedData.Set(signedResults)
				signedList.Refresh()
			}
		} else if session.Domain == "app.verify" {
			if a.Driver().Device().IsMobile() {
				dialogVerify := dialog.NewFileOpen(func(uc fyne.URIReadCloser, err error) {
					errorText.Text = ""
					if uc != nil {
						fileName := uc.URI().Name()
						if filepath.Ext(fileName) != ".signed" {
							errorText.Text = "requires a .signed file"
							errorText.Color = colors.Red
							errorText.Refresh()
							return
						}

						filedata, err := readFromURI(uc)
						if err != nil {
							logger.Errorf("[Engram] Cannot read URI file data for %s: %s\n", fileName, err)
							errorText.Text = "cannot read file data"
							errorText.Color = colors.Red
							errorText.Refresh()
							return
						}

						signer, message, err := engram.Disk.CheckSignature(filedata)
						if err != nil {
							logger.Errorf("[Engram] Signature verification failed for %s: %s\n", fileName, err)
							errorText.Text = "signature verification failed"
							errorText.Color = colors.Red
							errorText.Refresh()
							return
						}

						logger.Printf("[Engram] %s signed by: %s\n", fileName, signer.String())
						if isASCII(string(message)) {
							fmt.Println(string(message))
						}

						errorText.Text = "verified file successfully"
						errorText.Color = colors.Green
						errorText.Refresh()

						verifiedResults = append(verifiedResults, fileName+";;;"+signer.String())
						verifiedData.Set(verifiedResults)
						verifiedList.Refresh()

						verifiedLen := len(verifiedResults)
						labelResults.Text = fmt.Sprintf("   RESULTS  (%d / %d)", verifiedLen, verifiedLen)
						labelResults.Refresh()
					}
				}, session.Window)

				dialogVerify.Resize(fyne.NewSize(ui.Width, ui.Height))
				dialogVerify.SetView(dialog.ListView)
				dialogVerify.Show()
			} else {
				verifiedLen := len(verifiedResults)
				count := 1 + verifiedLen

				for i, f := range files {
					inputFileName := f.Name()

					uc, err := storage.Reader(f)
					if err != nil {
						logger.Errorf("[Engram] Cannot create reader for %s: %s\n", inputFileName, err)
						errorText.Text = fmt.Sprintf("could not access file %d", i)
						errorText.Color = colors.Red
						errorText.Refresh()
						continue
					}

					filedata, err := readFromURI(uc)
					if err != nil {
						logger.Errorf("[Engram] Cannot read file data for %s: %s\n", inputFileName, err)
						errorText.Text = fmt.Sprintf("could not read file %d", i)
						errorText.Color = colors.Red
						errorText.Refresh()
						continue
					}

					outputfile := strings.TrimSuffix(inputFileName, ".signed")

					if signer, message, err := engram.Disk.CheckSignature(filedata); err != nil {
						logger.Errorf("[Engram] Signature verification failed for %s: %s\n", inputFileName, err)
						errorText.Text = fmt.Sprintf("signature verification %d failed", i)
						errorText.Color = colors.Red
						errorText.Refresh()
					} else {
						logger.Printf("[Engram] Signed by: %s\n", signer.String())

						if isASCII(string(message)) {
							logger.Printf("[Engram] Message for %s: %s\n", inputFileName, signer.String())
						}

						if err := os.WriteFile(outputfile, message, 0600); err != nil {
							logger.Errorf("[Engram] Cannot write output file for %s: %s\n", outputfile, err)
							continue
						}

						logger.Printf("[Engram] Successfully wrote message to file: %s\n", outputfile)

						labelResults.Text = fmt.Sprintf("   RESULTS  (%d / %d)", count, len(files)+verifiedLen)
						labelResults.Refresh()
						verifiedResults = append(verifiedResults, inputFileName+";;;"+signer.String())
						count += 1
					}
				}

				verifiedData.Set(verifiedResults)
				verifiedList.Refresh()
			}
		}
	})

	top := container.NewVBox(
		rectSpacer,
		rectSpacer,
		heading,
	)

	center := container.NewStack(
		rectWidth100,
		container.NewHBox(
			layout.NewSpacer(),
			container.NewStack(
				rectWidth90,
				container.NewVBox(
					rectSpacer,
					rectSpacer,
					selectType,
					rectSpacer,
					rectSpacer,
					btnBrowse,
					rectSpacer,
					rectSpacer,
					container.NewHBox(
						layout.NewSpacer(),
						labelAction,
						layout.NewSpacer(),
					),
					rectSpacer,
					errorText,
					rectSpacer,
					labelSeparator,
					rectSpacer,
					rectSpacer,
					labelResults,
					rectSpacer,
					rectSpacer,
					container.NewStack(
						rectBox,
						signedList,
					),
					rectSpacer,
				),
			),
			layout.NewSpacer(),
		),
	)

	selectType.OnChanged = func(s string) {
		if s == "Sign Files" {
			session.Domain = "app.sign"
			signedList.UnselectAll()
			center.Objects[1].(*fyne.Container).Objects[1].(*fyne.Container).Objects[1].(*fyne.Container).Objects[18].(*fyne.Container).Objects[1] = signedList
			signedData.Set(signedResults)
			signedList.Refresh()
			signedLen := len(signedResults)
			labelResults.Text = fmt.Sprintf("   RESULTS  (%d / %d)", signedLen, signedLen)
			labelResults.Refresh()
		} else {
			session.Domain = "app.verify"
			center.Objects[1].(*fyne.Container).Objects[1].(*fyne.Container).Objects[1].(*fyne.Container).Objects[18].(*fyne.Container).Objects[1] = verifiedList
			verifiedData.Set(verifiedResults)
			verifiedList.Refresh()
			verifiedLen := len(verifiedResults)
			labelResults.Text = fmt.Sprintf("   RESULTS  (%d / %d)", verifiedLen, verifiedLen)
			labelResults.Refresh()
		}

		errorText.Text = ""
		errorText.Refresh()
	}

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

	body := container.NewVBox(
		top,
		center,
	)

	layout := container.NewStack(
		frame,
		container.NewBorder(
			body,
			bottom,
			nil,
			nil,
		),
	)

	return NewVScroll(layout)
}

func layoutContractBuilder(promptText string) fyne.CanvasObject {
	session.Domain = "app.sc.builder"

	frame := &iframe{}

	rectBox := canvas.NewRectangle(color.Transparent)
	rectBox.SetMinSize(fyne.NewSize(ui.MaxWidth*0.9, ui.MaxHeight*0.35))

	rectWidth100 := canvas.NewRectangle(color.Transparent)
	rectWidth100.SetMinSize(fyne.NewSize(ui.Width*0.99, 10))

	rectWidth90 := canvas.NewRectangle(color.Transparent)
	rectWidth90.SetMinSize(fyne.NewSize(ui.Width*0.9, 10))

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(6, 5))

	heading := canvas.NewText("C O N T R A C T    B U I L D E R", colors.Gray)
	heading.TextSize = 16
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	errorText := canvas.NewText(promptText, colors.Red)
	errorText.TextSize = 12
	errorText.Alignment = fyne.TextAlignCenter

	// Open .bas SC from file browser
	dialogBrowse := dialog.NewFileOpen(func(uc fyne.URIReadCloser, err error) {
		errorText.Text = ""
		if uc != nil {
			filename := uc.URI().Name()
			if uc.URI().MimeType() != "text/plain" {
				logger.Errorf("[Engram] Cannot open file %s in contract builder\n", filename)
				errorText.Text = "cannot open file"
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}

			if filepath.Ext(filename) != ".bas" {
				errorText.Text = "requires a .bas file"
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}

			filedata, err := readFromURI(uc)
			if err != nil {
				logger.Errorf("[Engram] Cannot read URI file data for %s: %s\n", filename, err)
				errorText.Text = "cannot read file data"
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}

			if !isASCII(string(filedata)) {
				errorText.Text = "invalid file data"
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}

			removeOverlays()
			capture := session.Window.Content()
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutContractEditor(strings.TrimSuffix(filename, ".bas"), string(filedata)))
			session.LastDomain = capture
		}
	}, session.Window)

	if !a.Driver().Device().IsMobile() {
		// Open file browser in current directory
		uri, err := storage.ListerForURI(storage.NewFileURI(AppPath()))
		if err == nil {
			dialogBrowse.SetLocation(uri)
		} else {
			logger.Errorf("[Engram] Could not open current directory %s\n", err)
		}
	}

	// Resize browser to app size and add SC file filter
	dialogBrowse.Resize(fyne.NewSize(ui.Width, ui.Height))
	dialogBrowse.SetFilter(storage.NewExtensionFileFilter([]string{".bas"}))
	dialogBrowse.SetView(dialog.ListView)

	btnBrowse := widget.NewButton("Browse Files", nil)
	btnBrowse.OnTapped = func() {
		dialogBrowse.Show()
	}

	btnEditor := widget.NewButton("Open Editor", nil)
	btnEditor.OnTapped = func() {
		removeOverlays()
		capture := session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutContractEditor("", ""))
		session.LastDomain = capture
	}

	labelAction := canvas.NewText("( DRAG-AND-DROP ENABLED )", colors.Gray)
	labelAction.TextSize = 12
	labelAction.Alignment = fyne.TextAlignLeading
	labelAction.TextStyle = fyne.TextStyle{Bold: true}

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

	labelSeparator := widget.NewRichTextFromMarkdown("")
	labelSeparator.Wrapping = fyne.TextWrapOff
	labelSeparator.ParseMarkdown("---")

	linkBack := widget.NewHyperlinkWithStyle("Back to Dashboard", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkBack.OnTapped = func() {
		removeOverlays()
		capture := session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutDashboard())
		session.LastDomain = capture
	}

	// Handle drag & drop files for smart contracts
	session.Window.SetOnDropped(func(p fyne.Position, files []fyne.URI) {
		if session.Domain == "app.sc.builder" {
			errorText.Text = ""
			errorText.Refresh()

			if len(files) > 1 {
				errorText.Text = "single .bas file only"
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			} else {
				uri, err := storage.Reader(files[0])
				if err != nil {
					errorText.Text = "could not read dropped file"
					errorText.Color = colors.Red
					errorText.Refresh()
					return
				}

				filename := files[0].Name()
				if filepath.Ext(filename) != ".bas" {
					errorText.Text = "requires a .bas file"
					errorText.Color = colors.Red
					errorText.Refresh()
					return
				}

				filedata, err := readFromURI(uri)
				if err != nil {
					logger.Errorf("[Engram] Cannot read file data for %s: %s\n", filename, err)
					errorText.Text = "cannot read file data"
					errorText.Color = colors.Red
					errorText.Refresh()
					return
				}

				go func() {
					removeOverlays()
					capture := session.Window.Content()
					session.Window.SetContent(layoutTransition())
					session.Window.SetContent(layoutContractEditor(strings.TrimSuffix(filepath.Base(filename), ".bas"), string(filedata)))
					session.LastDomain = capture
				}()
			}
		}
	})

	entryClone := widget.NewEntry()
	entryClone.SetPlaceHolder("Clone SCID")
	if session.Offline {
		entryClone.Disable()
		entryClone.SetText("Cloning disabled in offline mode")
	}

	entryClone.OnChanged = func(s string) {
		if len(s) == 64 {
			removeOverlays()
			capture := session.Window.Content()
			session.Window.SetContent(layoutTransition())

			code, err := getContractCode(s)
			if err != nil {
				logger.Errorf("[Engram] Clone SC: %s\n", err)
				errorText.Text = "cannot get contract for clone"
				errorText.Color = colors.Red
				errorText.Refresh()
				session.Window.SetContent(layoutContractBuilder(errorText.Text))
				return
			}

			if code == "" {
				errorText.Text = "contract does not exists"
				errorText.Color = colors.Red
				errorText.Refresh()
				session.Window.SetContent(layoutContractBuilder(errorText.Text))
				return
			}

			session.Window.SetContent(layoutContractEditor("", code))
			session.LastDomain = capture
		} else {
			if s == "" {
				errorText.Text = ""
				errorText.Refresh()
			} else {
				errorText.Text = "not a valid scid"
				errorText.Color = colors.Red
				errorText.Refresh()
			}
		}
	}

	top := container.NewVBox(
		rectSpacer,
		rectSpacer,
		heading,
	)

	center := container.NewStack(
		rectWidth100,
		container.NewHBox(
			layout.NewSpacer(),
			container.NewStack(
				rectWidth90,
				container.NewVBox(
					rectSpacer,
					rectSpacer,
					entryClone,
					errorText,
					rectSpacer,
					btnBrowse,
					rectSpacer,
					rectSpacer,
					container.NewHBox(
						layout.NewSpacer(),
						labelAction,
						layout.NewSpacer(),
					),
					rectSpacer,
					rectSpacer,
					btnEditor,
					rectSpacer,
					labelSeparator,
					rectSpacer,
					rectBox,
				),
			),
			layout.NewSpacer(),
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

	body := container.NewVBox(
		top,
		center,
	)

	layout := container.NewStack(
		frame,
		container.NewBorder(
			body,
			bottom,
			nil,
			nil,
		),
	)

	return NewVScroll(layout)
}

func layoutContractEditor(filename, filedata string) fyne.CanvasObject {
	session.Domain = "app.sc.editor"

	frame := &iframe{}

	rectBox := canvas.NewRectangle(color.Transparent)
	rectBox.SetMinSize(fyne.NewSize(ui.MaxWidth*0.9, ui.MaxHeight*0.35))

	rectWidth100 := canvas.NewRectangle(color.Transparent)
	rectWidth100.SetMinSize(fyne.NewSize(ui.Width*0.99, 10))

	rectWidth90 := canvas.NewRectangle(color.Transparent)
	rectWidth90.SetMinSize(fyne.NewSize(ui.Width*0.9, 10))

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(6, 5))

	rectCode := canvas.NewRectangle(color.Transparent)
	rectCode.SetMinSize(fyne.NewSize(ui.MaxWidth*0.9, ui.MaxHeight*0.35))

	heading := canvas.NewText("C O N T R A C T    E D I T O R", colors.Green)
	heading.TextSize = 16
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	labelHeaders := canvas.NewText("   HEADERS", colors.Gray)
	labelHeaders.TextSize = 14
	labelHeaders.Alignment = fyne.TextAlignLeading
	labelHeaders.TextStyle = fyne.TextStyle{Bold: true}

	labelCode := canvas.NewText("   CODE (DVM-BASIC)", colors.Gray)
	labelCode.TextSize = 14
	labelCode.Alignment = fyne.TextAlignLeading
	labelCode.TextStyle = fyne.TextStyle{Bold: true}

	labelCodeSize := canvas.NewText("(0.0KB) ", colors.Green)
	labelCodeSize.TextSize = 12
	labelCodeSize.Alignment = fyne.TextAlignTrailing

	errorText := canvas.NewText(" ", colors.Green)
	errorText.TextSize = 12
	errorText.Alignment = fyne.TextAlignCenter

	var nameHdr, iconURLHdr, descrHdr string
	nameHdr = filename

	// Get headers from contract code initialize func
	if filedata != "" {
		contract, _, err := dvm.ParseSmartContract(filedata)
		if err == nil {
			for n, f := range contract.Functions {
				if n == "InitializePrivate" || n == "Initialize" {
					for _, line := range f.Lines {
						lineLen := len(line) - 1
						if lineLen < 5 {
							// Line is to short to be a STORE
							continue
						}

						for i, parts := range line {
							if parts == "STORE" {
								// Find if code is storing headers
								if line[i+2] == `"nameHdr"` {
									nameHdr = strings.Trim(line[i+4], `"`)
								} else if line[i+2] == `"iconURLHdr"` {
									iconURLHdr = strings.Trim(line[i+4], `"`)
								} else if line[i+2] == `"descrHdr"` {
									descrHdr = strings.Trim(line[i+4], `"`)
								}
							}
						}
					}
				}
			}
		}
	}

	entryName := widget.NewEntry()
	entryName.SetText(nameHdr)
	entryName.SetPlaceHolder("Name")
	entryName.Validator = func(s string) (err error) {
		if s == "" {
			err = fmt.Errorf("enter a name")
			errorText.Text = err.Error()
			errorText.Color = colors.Red
			errorText.Refresh()
			return
		}

		errorText.Text = ""
		errorText.Refresh()

		return nil
	}

	entryIcon := widget.NewEntry()
	entryIcon.SetPlaceHolder("Icon")
	entryIcon.SetText(iconURLHdr)
	entryIcon.Validator = func(s string) (err error) {
		if s == "" {
			err = fmt.Errorf("enter icon URL")
			errorText.Text = err.Error()
			errorText.Color = colors.Red
			errorText.Refresh()
			return
		}

		errorText.Text = ""
		errorText.Refresh()

		return nil
	}

	var entryUpdated bool
	entryDescription := widget.NewEntry()
	entryDescription.SetPlaceHolder("Description")
	entryDescription.SetText(descrHdr)
	entryDescription.Validator = func(s string) (err error) {
		if s == "" && entryUpdated {
			err = fmt.Errorf("enter description")
			errorText.Text = err.Error()
			errorText.Color = colors.Red
			errorText.Refresh()
			return
		}

		entryUpdated = true

		errorText.Text = ""
		errorText.Refresh()

		return nil
	}

	var unsavedChanges bool
	entryCode := widget.NewMultiLineEntry()
	entryCode.SetPlaceHolder("Code")
	entryCode.Wrapping = fyne.TextWrapWord
	entryCode.OnChanged = func(s string) {
		errorText.Text = ""
		errorText.Refresh()

		size := tela.GetCodeSizeInKB(s)

		labelCodeSize.Text = fmt.Sprintf("(%.2fKB) ", size)
		if size > 20 {
			labelCodeSize.Color = colors.Red
			errorText.Text = "contract size is to large"
			errorText.Color = colors.Red
			errorText.Refresh()
		} else if size > 18.5 {
			labelCodeSize.Color = colors.Yellow
		} else {
			labelCodeSize.Color = colors.Green
		}
		labelCodeSize.Refresh()

		if s != filedata {
			unsavedChanges = true
		} else {
			unsavedChanges = false
		}
	}

	entryCode.SetText(filedata)

	options := []string{"Initialize", "Set Headers", "New Function", "Parse", "Format", "Clear", "Export"}
	if !session.Offline {
		splice := append([]string{"Import Function"}, options[3:]...)
		options = append(options[:3], splice...)
		options = append(options, "Install")
	}

	selectEditor := widget.NewSelect(options, nil)

	entryForm := container.NewVBox(
		rectSpacer,
		selectEditor,
		rectSpacer,
		container.NewBorder(
			nil,
			nil,
			labelCode,
			labelCodeSize,
			nil,
		),
		container.NewStack(
			rectCode,
			entryCode,
		),
		errorText,
		rectSpacer,
		labelHeaders,
		rectSpacer,
		entryName,
		entryIcon,
		entryDescription,
	)

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

	labelSeparator := widget.NewRichTextFromMarkdown("")
	labelSeparator.Wrapping = fyne.TextWrapOff
	labelSeparator.ParseMarkdown("---")

	linkBack := widget.NewHyperlinkWithStyle("Back to Contract Builder", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkBack.OnTapped = func() {
		if unsavedChanges {
			verificationOverlay(
				false,
				"CONTRACT  EDITOR",
				"Leave with unsaved changes",
				"Confirm",
				func(b bool) {
					if b {
						capture := session.Window.Content()
						session.Window.SetContent(layoutTransition())
						session.Window.SetContent(layoutContractBuilder(""))
						session.LastDomain = capture
					}
				},
			)
		} else {
			removeOverlays()
			capture := session.Window.Content()
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutContractBuilder(""))
			session.LastDomain = capture
		}
	}

	selectEditor.OnChanged = func(s string) {
		errorText.Text = ""
		errorText.Refresh()

		switch s {
		case "Initialize": // Set entry text with new starter initialize func
			if entryCode.Text == "" {
				entryCode.SetText(dvmInitFuncExample())
				errorText.Text = "new initialize function created"
				errorText.Color = colors.Green
				errorText.Refresh()
				return
			}

			verificationOverlay(
				false,
				"CONTRACT  EDITOR",
				"Reset to default initialize function",
				"Confirm",
				func(b bool) {
					if b {
						entryCode.SetText(dvmInitFuncExample())
						errorText.Text = "new initialize function created"
						errorText.Color = colors.Green
						errorText.Refresh()
					}
				},
			)
		case "New Function": // Add a new starter initialize func to code entry
			increment := 1
			var hasInitFunc bool
			fn := tela.GetSmartContractFuncNames(entryCode.Text)
			for _, n := range fn {
				// Increment function number if new() already esists
				if strings.TrimRight(n, "0123456789") == "new" {
					increment++
				}

				if n == "InitializePrivate" || n == "Initialize" {
					hasInitFunc = true
				}
			}

			if !hasInitFunc {
				errorText.Text = "no initialize function"
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}

			if strings.HasSuffix(entryCode.Text, "\n") {
				entryCode.SetText(entryCode.Text + "\n" + dvmFuncExample(increment))
			} else {
				entryCode.SetText(entryCode.Text + "\n\n" + dvmFuncExample(increment))
			}

			errorText.Text = "new function added"
			errorText.Color = colors.Green
			errorText.Refresh()
		case "Import Function": // Import a function from an on-chain scid
			var hasInitFunc bool
			fn := tela.GetSmartContractFuncNames(entryCode.Text)
			for _, n := range fn {
				if n == "InitializePrivate" || n == "Initialize" {
					hasInitFunc = true
					break
				}
			}

			entryEntrypoint := widget.NewEntry()
			entryEntrypoint.SetPlaceHolder("Function name")
			entryEntrypoint.Validator = func(s string) (err error) {
				if s == "" || (len(s) > 0 && !unicode.IsLetter(rune(s[0]))) {
					return fmt.Errorf("invalid function name")
				}

				return nil
			}

			entrySCID := widget.NewEntry()
			entrySCID.SetPlaceHolder("SCID")
			entrySCID.Validator = func(s string) (err error) {
				if len(s) != 64 {
					return fmt.Errorf("not a valid scid")
				}

				return nil
			}

			overlay := session.Window.Canvas().Overlays()

			header := canvas.NewText("CONTRACT  EDITOR", colors.Gray)
			header.TextSize = 14
			header.Alignment = fyne.TextAlignCenter
			header.TextStyle = fyne.TextStyle{Bold: true}

			subHeader := canvas.NewText("Import an existing function", colors.Account)
			subHeader.TextSize = 22
			subHeader.Alignment = fyne.TextAlignCenter
			subHeader.TextStyle = fyne.TextStyle{Bold: true}

			linkCancel := widget.NewHyperlinkWithStyle("Cancel", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
			linkCancel.OnTapped = func() {
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

			paramsContainer := container.NewVBox(entrySCID, entryEntrypoint)

			btnImport := widget.NewButton("Import", nil)
			btnImport.OnTapped = func() {
				if entrySCID.Validate() != nil {
					entrySCID.FocusGained()
					entrySCID.FocusLost()
					return
				}

				if entryEntrypoint.Validate() != nil {
					entryEntrypoint.FocusGained()
					entryEntrypoint.FocusLost()
					return
				}

				defer removeOverlays()

				if !hasInitFunc {
					if entryEntrypoint.Text != "InitializePrivate" && entryEntrypoint.Text != "Initialize" {
						errorText.Text = "need initializing function first"
						errorText.Color = colors.Red
						errorText.Refresh()
						return
					}
				}

				code, err := getContractCode(entrySCID.Text)
				if err != nil {
					logger.Errorf("[Engram] Editor import function error: %s\n", err)
					errorText.Text = "cannot get contract for function import"
					errorText.Color = colors.Red
					errorText.Refresh()
					return
				}

				if code == "" {
					errorText.Text = "contract does not exists"
					errorText.Color = colors.Red
					errorText.Refresh()
					return
				}

				entrypoint := entryEntrypoint.Text
				contract, pos, err := dvm.ParseSmartContract(code)
				if err != nil {
					logger.Errorf("[Engram] Editor import parsing error: %s %s\n", err, pos)
					errorText.Text = fmt.Sprintf("error parsing contract %s", pos)
					errorText.Color = colors.Red
					errorText.Refresh()
					return
				}

				var tempSC dvm.SmartContract
				tempSC.Functions = make(map[string]dvm.Function)

				for name, f := range contract.Functions {
					if name == entrypoint {
						tempSC.Functions[name] = f
						break
					}
				}

				if tempSC.Functions[entrypoint].LineNumbers == nil {
					errorText.Text = "function not found on scid"
					errorText.Color = colors.Red
					errorText.Refresh()
					return
				}

				formatted, err := tela.FormatSmartContract(tempSC, fmt.Sprintf("Function %s", entrypoint))
				if err != nil {
					logger.Errorf("[Engram] Editor import formatting error: %s\n", err)
					errorText.Text = "could not parse dvm to string"
					errorText.Color = colors.Red
					errorText.Refresh()
					return
				}

				if entryCode.Text == "" {
					entryCode.SetText(formatted)
				} else if strings.HasSuffix(entryCode.Text, "\n") {
					entryCode.SetText(entryCode.Text + "\n" + formatted)
				} else {
					entryCode.SetText(entryCode.Text + "\n\n" + formatted)
				}

				errorText.Text = "imported function successfully"
				errorText.Color = colors.Green
				errorText.Refresh()
			}

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
								subHeader,
							),
							widget.NewLabel(""),
							rectSpacer,
							rectSpacer,
							paramsContainer,
							rectSpacer,
							rectSpacer,
							btnImport,
							rectSpacer,
							rectSpacer,
							container.NewHBox(
								layout.NewSpacer(),
								linkCancel,
								layout.NewSpacer(),
							),
							rectSpacer,
							rectSpacer,
						),
					),
				),
			)
		case "Clear": // Clears SC code entry
			verificationOverlay(
				false,
				"CONTRACT  EDITOR",
				"Clear code entry",
				"Confirm",
				func(b bool) {
					if b {
						entryCode.SetText("")
						errorText.Text = "contract code cleared"
						errorText.Color = colors.Green
						errorText.Refresh()
					}
				},
			)
		case "Parse": // Parse SC for errors
			if entryCode.Text == "" {
				errorText.Text = "contract code is empty"
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}

			_, pos, err := dvm.ParseSmartContract(entryCode.Text)
			if err != nil {
				errorText.Text = fmt.Sprintf("error parsing contract %s", pos)
				errorText.Color = colors.Red
				errorText.Refresh()
				logger.Errorf("[Engram] Parse SC: %s %s\n", err, pos)
				return
			}

			errorText.Text = "contract parsed successfully"
			errorText.Color = colors.Green
			errorText.Refresh()
		case "Set Headers": // Set Artificer standard headers into initialize func
			if entryCode.Text == "" {
				errorText.Text = "contract code is empty"
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}

			contract, pos, err := dvm.ParseSmartContract(entryCode.Text)
			if err != nil {
				errorText.Text = fmt.Sprintf("error parsing contract %s", pos)
				errorText.Color = colors.Red
				errorText.Refresh()
				logger.Errorf("[Engram] Set SC Headers: %s %s\n", err, pos)
				return
			}

			if entryName.Validate() == nil && entryIcon.Validate() == nil && entryDescription.Validate() == nil {
				// Create add header func to use later in confirmations
				addFunction := func() {
					var haveHeader [uint64(3)]bool
					for name, function := range contract.Functions {
						// Find initialize func
						if name == "Initialize" || name == "InitializePrivate" {
							for _, line := range function.Lines {
								lineLen := len(line) - 1
								if lineLen < 5 {
									// Line is to short to be a STORE
									continue
								}

								for i, parts := range line {
									if parts == "STORE" {
										// Find if code is storing headers and update vars with header entry value
										if line[i+2] == `"nameHdr"` {
											haveHeader[0] = true
											line[i+4] = fmt.Sprintf(`"%s"`, entryName.Text)
										} else if line[i+2] == `"iconURLHdr"` {
											haveHeader[1] = true
											line[i+4] = fmt.Sprintf(`"%s"`, entryIcon.Text)
										} else if line[i+2] == `"descrHdr"` {
											haveHeader[2] = true
											line[i+4] = fmt.Sprintf(`"%s"`, entryDescription.Text)
										}
									}
								}
							}
						}
					}

					// Check if any headers are missing
					var needToAdd, hasInitFunc bool
					for _, hh := range haveHeader {
						if !hh {
							needToAdd = true
							break
						}
					}

					// SC has all headers already, update the code entry
					if !needToAdd {
						code, err := tela.FormatSmartContract(contract, entryCode.Text)
						if err != nil {
							logger.Errorf("[Engram] Format code error: %s\n", err)
							err = errors.New("could not parse dvm to string")
							errorText.Text = err.Error()
							errorText.Color = colors.Red
							errorText.Refresh()
							return
						}

						entryCode.SetText(code)

						errorText.Text = "headers updated"
						errorText.Color = colors.Green
						errorText.Refresh()
						return
					}

					// SC is missing one or more headers so they will be added into initialize func
					for name, function := range contract.Functions {
						if name == "Initialize" || name == "InitializePrivate" {
							hasInitFunc = true

							lineLen := len(function.LineNumbers)
							indexEnd := lineLen - 1

							// Starting from the last line number loop upwards
							for i := 0; i < lineLen; i++ {
								index := indexEnd - i
								if index < 0 {
									break
								}

								line := function.Lines[function.LineNumbers[index]]
								if len(line) < 1 {
									continue
								}

								// If line is RETURN 0 will inject headers here and push RETURN 0 line down if there is room
								if line[0] == "RETURN" && line[1] == "0" {
									if index-1 < 0 {
										err = errors.New("no room for header lines")
										errorText.Text = err.Error()
										errorText.Color = colors.Red
										errorText.Refresh()
										return
									} else if i > 0 && function.LineNumbers[index+1] < function.LineNumbers[index]+4 {
										err = fmt.Errorf("no room for header lines below %d", function.LineNumbers[index])
										errorText.Text = err.Error()
										errorText.Color = colors.Red
										errorText.Refresh()
										return
									} else {
										var addedLines, skipedLines uint64
										for u := uint64(1); u < 5; u++ {
											addLineNum := function.LineNumbers[index] + (u - 1) - skipedLines
											switch u {
											case 1: // nameHdr
												if !haveHeader[0] {
													function.Lines[addLineNum] = []string{"STORE", "(", `"nameHdr"`, ",", fmt.Sprintf(`"%s"`, entryName.Text), ")"}
													if u != 1 {
														function.LineNumbers = append(function.LineNumbers, addLineNum)
													}
													addedLines++
												} else {
													// Count skip if we have already to subtract to line number
													skipedLines++
													continue
												}
											case 2: // iconURLHdr
												if !haveHeader[1] {
													function.Lines[addLineNum] = []string{"STORE", "(", `"iconURLHdr"`, ",", fmt.Sprintf(`"%s"`, entryIcon.Text), ")"}
													if u != 1 && skipedLines != 1 {
														function.LineNumbers = append(function.LineNumbers, addLineNum)
													}
													addedLines++
												} else {
													skipedLines++
													continue
												}
											case 3: // descrHdr
												if !haveHeader[2] {
													function.Lines[addLineNum] = []string{"STORE", "(", `"descrHdr"`, ",", fmt.Sprintf(`"%s"`, entryDescription.Text), ")"}
													if u != 1 && skipedLines != 2 {
														function.LineNumbers = append(function.LineNumbers, addLineNum)
													}
													addedLines++
												}
											case 4:
												function.Lines[addLineNum] = []string{"RETURN", "0"}
												function.LineNumbers = append(function.LineNumbers, addLineNum)
											}
										}

										// If changes were made sort line numbers and add them to index
										if addedLines > 0 {
											sort.Slice(function.LineNumbers, func(i, j int) bool {
												return function.LineNumbers[i] < function.LineNumbers[j]
											})

											for u, ln := range function.LineNumbers {
												function.LinesNumberIndex[ln] = uint64(u)
											}

											contract.Functions[name] = function
										}

										// fmt.Println("Lines", contract.Functions[name].Lines)
										// fmt.Println("LineNumbers", contract.Functions[name].LineNumbers)
										// fmt.Println("LineNumberIndex", contract.Functions[name].LinesNumberIndex)

										break
									}
								}
							}
						}
					}

					if !hasInitFunc {
						err = errors.New("no initialize function")
						errorText.Text = err.Error()
						errorText.Color = colors.Red
						errorText.Refresh()
						return
					}

					code, err := tela.FormatSmartContract(contract, entryCode.Text)
					if err != nil {
						logger.Errorf("[Engram] Format code error: %s\n", err)
						err = errors.New("could not parse dvm to string")
						errorText.Text = err.Error()
						errorText.Color = colors.Red
						errorText.Refresh()
						return
					}

					if code == entryCode.Text {
						errorText.Text = "did not change headers"
						errorText.Color = colors.Red
						errorText.Refresh()
						return
					}

					entryCode.SetText(code)

					errorText.Text = "contract headers added successfully"
					errorText.Color = colors.Green
					errorText.Refresh()
				}

				codeCheck, err := tela.FormatSmartContract(contract, entryCode.Text)
				if err != nil {
					logger.Errorf("[Engram] Format code error: %s\n", err)
					err = errors.New("could not parse dvm to string")
					errorText.Text = err.Error()
					errorText.Color = colors.Red
					errorText.Refresh()
					return
				}

				// Warn user that code will be formatted if headers are added
				if codeCheck != entryCode.Text {
					verificationOverlay(
						false,
						"CONTRACT  EDITOR",
						"Setting headers formats your code",
						"Confirm",
						func(b bool) {
							if b {
								addFunction()
							}
						},
					)
				} else {
					addFunction()
				}
			}
		case "Format": // Format SC code
			if entryCode.Text == "" {
				errorText.Text = "contract code is empty"
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}

			contract, pos, err := dvm.ParseSmartContract(entryCode.Text)
			if err != nil {
				errorText.Text = fmt.Sprintf("error parsing contract %s", pos)
				errorText.Color = colors.Red
				errorText.Refresh()
				logger.Errorf("[Engram] Format: %s %s\n", err, pos)
				return
			}

			code, err := tela.FormatSmartContract(contract, entryCode.Text)
			if err != nil {
				logger.Errorf("[Engram] Format code error: %s\n", err)
				err = errors.New("could not parse dvm to string")
				errorText.Text = err.Error()
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}

			if code == entryCode.Text {
				errorText.Text = "contract code is formatted"
				errorText.Color = colors.Green
				errorText.Refresh()
				return
			}

			verificationOverlay(
				false,
				"CONTRACT  EDITOR",
				"Remove whitespace and comments",
				"Confirm",
				func(b bool) {
					if b {
						entryCode.SetText(code)

						errorText.Text = "contract code formatted successfully"
						errorText.Color = colors.Green
						errorText.Refresh()
					}
				},
			)
		case "Export": // Export SC to file
			if entryCode.Text == "" {
				errorText.Text = "contract code is empty"
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}

			exportFileName := fmt.Sprintf("%s.bas", entryName.Text)

			data := []byte(entryCode.Text)
			dialogFileSave := dialog.NewFileSave(func(uri fyne.URIWriteCloser, err error) {
				if err != nil {
					logger.Errorf("[Engram] File dialog: %s\n", err)
					errorText.Text = "could not export contract file"
					errorText.Color = colors.Red
					errorText.Refresh()
					return
				}

				if uri == nil {
					return // Canceled
				}

				_, err = writeToURI(data, uri)
				if err != nil {
					logger.Errorf("[Engram] Exporting %s: %s\n", exportFileName, err)
					errorText.Text = "error exporting contract file"
					errorText.Color = colors.Red
					errorText.Refresh()
					return
				}

				unsavedChanges = false
				filedata = entryCode.Text
				errorText.Text = "exported contract file successfully"
				errorText.Color = colors.Green
				errorText.Refresh()

			}, session.Window)

			if !a.Driver().Device().IsMobile() {
				// Open file browser in current directory
				uri, err := storage.ListerForURI(storage.NewFileURI(AppPath()))
				if err == nil {
					dialogFileSave.SetLocation(uri)
				} else {
					logger.Errorf("[Engram] Could not open current directory %s\n", err)
				}
			}

			dialogFileSave.SetFilter(storage.NewExtensionFileFilter([]string{".bas"}))
			dialogFileSave.SetView(dialog.ListView)
			dialogFileSave.SetFileName(exportFileName)
			dialogFileSave.Resize(fyne.NewSize(ui.Width, ui.Height))
			dialogFileSave.Show()
		case "Install": // Install SC
			code := entryCode.Text
			if code == "" {
				errorText.Text = "contract code is empty"
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}

			contract, pos, err := dvm.ParseSmartContract(code)
			if err != nil {
				logger.Errorf("[Engram] Install SC: %s %s\n", err, pos)
				errorText.Text = fmt.Sprintf("error parsing contract %s", pos)
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}

			var entrypoint string
			var args []rpc.Argument
			for name, function := range contract.Functions {
				if name == "InitializePrivate" || name == "Initialize" {
					entrypoint = name
					for _, v := range function.Params {
						switch v.Type {
						case 0x4:
							args = append(args, rpc.Argument{Name: v.Name, DataType: rpc.DataUint64, Value: v.ValueUint64})
						case 0x5:
							args = append(args, rpc.Argument{Name: v.Name, DataType: rpc.DataString, Value: v.ValueString})
						}
					}
				}
			}

			if entrypoint == "" {
				errorText.Text = "missing initializing entrypoint"
				errorText.Color = colors.Red
				errorText.Refresh()
				return
			}

			function := contract.Functions[entrypoint]

			var paramList []fyne.Widget
			if len(function.Params) > 0 {
				params := function.Params
				for i := range params {
					p := i
					entry := widget.NewEntry()
					entry.PlaceHolder = params[p].Name
					if params[p].Type == 0x4 {
						entry.PlaceHolder = params[p].Name + " (Numbers Only)"
					}

					entry.Validator = func(s string) error {
						switch params[p].Type {
						case 0x5:
							return nil
						case 0x4:
							if params[p].Name+" (Numbers Only)" == entry.PlaceHolder {
								amount, err := globals.ParseAmount(s)
								if err != nil {
									logger.Debugf("[%s] Param error: %s\n", params[p].Name, err)
									return err
								} else {
									logger.Debugf("[%s] Amount: %d\n", params[p].Name, amount)
								}
							}
						}

						return nil
					}

					paramList = append(paramList, entry)
				}

				overlay := session.Window.Canvas().Overlays()

				header := canvas.NewText("INSTALL  SMART  CONTRACT", colors.Gray)
				header.TextSize = 14
				header.Alignment = fyne.TextAlignCenter
				header.TextStyle = fyne.TextStyle{Bold: true}

				subHeader := canvas.NewText(fmt.Sprintf("%s params", entrypoint), colors.Account)
				subHeader.TextSize = 22
				subHeader.Alignment = fyne.TextAlignCenter
				subHeader.TextStyle = fyne.TextStyle{Bold: true}

				linkClose := widget.NewHyperlinkWithStyle("Close", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
				linkClose.OnTapped = func() {
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

				paramsContainer := container.NewVBox()

				btnInstall := widget.NewButton("Install", nil)

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
									subHeader,
								),
								widget.NewLabel(""),
								//selectRingMembers,
								rectSpacer,
								rectSpacer,
								paramsContainer,
								rectSpacer,
								rectSpacer,
								btnInstall,
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

				for _, w := range paramList {
					c := container.NewStack(
						span,
						w,
					)

					paramsContainer.Add(c)
					paramsContainer.Refresh()
				}

				btnInstall.OnTapped = func() {
					validated := true
					for _, w := range paramList {
						entry, ok := w.(*widget.Entry)
						if !ok {
							continue
						}

						if entry.Validate() != nil {
							entry.FocusGained()
							entry.FocusLost()
							validated = false
							break
						}
					}

					if !validated {
						return
					}

					btnInstall.Text = "Installing..."
					btnInstall.Disable()
					btnInstall.Refresh()

					verificationOverlay(
						true,
						"CONTRACT  EDITOR",
						"",
						"",
						func(b bool) {
							if b {
								_, err := installSC(code, args)
								if err != nil {
									errorText.Text = err.Error()
									errorText.Color = colors.Red
									errorText.Refresh()
									return
								}

								unsavedChanges = false
								errorText.Text = "contract installed successfully"
								errorText.Color = colors.Green
								errorText.Refresh()
							}

							overlay.Top().Hide()
							overlay.Remove(overlay.Top())
							overlay.Remove(overlay.Top())
						},
					)
				}

				paramsContainer.Refresh()
				overlay.Top().Show()
			} else {
				verificationOverlay(
					true,
					"CONTRACT  EDITOR",
					"",
					"",
					func(b bool) {
						if b {
							_, err := installSC(code, args)
							if err != nil {
								errorText.Text = err.Error()
								errorText.Color = colors.Red
								errorText.Refresh()
								return
							}

							unsavedChanges = false
							errorText.Text = "contract installed successfully"
							errorText.Color = colors.Green
							errorText.Refresh()
						}
					},
				)
			}
		}
	}

	top := container.NewVBox(
		rectSpacer,
		rectSpacer,
		heading,
	)

	center := container.NewStack(
		rectWidth100,
		container.NewHBox(
			layout.NewSpacer(),
			container.NewStack(
				rectWidth90,
				container.NewVBox(
					rectSpacer,
					container.NewStack(
						rectBox,
						entryForm,
					),
					rectSpacer,
				),
			),
			layout.NewSpacer(),
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

	body := container.NewVBox(
		top,
		center,
	)

	layout := container.NewStack(
		frame,
		container.NewBorder(
			body,
			bottom,
			nil,
			nil,
		),
	)

	return NewVScroll(layout)
}

func layoutTELA() fyne.CanvasObject {
	session.Domain = "app.tela"

	var history []string
	var historyData binding.StringList
	var historyList *widget.List

	var searching []string
	var searchData binding.StringList
	var searchList *widget.List

	var serving []string
	var servingData binding.StringList
	var servingList *widget.List

	frame := &iframe{}
	rectLeft := canvas.NewRectangle(color.Transparent)
	rectLeft.SetMinSize(fyne.NewSize(ui.Width*0.40, 35))

	rectRight := canvas.NewRectangle(color.Transparent)
	rectRight.SetMinSize(fyne.NewSize(ui.Width*0.58, 35))

	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(fyne.NewSize(ui.Width, ui.Height*0.36))

	rectWidth := canvas.NewRectangle(color.Transparent)
	rectWidth.SetMinSize(fyne.NewSize(ui.Width, 10))

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(6, 5))

	heading := canvas.NewText("T E L A    B R O W S E R", colors.Gray)
	heading.TextSize = 16
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	results := canvas.NewText("", colors.Green)
	results.TextSize = 13

	labelLastScan := canvas.NewText("", colors.Green)
	labelLastScan.TextSize = 13

	errorText := canvas.NewText(" ", colors.Green)
	errorText.TextSize = 12
	errorText.Alignment = fyne.TextAlignCenter

	historyData = binding.BindStringList(&history)
	historyList = widget.NewListWithData(historyData,
		func() fyne.CanvasObject {
			return container.NewStack(
				container.NewVBox(
					container.NewStack(
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

			co.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*widget.Label).SetText(split[0])
		},
	)

	searchData = binding.BindStringList(&searching)
	searchList = widget.NewListWithData(searchData,
		func() fyne.CanvasObject {
			return container.NewStack(
				container.NewVBox(
					container.NewStack(
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

			co.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*widget.Label).SetText(split[0])
		},
	)

	servingData = binding.BindStringList(&serving)
	servingList = widget.NewListWithData(servingData,
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

			co.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*widget.Label).SetText(split[1])
			co.(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*fyne.Container).Objects[1].(*widget.Label).SetText(split[0])
		},
	)

	menuLabel := canvas.NewText("  M O R E   O P T I O N S  ", colors.Gray)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	entryHistory := widget.NewEntry()
	entryHistory.PlaceHolder = "Search History"
	entryHistory.Disable()

	entryServeSCID := widget.NewEntry()
	entryServeSCID.PlaceHolder = "Serve by SCID"

	entryAddSCID := widget.NewEntry()
	entryAddSCID.PlaceHolder = "Add SCID"
	entryAddSCID.OnChanged = func(s string) {
		errorText.Text = ""
		errorText.Refresh()
		if len(s) == 64 {
			if gnomon.Index != nil {
				if gnomon.GetAllSCIDVariableDetails(s) != nil {
					errorText.Text = "scid already exists"
					errorText.Color = colors.Yellow
					errorText.Refresh()
					return
				}

				code, err := getContractCode(s)
				if err != nil || code == "" {
					errorText.Text = "could not get scid"
					errorText.Color = colors.Red
					errorText.Refresh()
					return
				}

				err = gnomon.AddSCIDToIndex(s)
				if err != nil {
					errorText.Text = "error adding scid"
					errorText.Color = colors.Red
					errorText.Refresh()
					return
				}

				entryAddSCID.SetText("")
				errorText.Text = "scid added"
				errorText.Color = colors.Green
				errorText.Refresh()
			}
		}
	}

	entrySearchCompletions := []string{"author:", "durl:", "name:", "my:"}
	entrySearch := x.NewCompletionEntry(entrySearchCompletions)
	entrySearch.PlaceHolder = "Search TELA"
	entrySearch.Disable()

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

	var isSearching bool

	linkBack := widget.NewHyperlinkWithStyle("Back to Dashboard", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkBack.OnTapped = func() {
		session.Domain = "app.wallet" // break any loops now
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutDashboard())
		removeOverlays()
	}

	linkClearHistory := widget.NewHyperlinkWithStyle("Clear All", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: false})
	linkClearHistory.OnTapped = func() {
		verificationOverlay(
			false,
			"TELA BROWSER",
			"Clear history?",
			"Confirm",
			func(b bool) {
				if b {
					if gnomon.Index == nil || session.Offline {
						return
					}

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

					tree, err := ss.GetTree("TELA History")
					if err != nil {
						return
					}

					c := tree.Cursor()

					for k, _, err := c.First(); err == nil; k, _, err = c.Next() {
						DeleteKey(tree.GetName(), k)
					}

					session.Window.SetContent(layoutTransition())
					session.Window.SetContent(layoutTELA())
				}
			},
		)
	}

	wSelect := widget.NewSelect([]string{"History", "Active", "Search", "Settings"}, nil)
	wSelect.SetSelectedIndex(0)

	btnShutdown := widget.NewButton("Shutdown TELA", nil)

	var rescanRecheck bool
	var lastScan, searchExclusions string
	var minLikes float64
	var telaSCIDs []string
	var telaSearch []tela.INDEX
	var sAll = map[string]bool{}

	getSearchResults := func() {
		entrySearch.Disable()
		if isSearching {
			return
		}

		isSearching = true

		// Already scanned
		if len(telaSearch) > 0 {
			searching = telaSearchDisplayAll(telaSearch)
			searchData.Set(searching)

			results.Text = fmt.Sprintf("  TELA SCIDs:  %d", len(telaSearch))
			results.Color = colors.Green
			results.Refresh()
			entrySearch.Enable()

			labelLastScan.Text = fmt.Sprintf("  %s", lastScan)
			labelLastScan.Color = colors.Green
			labelLastScan.Refresh()
			isSearching = false

			return
		}

		telaSearch = []tela.INDEX{}
		searchData.Set(nil)
		btnShutdown.Disable()
		labelLastScan.Text = ""
		labelLastScan.Refresh()
		defer func() {
			isSearching = false
			if !session.Offline && gnomon.Index != nil {
				if btnShutdown.Disabled() {
					btnShutdown.Enable()
				}
			}
		}()

		if gnomon.Index == nil {
			return
		}

		for gnomon.Index.LastIndexedHeight < int64(engram.Disk.Get_Daemon_Height()) {
			if !strings.Contains(session.Domain, ".tela") {
				return
			}

			entrySearch.Disable()
			results.Text = "  Gnomon is syncing..."
			results.Color = colors.Yellow
			results.Refresh()
			time.Sleep(time.Second)
		}

		storedAllSCIDs, err := GetEncryptedValue("TELA Search", []byte("Searched SCIDs"))
		if err != nil {
			// Nothing stored, scan for SCIDs
			sAll = map[string]bool{}
			logger.Debugf("[Engram] Could not get stored TELA Searched SCIDs: %s\n", err)
		} else {
			json.Unmarshal(storedAllSCIDs, &sAll)
		}

		storedSCIDs, err := GetEncryptedValue("TELA Search", []byte("SCIDs"))
		if err != nil {
			// Nothing stored, scan for SCIDs
			telaSCIDs = []string{}
			logger.Debugf("[Engram] Could not get stored TELA SCIDs: %s\n", err)
		} else {
			// Have stored SCIDs
			json.Unmarshal(storedSCIDs, &telaSCIDs)

			for i, sc := range telaSCIDs {
				if index, err := tela.GetINDEXInfo(sc, session.Daemon); err == nil {
					if gnomon.GetAllSCIDVariableDetails(sc) == nil {
						results.Text = fmt.Sprintf("  Adding... (%d / %d)", i, len(telaSCIDs))
						results.Color = colors.Yellow
						results.Refresh()

						gnomon.AddSCIDToIndex(sc)
					}

					_, err := getLikesRatio(sc, index.DURL, searchExclusions, minLikes)
					if err != nil {
						continue
					}

					telaSearch = append(telaSearch, index)
				}
			}

			// If recheck is false, run a rescan that pulls in any new contracts when first OnChanged to Search
			if rescanRecheck && (len(telaSearch) > 0 || len(telaSCIDs) > 0) {
				searching = telaSearchDisplayAll(telaSearch)
				searchData.Set(searching)

				results.Text = fmt.Sprintf("  TELA SCIDs:  %d", len(telaSearch))
				results.Color = colors.Green
				results.Refresh()
				entrySearch.Enable()

				if last, err := GetEncryptedValue("TELA Search", []byte("Last Scan")); err == nil {
					lastScan = string(last)
					labelLastScan.Text = fmt.Sprintf("  %s", lastScan)
					labelLastScan.Color = colors.Green
					labelLastScan.Refresh()
				}

				return
			}
		}

		var wg sync.WaitGroup

		all := gnomon.GetAllOwnersAndSCIDs()
		allLen := len(all)
		scanned := 0
		workers := make(chan struct{}, runtime.NumCPU())

		for sc := range all {
			workers <- struct{}{}
			if gnomon.Index == nil || !strings.Contains(session.Domain, ".tela") {
				break
			}

			scanned++
			results.Text = fmt.Sprintf("  Scanning... (%d / %d)", scanned, allLen)
			results.Color = colors.Yellow
			results.Refresh()

			wg.Add(1)
			go func(scid string) {
				defer func() {
					<-workers
					wg.Done()
				}()

				if !rescanRecheck && (sAll[scid] || scidExist(telaSCIDs, scid)) {
					return
				}

				vs, _, err := gnomon.Index.GetSCIDValuesByKey([]*structures.SCIDVariable{}, scid, "telaVersion", gnomon.Index.ChainHeight)
				if err != nil {
					return
				}

				if vs != nil {
					if index, err := tela.GetINDEXInfo(scid, session.Daemon); err == nil {
						if len(index.DOCs) > 0 {
							if strings.HasSuffix(index.DURL, tela.TAG_LIBRARY) || strings.Contains(index.DURL, tela.TAG_DOC_SHARDS) {
								return
							}

							if gnomon.GetAllSCIDVariableDetails(scid) == nil {
								gnomon.AddSCIDToIndex(scid)
							}

							telaSCIDs = append(telaSCIDs, scid)

							_, err := getLikesRatio(scid, index.DURL, searchExclusions, minLikes)
							if err != nil {
								return
							}

							telaSearch = append(telaSearch, index)
						}
					}
				}
			}(sc)
		}

		if !strings.Contains(session.Domain, ".tela") {
			return
		}

		wg.Wait()

		searching = telaSearchDisplayAll(telaSearch)
		searchData.Set(searching)

		results.Text = fmt.Sprintf("  TELA SCIDs:  %d", len(telaSearch))
		results.Color = colors.Green
		results.Refresh()

		timeNow := time.Now().Format(time.RFC822)
		StoreEncryptedValue("TELA Search", []byte("Last Scan"), []byte(timeNow))
		if storeSCIDs, err := json.Marshal(telaSCIDs); err == nil {
			StoreEncryptedValue("TELA Search", []byte("SCIDs"), storeSCIDs)
		}

		if !rescanRecheck {
			for sc := range all {
				sAll[sc] = true
			}

			if sAllSCIDs, err := json.Marshal(sAll); err == nil {
				StoreEncryptedValue("TELA Search", []byte("Searched SCIDs"), sAllSCIDs)
			}
		}

		lastScan = timeNow
		labelLastScan.Text = fmt.Sprintf("  %s", lastScan)
		labelLastScan.Color = colors.Green
		labelLastScan.Refresh()
		entrySearch.Enable()
	}

	entrySearch.OnChanged = func(s string) {
		errorText.Text = ""
		errorText.Refresh()

		if s == "" {
			go getSearchResults()
			if !a.Driver().Device().IsMobile() {
				entrySearch.HideCompletion()
			}

			return
		}

		if !a.Driver().Device().IsMobile() {
			if len(s) < 3 {
				entrySearch.SetOptions(append([]string{s}, entrySearchCompletions...))
				entrySearch.ShowCompletion()
			} else {
				entrySearch.HideCompletion()
			}
		}

		var queryResult []string
		query := strings.Split(s, ":")
		if len(query) < 2 {
			if len(s) == 64 {
				// Search scid
				for _, ind := range telaSearch {
					_, err := getLikesRatio(ind.SCID, ind.DURL, searchExclusions, minLikes)
					if err != nil {
						continue
					}

					if ind.SCID == s {
						queryResult = append(queryResult, telaSearchDisplay(ind))
						break
					}
				}
			} else {
				// Search all
				for _, ind := range telaSearch {
					_, err := getLikesRatio(ind.SCID, ind.DURL, searchExclusions, minLikes)
					if err != nil {
						continue
					}

					data := []string{
						ind.NameHdr,
						ind.DescrHdr,
						ind.DURL,
						ind.SCID,
					}

					for _, split := range data {
						if strings.Contains(split, s) {
							queryResult = append(queryResult, telaSearchDisplay(ind))
							break
						}
					}
				}
			}

			sort.Strings(queryResult)
			searching = queryResult
			searchData.Set(searching)
			searchList.Refresh()

			results.Text = fmt.Sprintf("  TELA SCIDs:  %d", len(queryResult))
			results.Color = colors.Green
			results.Refresh()
			entrySearch.Enable()

			return
		}

		switch query[0] {
		case "name":
			for _, ind := range telaSearch {
				_, err := getLikesRatio(ind.SCID, ind.DURL, searchExclusions, minLikes)
				if err != nil {
					continue
				}

				if strings.Contains(ind.NameHdr, query[1]) {
					queryResult = append(queryResult, telaSearchDisplay(ind))
				}
			}
		case "durl":
			for _, ind := range telaSearch {
				_, err := getLikesRatio(ind.SCID, ind.DURL, searchExclusions, minLikes)
				if err != nil {
					continue
				}

				if strings.Contains(ind.DURL, query[1]) {
					queryResult = append(queryResult, telaSearchDisplay(ind))
				}
			}
		case "my":
			for _, ind := range telaSearch {
				if ind.Author == engram.Disk.GetAddress().String() {
					queryResult = append(queryResult, telaSearchDisplay(ind))
				}
			}
		case "author":
			if len(query[1]) != 66 {
				return
			}

			_, err := globals.ParseValidateAddress(query[1])
			if err != nil {
				return
			}

			for _, ind := range telaSearch {
				_, err := getLikesRatio(ind.SCID, ind.DURL, searchExclusions, minLikes)
				if err != nil {
					continue
				}

				if ind.Author == query[1] {
					queryResult = append(queryResult, telaSearchDisplay(ind))
				}
			}
		default:
			errorText.Text = "unknown search prefix"
			errorText.Color = colors.Red
			errorText.Refresh()

			return
		}

		sort.Strings(queryResult)
		searching = queryResult
		searchData.Set(searching)
		searchList.Refresh()

		results.Text = fmt.Sprintf("  TELA SCIDs:  %d", len(queryResult))
		results.Color = colors.Green
		results.Refresh()
		entrySearch.Enable()
	}

	// Refresh the active server list
	refreshServerList := func() {
		time.Sleep(time.Second * 2)
		var serversRunning []string
		for _, serv := range tela.GetServerInfo() {
			serversRunning = append(serversRunning, serv.Name+";;;"+serv.Address+";;;;;;"+serv.SCID)
		}

		sort.Strings(serversRunning)
		servingData.Set(serversRunning)
		servingList.Refresh()
		if !isSearching && wSelect.Selected == "Active" {
			results.Text = fmt.Sprintf("  Active Servers:  %d", len(serversRunning))
			results.Color = colors.Green
			results.Refresh()
		}
	}

	btnShutdown.OnTapped = func() {
		switch btnShutdown.Text {
		case "Rescan Blockchain":
			verificationOverlay(
				false,
				"TELA BROWSER",
				"Rescan blockchain?",
				"Confirm",
				func(b bool) {
					if b {
						if isSearching {
							return
						}

						telaSearch = []tela.INDEX{}
						telaSCIDs = []string{}
						if rescanRecheck {
							DeleteKey("TELA Search", []byte("SCIDs"))
							DeleteKey("TELA Search", []byte("Searched SCIDs"))
						}
						errorText.Text = ""
						errorText.Refresh()
						go getSearchResults()
					}
				},
			)
		default:
			verificationOverlay(
				false,
				"TELA BROWSER",
				"Shutdown all active TELA servers?",
				"Confirm",
				func(b bool) {
					if b {
						tela.ShutdownTELA()
						servingData.Set(nil)
						errorText.Text = ""
						errorText.Refresh()
					}
				},
			)
		}

		go refreshServerList()
	}

	entrySpacer := canvas.NewRectangle(color.Transparent)
	entrySpacer.SetMinSize(fyne.NewSize(140, 0))

	entryPort := widget.NewEntry()
	entryPort.SetText(strconv.Itoa(tela.PortStart()))
	entryPort.Validator = func(s string) (err error) {
		i, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("invalid port")
		}

		return tela.SetPortStart(i)
	}

	entryMinLikes := widget.NewEntry()
	entryMinLikes.SetPlaceHolder("Likes %")
	if storedMinLikes, err := GetEncryptedValue("TELA Settings", []byte("Min Likes")); err == nil {
		if f, err := strconv.ParseFloat(string(storedMinLikes), 64); err == nil {
			minLikes = f
			entryMinLikes.SetText(string(storedMinLikes))
		}
	} else {
		minLikes = 30
		entryMinLikes.SetText("30")
	}

	entryMinLikes.Validator = func(s string) (err error) {
		i, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("invalid percent")
		}

		if i < 0 || i > 100 {
			err = fmt.Errorf("must be 0 to 100")
			return
		}

		// Clear search results but keep scids
		telaSearch = []tela.INDEX{}

		minLikes = float64(i)
		StoreEncryptedValue("TELA Settings", []byte("Min Likes"), []byte(s))

		return
	}

	entryExclusions := widget.NewEntry()
	entryExclusions.SetPlaceHolder("dURL Exclusions (exclude1,exclude2)")
	if storedExclusions, err := GetEncryptedValue("TELA Settings", []byte("Exclusions")); err == nil {
		searchExclusions = string(storedExclusions)
		entryExclusions.SetText(searchExclusions)
	}

	entryExclusions.OnChanged = func(s string) {
		if s != "" {
			StoreEncryptedValue("TELA Settings", []byte("Exclusions"), []byte(s))
		} else {
			DeleteKey("TELA Settings", []byte("Exclusions"))
		}

		// Clear search results but keep scids
		telaSearch = []tela.INDEX{}

		searchExclusions = s
	}

	wUpdates := widget.NewSelect([]string{xswd.Deny.String(), xswd.Allow.String()}, nil)
	if tela.UpdatesAllowed() {
		wUpdates.SetSelectedIndex(1)
	} else {
		wUpdates.SetSelectedIndex(0)
	}

	wUpdates.OnChanged = func(s string) {
		if s == xswd.Allow.String() {
			tela.AllowUpdates(true)
		} else {
			tela.AllowUpdates(false)
		}
	}

	if storedRescanRecheck, err := GetEncryptedValue("TELA Settings", []byte("Rescan Recheck")); err == nil {
		if string(storedRescanRecheck) == "Yes" {
			rescanRecheck = true
		} else {
			rescanRecheck = false
		}
	}

	wRescanRecheck := widget.NewSelect([]string{"No", "Yes"}, nil)
	if rescanRecheck {
		wRescanRecheck.SetSelectedIndex(1)
	} else {
		wRescanRecheck.SetSelectedIndex(0)
	}

	wRescanRecheck.OnChanged = func(s string) {
		if s == "Yes" {
			rescanRecheck = true
		} else {
			rescanRecheck = false
		}

		StoreEncryptedValue("TELA Settings", []byte("Rescan Recheck"), []byte(s))
	}

	historyBox := container.NewStack(
		rectList,
		historyList,
	)

	searchBox := container.NewStack(
		rectList,
		searchList,
	)

	servingBox := container.NewStack(
		rectList,
		servingList,
	)

	linkSpacer := canvas.NewRectangle(color.Transparent)
	linkSpacer.SetMinSize(fyne.NewSize(0, 40))

	linkResetDefaults := widget.NewHyperlinkWithStyle("Reset Default Settings", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkResetDefaults.OnTapped = func() {
		verificationOverlay(
			false,
			"TELA BROWSER",
			"Reset to default settings?",
			"Confirm",
			func(b bool) {
				if b {
					wUpdates.SetSelectedIndex(0)
					wRescanRecheck.SetSelectedIndex(0)
					entryPort.SetText(strconv.Itoa(tela.DEFAULT_PORT_START))
					entryMinLikes.SetText("30")
					entryExclusions.SetText("")
				}
			},
		)
	}

	linkSearchClear := widget.NewHyperlinkWithStyle("Delete Search Data", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkSearchClear.OnTapped = func() {
		verificationOverlay(
			false,
			"TELA BROWSER",
			"Delete stored search data?",
			"Confirm",
			func(b bool) {
				if b {
					telaSearch = []tela.INDEX{}
					telaSCIDs = []string{}
					DeleteKey("TELA Search", []byte("SCIDs"))
					DeleteKey("TELA Search", []byte("Searched SCIDs"))
					DeleteKey("TELA Search", []byte("Last Scan"))
					linkSearchClear.Hide()
				}
			},
		)
	}

	settingsBox := container.NewVScroll(
		container.NewStack(
			rectList,
			container.NewBorder(
				nil,
				nil,
				nil,
				layout.NewSpacer(),
				container.NewVBox(
					container.NewBorder(
						nil,
						nil,
						widget.NewRichTextFromMarkdown("### Allow Content Updates"),
						wUpdates,
					),
					container.NewBorder(
						nil,
						nil,
						widget.NewRichTextFromMarkdown("### Rescan Recheck"),
						wRescanRecheck,
					),
					container.NewBorder(
						nil,
						nil,
						widget.NewRichTextFromMarkdown("### Start Port Range"),
						container.NewStack(
							entrySpacer,
							entryPort,
						),
					),
					container.NewBorder(
						nil,
						nil,
						widget.NewRichTextFromMarkdown("### Search Min Likes %"),
						container.NewStack(
							entrySpacer,
							entryMinLikes,
						),
					),
					container.NewBorder(
						widget.NewRichTextFromMarkdown("### Search Exclusions"),
						nil,
						nil,
						nil,
						entryExclusions,
					),
					rectSpacer,
					rectSpacer,
					container.NewStack(
						linkSpacer,
						linkResetDefaults,
					),
					rectSpacer,
					container.NewStack(
						linkSpacer,
						linkSearchClear,
					),
					rectSpacer,
				),
			),
		),
	)
	settingsBox.SetMinSize(fyne.NewSize(ui.Width, ui.Height*0.36))

	headerSpacer := canvas.NewRectangle(color.Transparent)
	headerSpacer.SetMinSize(fyne.NewSize(0, 35))

	layoutBrowser := container.NewStack(
		rectWidth,
		container.NewHBox(
			layout.NewSpacer(),
			container.NewVBox(
				rectSpacer,
				container.NewHBox(
					results,
					layout.NewSpacer(),
					headerSpacer,
					linkClearHistory,
				),
				rectSpacer,
				rectSpacer,
				entryHistory,
				errorText,
				rectSpacer,
				wSelect,
				rectSpacer,
				historyBox,
				rectSpacer,
				rectSpacer,
				rectSpacer,
				btnShutdown,
			),
			layout.NewSpacer(),
		),
	)

	var historyFound = true
	var historyResults []string

	getHistoryResults := func() {
		if !historyFound {
			return
		}

		historyFound = false
		historyResults = nil
		historyData.Set(nil)
		defer func() {
			historyFound = true
		}()

		if engram.Disk != nil && gnomon.Index != nil {
			for gnomon.Index.LastIndexedHeight < int64(engram.Disk.Get_Daemon_Height()) {
				if !strings.Contains(session.Domain, ".tela") {
					return
				}

				entryHistory.Disable()
				results.Text = "  Gnomon is syncing..."
				results.Color = colors.Yellow
				results.Refresh()
				time.Sleep(time.Second)
			}

			entryHistory.Enable()
			results.Text = "  Loading previous search history..."
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

			tree, err := ss.GetTree("TELA History")
			if err != nil {
				return
			}

			c := tree.Cursor()

			for k, _, err := c.First(); err == nil; k, _, err = c.Next() {
				scid := crypto.HashHexToHash(string(k))

				title, desc, _, _, _ := getContractHeader(scid)

				if title == "" {
					title = scid.String()
				}

				if len(title) > 36 {
					title = title[0:36] + "..."
				}

				if desc == "" {
					desc = "N/A"
				}

				if len(desc) > 40 {
					desc = desc[0:40] + "..."
				}

				historyResults = append(historyResults, title+";;;"+desc+";;;;;;"+scid.String())
			}

			sort.Strings(historyResults)
			history = historyResults
			historyData.Set(history)
			historyList.Refresh()

			results.Text = fmt.Sprintf("  Search History:  %d", len(historyResults))
			results.Color = colors.Green
			results.Refresh()
			btnShutdown.Enable()
		}
	}

	entryHistory.OnChanged = func(s string) {
		if s == "" {
			go getHistoryResults()
			return
		}

		var queryResult []string
		for _, data := range history {
			for _, split := range strings.Split(data, ";;;") {
				if strings.Contains(split, s) {
					queryResult = append(queryResult, data)
					break
				}
			}
		}

		sort.Strings(queryResult)
		history = queryResult
		historyData.Set(history)
		historyList.Refresh()

		results.Text = fmt.Sprintf("  Search History:  %d", len(queryResult))
		results.Color = colors.Green
		results.Refresh()
		entryHistory.Enable()
	}

	wSelect.OnChanged = func(s string) {
		errorText.Text = ""
		errorText.Refresh()
		if !session.Offline {
			btnShutdown.Enable()
		}

		switch s {
		case "Active":
			servingData.Set(nil)

			var serversRunning []string
			for _, serv := range tela.GetServerInfo() {
				serversRunning = append(serversRunning, serv.Name+";;;"+serv.Address+";;;;;;"+serv.SCID)
			}

			sort.Strings(serversRunning)
			servingData.Set(serversRunning)

			if !isSearching {
				if session.Offline {
					results.Text = "  Disabled in offline mode."
					results.Color = colors.Gray
					results.Refresh()
				} else {
					results.Text = fmt.Sprintf("  Active Servers:  %d", len(serversRunning))
					results.Color = colors.Green
					results.Refresh()
				}
			}

			labelLastScan.Text = ""
			labelLastScan.Refresh()
			layoutBrowser.Objects[1].(*fyne.Container).Objects[1].(*fyne.Container).Objects[1].(*fyne.Container).Objects[3] = labelLastScan
			layoutBrowser.Objects[1].(*fyne.Container).Objects[1].(*fyne.Container).Objects[4] = entryServeSCID
			layoutBrowser.Objects[1].(*fyne.Container).Objects[1].(*fyne.Container).Objects[9] = servingBox
			btnShutdown.Text = "Shutdown TELA"
			btnShutdown.Refresh()
		case "History":
			if gnomon.Index == nil {
				results.Text = "  Gnomon is inactive."
				results.Color = colors.Gray
				results.Refresh()
			}

			if isSearching {
				linkClearHistory.Hide()
			} else {
				go getHistoryResults()
				linkClearHistory.Show()
			}

			layoutBrowser.Objects[1].(*fyne.Container).Objects[1].(*fyne.Container).Objects[1].(*fyne.Container).Objects[3] = linkClearHistory
			layoutBrowser.Objects[1].(*fyne.Container).Objects[1].(*fyne.Container).Objects[4] = entryHistory
			layoutBrowser.Objects[1].(*fyne.Container).Objects[1].(*fyne.Container).Objects[9] = historyBox
			btnShutdown.Text = "Shutdown TELA"
			btnShutdown.Refresh()
			servingList.UnselectAll()
		case "Search":
			if gnomon.Index == nil {
				results.Text = "  Gnomon is inactive."
				results.Color = colors.Gray
				results.Refresh()
			}

			go getSearchResults()

			layoutBrowser.Objects[1].(*fyne.Container).Objects[1].(*fyne.Container).Objects[1].(*fyne.Container).Objects[3] = labelLastScan
			layoutBrowser.Objects[1].(*fyne.Container).Objects[1].(*fyne.Container).Objects[4] = entrySearch
			layoutBrowser.Objects[1].(*fyne.Container).Objects[1].(*fyne.Container).Objects[9] = searchBox
			btnShutdown.Text = "Rescan Blockchain"
			btnShutdown.Refresh()
			if isSearching {
				btnShutdown.Disable()
			}
		case "Settings":
			if !isSearching {
				if session.Offline {
					results.Text = "  Disabled in offline mode."
					results.Color = colors.Gray
					results.Refresh()
				} else {
					results.Text = fmt.Sprintf("  Active Servers:  %d", len(tela.GetServerInfo()))
					results.Color = colors.Green
					results.Refresh()
					linkResetDefaults.Show()
					if gnomon.Index != nil {
						wRescanRecheck.Enable()
						entryMinLikes.Enable()
						entryExclusions.Enable()
					}

					if _, err := GetEncryptedValue("TELA Search", []byte("SCIDs")); err == nil {
						linkSearchClear.Show()
					} else {
						linkSearchClear.Hide()
					}
				}
			} else {
				wRescanRecheck.Disable()
				entryMinLikes.Disable()
				entryExclusions.Disable()
				linkResetDefaults.Hide()
				linkSearchClear.Hide()
			}

			labelLastScan.Text = ""
			labelLastScan.Refresh()
			layoutBrowser.Objects[1].(*fyne.Container).Objects[1].(*fyne.Container).Objects[1].(*fyne.Container).Objects[3] = labelLastScan
			layoutBrowser.Objects[1].(*fyne.Container).Objects[1].(*fyne.Container).Objects[4] = entryAddSCID
			layoutBrowser.Objects[1].(*fyne.Container).Objects[1].(*fyne.Container).Objects[9] = settingsBox
			btnShutdown.Text = "Shutdown TELA"
			btnShutdown.Refresh()
			servingList.UnselectAll()
		}
	}

	if session.Offline {
		results.Text = "  Disabled in offline mode."
		results.Color = colors.Gray
		results.Refresh()
		wUpdates.Disable()
		entryServeSCID.Disable()
		entryAddSCID.Disable()
		wRescanRecheck.Disable()
		entryExclusions.Disable()
		entryMinLikes.Disable()
		entryPort.Disable()
		btnShutdown.Disable()
	} else if gnomon.Index == nil {
		results.Text = "  Gnomon is inactive."
		results.Color = colors.Gray
		results.Refresh()
		entryAddSCID.Disable()
		wRescanRecheck.Disable()
		entryExclusions.Disable()
		entryMinLikes.Disable()
	}

	entryServeSCID.OnChanged = func(s string) {
		errorText.Text = ""
		errorText.Refresh()
		if len(s) == 64 {
			go func() {
				// Create a TELALink to parse and get its ratings for user to verifiy before serving the content
				telaLink := TELALink_Params{TelaLink: fmt.Sprintf("tela://open/%s", s)}
				linkPermission, err := AskPermissionForRequestE("Open TELA Link", telaLink)
				if err != nil {
					logger.Errorf("[Engram] Open TELA link: %s\n", err)
					errorText.Text = "error could not open TELA"
					errorText.Color = colors.Red
					errorText.Refresh()
					return
				}

				if linkPermission != xswd.Allow {
					entryServeSCID.SetText("")
					return
				}

				showLoadingOverlay()
				defer func() {
					go refreshServerList()
				}()

				var index tela.INDEX

				// If serving without Gnomon, scid will not end up in history
				if gnomon.Index != nil {
					result := gnomon.GetAllSCIDVariableDetails(s)
					if len(result) == 0 {
						_, err := getTxData(s)
						if err != nil {
							return
						}
					}

					index.NameHdr, index.DescrHdr, _, _, _ = getContractHeader(crypto.HashHexToHash(s))

					if index.NameHdr == "" {
						index.NameHdr = s
					}

					if len(index.NameHdr) > 36 {
						index.NameHdr = index.NameHdr[0:36] + "..."
					}

					if index.DescrHdr == "" {
						index.DescrHdr = "N/A"
					}

					if len(index.DescrHdr) > 40 {
						index.DescrHdr = index.DescrHdr[0:40] + "..."
					}
				}

				entryServeSCID.SetText("")

				if link, err := tela.ServeTELA(s, session.Daemon); err == nil {
					url, err := url.Parse(link)
					if err != nil {
						logger.Errorf("[Engram] TELA URL parse: %s\n", err)
						errorText.Text = "error could parse URL"
						errorText.Color = colors.Red
						errorText.Refresh()
						return // If url is not valid, scid won't be saved in history
					} else {
						err = fyne.CurrentApp().OpenURL(url)
						if err != nil {
							errorText.Text = "error could not open browser"
							errorText.Color = colors.Red
							errorText.Refresh()
						}
					}

					if gnomon.Index != nil {
						historyResults = append(historyResults, index.NameHdr+";;;"+index.DescrHdr+";;;;;;"+s)
						sort.Strings(historyResults)
						history = historyResults
						historyData.Set(history)
						historyList.Refresh()

						results.Text = fmt.Sprintf("  Search History:  %d", len(historyResults))
						results.Color = colors.Green
						results.Refresh()

						err = StoreEncryptedValue("TELA History", []byte(s), []byte(""))
						if err != nil {
							logger.Errorf("[Engram] Error saving TELA search result: %s\n", err)
						}
					}
				} else {
					if strings.Contains(err.Error(), "user defined no updates and content has been updated to") {
						removeOverlays()

						// Create a TELALink to parse and get its ratings for user to verifiy before serving updated content
						telaLink := TELALink_Params{TelaLink: fmt.Sprintf("tela://open/%s", s)}
						linkPermission, err := AskPermissionForRequestE("Allow Updated Content", telaLink)
						if err != nil {
							logger.Errorf("[Engram] Open TELA link: %s\n", err)
							errorText.Text = "error could not open TELA"
							errorText.Color = colors.Red
							errorText.Refresh()
							return
						}

						if linkPermission != xswd.Allow {
							entryServeSCID.SetText("")
							return
						}

						link, err := serveTELAUpdates(s)
						if err != nil {
							logger.Errorf("[Engram] Error serving TELA: %s\n", err)
							errorText.Text = telaErrorToString(err)
							errorText.Color = colors.Red
							errorText.Refresh()
							return
						}

						url, err := url.Parse(link)
						if err != nil {
							logger.Errorf("[Engram] TELA URL parse: %s\n", err)
							errorText.Text = "error could parse URL"
							errorText.Color = colors.Red
							errorText.Refresh()
							return
						} else {
							err = fyne.CurrentApp().OpenURL(url)
							if err != nil {
								errorText.Text = "error could not open browser"
								errorText.Color = colors.Red
								errorText.Refresh()
							}
						}

						if gnomon.Index != nil {
							historyResults = append(historyResults, index.NameHdr+";;;"+index.DescrHdr+";;;;;;"+s)
							sort.Strings(historyResults)
							history = historyResults
							historyData.Set(history)
							historyList.Refresh()

							results.Text = fmt.Sprintf("  Search History:  %d", len(historyResults))
							results.Color = colors.Green
							results.Refresh()

							err = StoreEncryptedValue("TELA History", []byte(s), []byte(""))
							if err != nil {
								logger.Errorf("[Engram] Error saving TELA search result: %s\n", err)
							}
						}

						return
					}

					logger.Errorf("[Engram] Error serving TELA: %s\n", err)
					errorText.Text = telaErrorToString(err)
					errorText.Color = colors.Red
					errorText.Refresh()
				}

				removeOverlays()
			}()
		}
	}

	go getHistoryResults()

	historyList.OnSelected = func(id widget.ListItemID) {
		errorText.Text = ""
		errorText.Refresh()
		showLoadingOverlay()
		defer removeOverlays()

		split := strings.Split(history[id], ";;;")
		if len(split) < 4 || len(split[3]) != 64 {
			logger.Errorf("[Engram] TELA Invalid SCID\n")
			errorText.Text = "invalid TELA scid"
			errorText.Color = colors.Red
			errorText.Refresh()
			return
		}

		index, err := tela.GetINDEXInfo(split[3], session.Daemon)
		if err != nil {
			logger.Errorf("[Engram] GetINDEXInfo: %s\n", err)
			errorText.Text = "invalid INDEX scid"
			errorText.Color = colors.Red
			errorText.Refresh()
			return
		}

		historyList.UnselectAll()
		historyList.FocusLost()
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTELAManager(index, refreshServerList))
	}

	searchList.OnSelected = func(id widget.ListItemID) {
		errorText.Text = ""
		errorText.Refresh()
		showLoadingOverlay()
		defer removeOverlays()

		split := strings.Split(searching[id], ";;;")
		if len(split) < 2 || len(split[1]) != 64 {
			logger.Errorf("[Engram] TELA Invalid SCID\n")
			errorText.Text = "invalid TELA scid"
			errorText.Color = colors.Red
			errorText.Refresh()
			return
		}

		index, err := tela.GetINDEXInfo(split[1], session.Daemon)
		if err != nil {
			logger.Errorf("[Engram] GetINDEXInfo: %s\n", err)
			errorText.Text = "invalid INDEX scid"
			errorText.Color = colors.Red
			errorText.Refresh()
			return
		}

		searchList.UnselectAll()
		searchList.FocusLost()
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTELAManager(index, refreshServerList))
	}

	servingList.OnSelected = func(id widget.ListItemID) {
		errorText.Text = ""
		errorText.Refresh()
		showLoadingOverlay()
		defer removeOverlays()

		split := strings.Split(serving[id], ";;;")
		if len(split) < 4 || len(split[3]) != 64 {
			logger.Errorf("[Engram] TELA Invalid SCID\n")
			errorText.Text = "invalid TELA scid"
			errorText.Color = colors.Red
			errorText.Refresh()
			return
		}

		index, err := tela.GetINDEXInfo(split[3], session.Daemon)
		if err != nil {
			logger.Errorf("[Engram] GetINDEXInfo: %s\n", err)
			errorText.Text = "invalid INDEX scid"
			errorText.Color = colors.Red
			errorText.Refresh()
			return
		}

		servingList.UnselectAll()
		servingList.FocusLost()
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTELAManager(index, refreshServerList))
	}

	top := container.NewVBox(
		rectSpacer,
		rectSpacer,
		container.NewCenter(
			heading,
		),
		rectSpacer,
		rectSpacer,
		container.NewCenter(
			layoutBrowser,
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

	return NewVScroll(layout)
}

// Layout details of a TELA INDEX
func layoutTELAManager(index tela.INDEX, callback func()) fyne.CanvasObject {
	session.Domain = "app.tela.manager"

	frame := &iframe{}

	rectBox := canvas.NewRectangle(color.Transparent)
	rectBox.SetMinSize(fyne.NewSize(ui.MaxWidth*0.99, ui.MaxHeight*0.58))

	rectWidth90 := canvas.NewRectangle(color.Transparent)
	rectWidth90.SetMinSize(fyne.NewSize(ui.Width, 10))

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(fyne.NewSize(6, 5))

	labelName := widget.NewRichTextFromMarkdown(index.NameHdr)
	labelName.Wrapping = fyne.TextWrapOff
	labelName.ParseMarkdown("## " + index.NameHdr)

	labelDesc := widget.NewRichTextFromMarkdown(index.DescrHdr)
	labelDesc.Wrapping = fyne.TextWrapWord

	labelDURL := canvas.NewText("   DURL", colors.Gray)
	labelDURL.TextSize = 14
	labelDURL.Alignment = fyne.TextAlignLeading
	labelDURL.TextStyle = fyne.TextStyle{Bold: true}

	textDURL := widget.NewRichTextFromMarkdown(index.DURL)
	textDURL.Wrapping = fyne.TextWrapWord

	labelSCID := canvas.NewText("   SMART  CONTRACT  ID", colors.Gray)
	labelSCID.TextSize = 14
	labelSCID.Alignment = fyne.TextAlignLeading
	labelSCID.TextStyle = fyne.TextStyle{Bold: true}

	textSCID := widget.NewRichTextFromMarkdown(index.SCID)
	textSCID.Wrapping = fyne.TextWrapWord

	linkViewExplorer := widget.NewHyperlinkWithStyle("View in Explorer", nil, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	linkViewExplorer.OnTapped = func() {
		if engram.Disk.GetNetwork() {
			link, _ := url.Parse("https://explorer.dero.io/tx/" + index.SCID)
			_ = fyne.CurrentApp().OpenURL(link)
		} else {
			link, _ := url.Parse("https://testnetexplorer.dero.io/tx/" + index.SCID)
			_ = fyne.CurrentApp().OpenURL(link)
		}
	}

	linkCopySCID := widget.NewHyperlinkWithStyle("Copy SCID", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkCopySCID.OnTapped = func() {
		session.Window.Clipboard().SetContent(index.SCID)
	}

	labelAuthor := canvas.NewText("   SMART  CONTRACT  AUTHOR", colors.Gray)
	labelAuthor.TextSize = 14
	labelAuthor.Alignment = fyne.TextAlignLeading
	labelAuthor.TextStyle = fyne.TextStyle{Bold: true}

	textAuthor := widget.NewRichTextFromMarkdown(index.Author)
	textAuthor.Wrapping = fyne.TextWrapWord

	linkMessageAuthor := widget.NewHyperlinkWithStyle("Message the Author", nil, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	linkMessageAuthor.OnTapped = func() {
		if index.Author != "" {
			messages.Contact = index.Author
			session.Window.Canvas().SetContent(layoutTransition())
			removeOverlays()
			session.Window.Canvas().SetContent(layoutPM())
		}
	}

	linkCopyAuthor := widget.NewHyperlinkWithStyle("Copy Address", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkCopyAuthor.OnTapped = func() {
		session.Window.Clipboard().SetContent(index.Author)
	}

	labelRatings := canvas.NewText("   TELA  RATINGS", colors.Gray)
	labelRatings.TextSize = 14
	labelRatings.Alignment = fyne.TextAlignLeading
	labelRatings.TextStyle = fyne.TextStyle{Bold: true}

	textLikes := widget.NewRichTextFromMarkdown("Likes:")
	textDislikes := widget.NewRichTextFromMarkdown("Dislikes:")
	textAverage := widget.NewRichTextFromMarkdown("Average:")

	ratingsBox := container.NewVBox(labelRatings)

	ratings, err := tela.GetRating(index.SCID, session.Daemon, 0)
	if err == nil {
		ratingsBox.Add(container.NewHBox(textLikes, canvas.NewText(fmt.Sprintf("%d", ratings.Likes), colors.Green)))
		ratingsBox.Add(container.NewHBox(textDislikes, canvas.NewText(fmt.Sprintf("%d", ratings.Dislikes), colors.Red)))
		ratingsBox.Add(container.NewHBox(textAverage, canvas.NewText(fmt.Sprintf("%0.1f/10", ratings.Average), colors.Account)))
	}

	labelStatus := canvas.NewText("   APPLICATION  STATUS", colors.Gray)
	labelStatus.TextSize = 14
	labelStatus.Alignment = fyne.TextAlignLeading
	labelStatus.TextStyle = fyne.TextStyle{Bold: true}

	textStatus := canvas.NewText("   Offline", colors.Red)
	textStatus.TextSize = 14
	textStatus.Alignment = fyne.TextAlignLeading
	textStatus.TextStyle = fyne.TextStyle{Bold: true}

	labelSeparator := widget.NewRichTextFromMarkdown("")
	labelSeparator.Wrapping = fyne.TextWrapOff
	labelSeparator.ParseMarkdown("---")
	labelSeparator2 := widget.NewRichTextFromMarkdown("")
	labelSeparator2.Wrapping = fyne.TextWrapOff
	labelSeparator2.ParseMarkdown("---")
	labelSeparator3 := widget.NewRichTextFromMarkdown("")
	labelSeparator3.Wrapping = fyne.TextWrapOff
	labelSeparator3.ParseMarkdown("---")
	labelSeparator4 := widget.NewRichTextFromMarkdown("")
	labelSeparator4.Wrapping = fyne.TextWrapOff
	labelSeparator4.ParseMarkdown("---")
	labelSeparator5 := widget.NewRichTextFromMarkdown("")
	labelSeparator5.Wrapping = fyne.TextWrapOff
	labelSeparator5.ParseMarkdown("---")
	labelSeparator6 := widget.NewRichTextFromMarkdown("")
	labelSeparator6.Wrapping = fyne.TextWrapOff
	labelSeparator6.ParseMarkdown("---")

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

	linkBack := widget.NewHyperlinkWithStyle("Back to TELA", nil, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	linkBack.OnTapped = func() {
		removeOverlays()
		capture := session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(session.LastDomain)
		session.Domain = "app.tela"
		session.LastDomain = capture
		go callback()
	}

	image := canvas.NewImageFromResource(resourceTelaIcon)
	image.SetMinSize(fyne.NewSize(ui.Width*0.2, ui.Width*0.2))
	image.FillMode = canvas.ImageFillContain

	if index.IconHdr != "" {
		path, err := fyne.LoadResourceFromURLString(index.IconHdr)
		if err != nil {
			image.Resource = resourceTelaIcon
		} else {
			image.Resource = path
		}

		image.SetMinSize(fyne.NewSize(ui.Width*0.3, ui.Width*0.3))
		image.FillMode = canvas.ImageFillContain
		image.Refresh()
	}

	linkRate := widget.NewHyperlinkWithStyle("Rate SCID", nil, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	linkRate.OnTapped = func() {
		rateTELAOverlay(index.NameHdr, index.SCID)
	}
	linkRate.Hide()

	// Check if wallet has rated SCID
	if gnomon.Index != nil {
		ratingStore, _ := gnomon.GetSCIDValuesByKey(index.SCID, engram.Disk.GetAddress().String())
		if ratingStore == nil {
			linkRate.Show()
		}
	}

	errorText := canvas.NewText(" ", colors.Green)
	errorText.TextSize = 12
	errorText.Alignment = fyne.TextAlignCenter

	spacerStatus := canvas.NewRectangle(color.Transparent)
	spacerStatus.SetMinSize(fyne.NewSize(0, 34))

	linkOpenInBrowser := widget.NewHyperlinkWithStyle("Open in Browser", nil, fyne.TextAlignTrailing, fyne.TextStyle{Bold: true})
	linkOpenInBrowser.Hide()
	linkOpenInBrowser.OnTapped = func() {
		params := fmt.Sprintf("tela://open/%s", index.SCID)
		var toggledUpdates bool
		if !tela.UpdatesAllowed() {
			// user has accepted updated content when serving, call AllowUpdates because OpenTELALink returns error on any updated content
			tela.AllowUpdates(true)
			toggledUpdates = true
		}

		link, err := tela.OpenTELALink(params, session.Daemon)
		if toggledUpdates {
			tela.AllowUpdates(false)
		}
		if err != nil {
			logger.Errorf("[Engram] handling TELA link: %s\n", err)
			errorText.Text = "error handling TELA link"
			errorText.Color = colors.Red
			errorText.Refresh()
			return
		}

		url, err := url.Parse(link)
		if err != nil {
			logger.Errorf("[Engram] TELA URL parse: %s\n", err)
			errorText.Text = "error could parse URL"
			errorText.Color = colors.Red
			errorText.Refresh()
		} else {
			err = fyne.CurrentApp().OpenURL(url)
			if err != nil {
				errorText.Text = "error could not open browser"
				errorText.Color = colors.Red
				errorText.Refresh()
			}
		}
	}

	btnServer := widget.NewButton("Start Application", nil)
	if tela.HasServer(index.DURL) {
		textStatus.Text = "   Online"
		textStatus.Color = colors.Green
		textStatus.Refresh()
		btnServer.Text = "Shutdown Application"
		btnServer.Refresh()
		linkOpenInBrowser.Show()
	}

	btnServer.OnTapped = func() {
		if btnServer.Text != "Start Application" {
			tela.ShutdownServer(index.DURL)
			errorText.Text = ""
			errorText.Refresh()
			textStatus.Text = "   Offline"
			textStatus.Color = colors.Red
			textStatus.Refresh()
			btnServer.Text = "Start Application"
			btnServer.Refresh()
			linkOpenInBrowser.Hide()
		} else {
			showLoadingOverlay()

			if link, err := tela.ServeTELA(index.SCID, session.Daemon); err == nil {
				url, err := url.Parse(link)
				if err != nil {
					logger.Errorf("[Engram] TELA URL parse: %s\n", err)
					errorText.Text = "error could parse URL"
					errorText.Color = colors.Red
					errorText.Refresh()
				} else {
					err = fyne.CurrentApp().OpenURL(url)
					if err != nil {
						errorText.Text = "error could not open browser"
						errorText.Color = colors.Red
						errorText.Refresh()
					}
				}

				textStatus.Text = "   Online"
				textStatus.Color = colors.Green
				textStatus.Refresh()
				btnServer.Text = "Shutdown Application"
				btnServer.Refresh()
				linkOpenInBrowser.Show()

				err = StoreEncryptedValue("TELA History", []byte(index.SCID), []byte(""))
				if err != nil {
					logger.Errorf("[Engram] Error saving TELA search result: %s\n", err)
				}
			} else {
				if strings.Contains(err.Error(), "user defined no updates and content has been updated to") {
					removeOverlays()

					go func() {
						// Create a TELALink to parse and get its ratings for user to verifiy before serving updated content
						telaLink := TELALink_Params{TelaLink: fmt.Sprintf("tela://open/%s", index.SCID)}
						linkPermission, err := AskPermissionForRequestE("Allow Updated Content", telaLink)
						if err != nil {
							logger.Errorf("[Engram] Open TELA link: %s\n", err)
							errorText.Text = "error could not open TELA"
							errorText.Color = colors.Red
							errorText.Refresh()
							return
						}

						if linkPermission != xswd.Allow {
							return
						}

						link, err := serveTELAUpdates(index.SCID)
						if err != nil {
							logger.Errorf("[Engram] Error serving TELA: %s\n", err)
							errorText.Text = telaErrorToString(err)
							errorText.Color = colors.Red
							errorText.Refresh()
							return
						}

						url, err := url.Parse(link)
						if err != nil {
							logger.Errorf("[Engram] TELA URL parse: %s\n", err)
							errorText.Text = "error could parse URL"
							errorText.Color = colors.Red
							errorText.Refresh()
						} else {
							err = fyne.CurrentApp().OpenURL(url)
							if err != nil {
								errorText.Text = "error could not open browser"
								errorText.Color = colors.Red
								errorText.Refresh()
							}
						}

						textStatus.Text = "   Online"
						textStatus.Color = colors.Green
						textStatus.Refresh()
						btnServer.Text = "Shutdown Application"
						btnServer.Refresh()
						linkOpenInBrowser.Show()

						err = StoreEncryptedValue("TELA History", []byte(index.SCID), []byte(""))
						if err != nil {
							logger.Errorf("[Engram] Error saving TELA search result: %s\n", err)
						}
					}()

					return
				}

				logger.Errorf("[Engram] Error serving TELA: %s\n", err)
				errorText.Text = telaErrorToString(err)
				errorText.Color = colors.Red
				errorText.Refresh()
			}

			removeOverlays()
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
						container.NewCenter(
							image,
						),
						rectSpacer,
						container.NewVBox(
							layout.NewSpacer(),
							labelName,
							layout.NewSpacer(),
						),
						rectSpacer,
						rectSpacer,
						labelDesc,
						rectSpacer,
						rectSpacer,
						labelSeparator,
						rectSpacer,
						rectSpacer,
						labelDURL,
						textDURL,
						rectSpacer,
						rectSpacer,
						labelSeparator2,
						rectSpacer,
						rectSpacer,
						labelStatus,
						rectSpacer,
						container.NewHBox(
							textStatus,
							layout.NewSpacer(),
							spacerStatus,
							linkOpenInBrowser,
						),
						rectSpacer,
						errorText,
						rectSpacer,
						btnServer,
						rectSpacer,
						rectSpacer,
						labelSeparator3,
						rectSpacer,
						rectSpacer,
						labelAuthor,
						textAuthor,
						container.NewHBox(
							linkMessageAuthor,
							layout.NewSpacer(),
						),
						container.NewHBox(
							linkCopyAuthor,
							layout.NewSpacer(),
						),
						rectSpacer,
						rectSpacer,
						labelSeparator4,
						rectSpacer,
						rectSpacer,
						labelSCID,
						textSCID,
						container.NewHBox(
							linkViewExplorer,
							layout.NewSpacer(),
						),
						container.NewHBox(
							linkCopySCID,
							layout.NewSpacer(),
						),
						rectSpacer,
						rectSpacer,
						labelSeparator5,
						rectSpacer,
						rectSpacer,
						rectSpacer,
						container.NewHBox(
							layout.NewSpacer(),
						),
						ratingsBox,
						container.NewHBox(
							layout.NewSpacer(),
						),
						container.NewHBox(
							linkRate,
							layout.NewSpacer(),
						),
						rectSpacer,
						rectSpacer,
						labelSeparator6,
						rectSpacer,
						rectSpacer,
						container.NewStack(
							rectWidth90,
						),
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

	return NewVScroll(layout)
}
