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
	session.Domain = appMain
	session.Path = string_
	session.Password = string_

	// Define objects

	btnLogin := widget.NewButton(signIn, nil)

	if session.Error != string_ {
		btnLogin.Text = session.Error
		btnLogin.Disable()
		btnLogin.Refresh()
		session.Error = string_
	}

	btnLogin.OnTapped = func() {
		if session.Path == string_ {
			btnLogin.Text = noAccountSelected
			btnLogin.Disable()
			btnLogin.Refresh()
		} else if session.Password == string_ {
			btnLogin.Text = invalidPassword
			btnLogin.Disable()
			btnLogin.Refresh()
		} else {
			btnLogin.Text = signIn
			btnLogin.Enable()
			btnLogin.Refresh()
			login()
			btnLogin.Text = session.Error
			btnLogin.Disable()
			btnLogin.Refresh()
			session.Error = string_
		}
	}

	btnLogin.Disable()

	session.Window.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
		if session.Domain == appMain ||
			session.Domain == appRegister {
			if k.Name == fyne.KeyReturn {
				if session.Path == string_ {
					btnLogin.Text = noAccountSelected
					btnLogin.Disable()
					btnLogin.Refresh()
				} else if session.Password == string_ {
					btnLogin.Text = invalidPassword
					btnLogin.Disable()
					btnLogin.Refresh()
				} else {
					btnLogin.Text = signIn
					btnLogin.Enable()
					btnLogin.Refresh()
					login()
					btnLogin.Text = invalidPassword
					btnLogin.Disable()
					btnLogin.Refresh()
					session.Error = string_
				}
			}
		} else {
			return
		}
	})

	menuLabel := canvas.NewText(
		moreOptionsBanner,
		colors.Gray,
	)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2, 2))

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2, 2))

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkCreate := widget.NewHyperlinkWithStyle(
		createAccount,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)

	linkCreate.OnTapped = func() {
		session.Domain = appCreate
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutNewAccount())
		removeOverlays()
	}

	linkRecover := widget.NewHyperlinkWithStyle(
		recoverAccountExisting,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkRecover.OnTapped = func() {
		session.Domain = appRestore
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutRestore())
		removeOverlays()
	}

	linkSettings := widget.NewHyperlinkWithStyle(
		stringSettings,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkSettings.OnTapped = func() {
		session.Domain = appSettings
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutSettings())
		removeOverlays()
	}

	modeData := binding.BindBool(&session.Offline)
	mode := widget.NewCheckWithData(offlineMode, modeData)
	mode.OnChanged = func(b bool) {
		if b {
			session.Offline = true
		} else {
			session.Offline = false
		}
	}

	footer := canvas.NewText(
		copyrightNoticeVersion+version.String(),
		colors.Gray,
	)
	footer.TextSize = 10
	footer.Alignment = fyne.TextAlignCenter
	footer.TextStyle = fyne.TextStyle{Bold: true}

	wPassword := NewReturnEntry()
	wPassword.OnReturn = btnLogin.OnTapped
	wPassword.Password = true
	wPassword.OnChanged = func(s string) {
		session.Error = string_
		btnLogin.Text = signIn
		btnLogin.Enable()
		btnLogin.Refresh()
		session.Password = s

		if len(s) < 1 {
			btnLogin.Disable()
			btnLogin.Refresh()
		} else if session.Path == string_ {
			btnLogin.Disable()
			btnLogin.Refresh()
		} else {
			btnLogin.Enable()
		}

		btnLogin.Refresh()
	}
	wPassword.SetPlaceHolder(stringPassword)

	// Get account databases in app directory
	list, err := GetAccounts()
	if err != nil {
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutAlert(2))
	}

	// Populate the accounts in dropdown menu
	wAccount := widget.NewSelect(list, nil)
	wAccount.PlaceHolder = selectAccount
	wAccount.OnChanged = func(s string) {
		session.Error = string_
		btnLogin.Text = signIn
		btnLogin.Refresh()

		// OnChange set wallet path
		if session.Testnet {
			session.Path = filepath.Join(AppPath(), stringtestnet) + string(filepath.Separator) + s
		} else {
			session.Path = filepath.Join(AppPath(), stringmainnet) + string(filepath.Separator) + s
		}

		if session.Password != string_ {
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

	wSpacer := widget.NewLabel(singlespace)

	rectStatus := canvas.NewRectangle(color.Transparent)
	rectStatus.SetMinSize(
		fyne.NewSize(
			10,
			10,
		),
	)

	res.bg2.SetMinSize(
		fyne.NewSize(
			ui.Width,
			ui.MaxHeight*0.2,
		),
	)
	res.bg2.Refresh()

	headerBox := canvas.NewRectangle(color.Transparent)
	headerBox.SetMinSize(
		fyne.NewSize(
			ui.Width,
			1,
		),
	)

	frame := &iframe{}

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(
		fyne.NewSize(
			ui.Width,
			5,
		),
	)

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
	resizeWindow(
		ui.MaxWidth,
		ui.MaxHeight,
	)

	session.Dashboard = stringmain
	session.Domain = appWallet

	address := engram.Disk.GetAddress().String()
	short := address[len(address)-10:]
	shard := fmt.Sprintf(
		"%x",
		sha1.Sum(
			[]byte(address),
		),
	)
	shard = shard[len(shard)-10:]
	wSpacer := widget.NewLabel(singlespace)

	session.Balance, _ = engram.Disk.Get_Balance()
	session.BalanceText = canvas.NewText(
		walletapi.FormatMoney(session.Balance),
		colors.Green,
	)
	session.BalanceText.TextSize = 28
	session.BalanceText.TextStyle = fyne.TextStyle{Bold: true}

	if session.BalanceUSD == string_ {
		session.BalanceUSDText = canvas.NewText(
			string_,
			colors.Gray,
		)
		session.BalanceUSDText.TextSize = 14
		session.BalanceUSDText.TextStyle = fyne.TextStyle{Bold: true}
	} else {
		session.BalanceUSDText = canvas.NewText(
			usdWithPad+session.BalanceUSD,
			colors.Gray,
		)
		session.BalanceUSDText.TextSize = 14
		session.BalanceUSDText.TextStyle = fyne.TextStyle{Bold: true}
	}

	network := string_
	if session.Testnet {
		network = testnetBanner
	} else {
		network = mainnetBanner
	}

	rectStatus := canvas.NewRectangle(color.Transparent)
	rectStatus.SetMinSize(
		fyne.NewSize(10, 10))
	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(
		fyne.NewSize(
			ui.Width, 20))
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

	path := strings.Split(
		session.Path,
		string(filepath.Separator),
	)
	accountName := canvas.NewText(
		path[len(path)-1],
		colors.Green,
	)
	accountName.TextStyle = fyne.TextStyle{Bold: true}
	accountName.TextSize = 18

	shortAddress := canvas.NewText(
		fourdots+short,
		colors.Gray,
	)
	shortAddress.TextStyle = fyne.TextStyle{Bold: true}
	shortAddress.TextSize = 22

	gramSend := widget.NewButton(centeredSend, nil)

	heading := canvas.NewText(
		balanceBanner,
		colors.Gray,
	)
	heading.TextSize = 16
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	sendDesc := canvas.NewText(addTxDetails, colors.Gray)
	sendDesc.TextSize = 18
	sendDesc.Alignment = fyne.TextAlignCenter
	sendDesc.TextStyle = fyne.TextStyle{Bold: true}

	sendHeading := canvas.NewText(
		sendMoney,
		colors.Green,
	)
	sendHeading.TextSize = 22
	sendHeading.Alignment = fyne.TextAlignCenter
	sendHeading.TextStyle = fyne.TextStyle{Bold: true}

	headerLabel := canvas.NewText(
		doublespace+network+doublespace,
		colors.Gray,
	)
	headerLabel.TextSize = 11
	headerLabel.Alignment = fyne.TextAlignCenter
	headerLabel.TextStyle = fyne.TextStyle{Bold: true}

	statusLabel := canvas.NewText(statusBanner, colors.Gray)
	statusLabel.TextSize = 11
	statusLabel.Alignment = fyne.TextAlignCenter
	statusLabel.TextStyle = fyne.TextStyle{Bold: true}

	daemonLabel := canvas.NewText(stringOFFLINE, colors.Gray)
	daemonLabel.TextSize = 12
	daemonLabel.Alignment = fyne.TextAlignCenter
	daemonLabel.TextStyle = fyne.TextStyle{Bold: false}

	if !session.Offline {
		daemonLabel.Text = session.Daemon
	}

	session.WalletHeight = engram.Disk.Get_Height()
	session.StatusText = canvas.NewText(
		fmt.Sprintf(
			"%d",
			session.WalletHeight,
		),
		colors.Gray,
	)
	session.StatusText.TextSize = 12
	session.StatusText.Alignment = fyne.TextAlignCenter
	session.StatusText.TextStyle = fyne.TextStyle{Bold: false}

	menuLabel := canvas.NewText(
		modulesBanner,
		colors.Gray,
	)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2, 2))

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2, 2))

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkLogout := widget.NewHyperlinkWithStyle(
		signOut,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkLogout.OnTapped = func() {
		closeWallet()
	}

	linkHistory := widget.NewHyperlinkWithStyle(
		viewHistory,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkHistory.OnTapped = func() {
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutHistory())
		removeOverlays()
	}

	menu := widget.NewSelect(
		[]string{
			stringIdentity,
			myAccount,
			stringMessages,
			stringTransfers,
			assetExplorer,
			stringServices,
			stringCyberdeck,
			stringDatapad,
			singlespace,
		},
		nil,
	)
	menu.PlaceHolder = selectModule
	menu.OnChanged = func(s string) {
		if s == myAccount {
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutAccount())
			removeOverlays()
		} else if s == stringTransfers {
			session.Window.Canvas().SetContent(layoutTransition())
			session.Window.Canvas().SetContent(layoutTransfers())
			removeOverlays()
		} else if s == assetExplorer {
			session.Window.Canvas().SetContent(layoutTransition())
			session.Window.Canvas().SetContent(layoutAssetExplorer())
			removeOverlays()
		} else if s == stringDatapad {
			session.Window.Canvas().SetContent(layoutTransition())
			session.Window.Canvas().SetContent(layoutDatapad())
			removeOverlays()
		} else if s == stringMessages {
			session.Window.Canvas().SetContent(layoutTransition())
			session.Window.Canvas().SetContent(layoutMessages())
			removeOverlays()
		} else if s == stringCyberdeck {
			session.Window.Canvas().SetContent(layoutTransition())
			session.Window.Canvas().SetContent(layoutCyberdeck())
			removeOverlays()
		} else if s == stringIdentity {
			session.Window.Canvas().SetContent(layoutTransition())
			session.Window.Canvas().SetContent(layoutIdentity())
			removeOverlays()
		} else if s == stringServices {
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutServiceAddress())
			removeOverlays()
		} else {
			session.Window.Canvas().SetContent(layoutTransition())
			session.Window.Canvas().SetContent(layoutDashboard())
			removeOverlays()
		}

		session.LastDomain = session.Window.Content()
	}

	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(
		fyne.NewSize(
			ui.Width,
			260,
		),
	)

	rect300 := canvas.NewRectangle(color.Transparent)
	rect300.SetMinSize(
		fyne.NewSize(
			ui.Width,
			50,
		),
	)

	rect.SetMinSize(
		fyne.NewSize(
			ui.Width,
			30,
		),
	)

	res.gram.SetMinSize(
		fyne.NewSize(
			ui.Width,
			150,
		),
	)

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(
		fyne.NewSize(
			10,
			5,
		),
	)

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
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutSend())
		removeOverlays()
	}

	session.Window.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
		if session.Domain != appWallet {
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
	session.Domain = appSend

	wSpacer := widget.NewLabel(singlespace)
	frame := &iframe{}

	btnSend := widget.NewButton(stringSave, nil)

	wAmount := widget.NewEntry()
	wAmount.SetPlaceHolder(stringAmount)

	wMessage := widget.NewEntry()
	wMessage.SetValidationError(nil)
	wMessage.SetPlaceHolder(stringMessage)
	wMessage.Validator = func(s string) error {
		bytes := []byte(s)
		if len(bytes) <= 130 {
			tx.Comment = s
			wMessage.SetValidationError(nil)
			return nil
		} else {
			err := errors.New(msgTooLong)
			wMessage.SetValidationError(err)
			return err
		}
	}

	wPaymentID := widget.NewEntry()
	wPaymentID.Validator = func(s string) (err error) {
		tx.PaymentID, err = strconv.ParseUint(
			s,
			10,
			64,
		)
		if err != nil {
			wPaymentID.SetValidationError(err)
			tx.PaymentID = 0
		}

		return
	}
	wPaymentID.SetPlaceHolder(payIDSlashServicePort)

	options := []string{
		anonSetNone,
		anonSetLower,
		anonSetLess,
		anonSetRecommended,
		anonSetMore,
		anonSetHigh,
		anonSetMost,
	}
	wRings := widget.NewSelect(options, nil)

	wReceiver := widget.NewEntry()
	wReceiver.SetPlaceHolder(receiverContact)
	wReceiver.SetValidationError(nil)
	wReceiver.Validator = func(s string) error {
		address, err := globals.ParseValidateAddress(s)
		if err != nil {
			tx.Address = nil
			_, addr, _ := checkUsername(s, -1)
			if addr == string_ {
				btnSend.Disable()
				err = errors.New(invalidContact)
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

				if address.Arguments.HasValue(
					rpc.RPC_VALUE_TRANSFER,
					rpc.DataUint64,
				) {
					amount := address.Arguments[address.Arguments.Index(
						rpc.RPC_VALUE_TRANSFER,
						rpc.DataUint64,
					)].Value
					tx.Amount = amount.(uint64)
					wAmount.Text = globals.FormatMoney(
						amount.(uint64),
					)
					if amount.(uint64) != 0.00000 {
						wAmount.Disable()
					}
					wAmount.Refresh()
				}

				if address.Arguments.HasValue(
					rpc.RPC_DESTINATION_PORT,
					rpc.DataUint64,
				) {
					port := address.Arguments[address.Arguments.Index(
						rpc.RPC_DESTINATION_PORT,
						rpc.DataUint64,
					)].Value
					tx.PaymentID = port.(uint64)
					wPaymentID.Text = strconv.FormatUint(
						port.(uint64),
						10,
					)
					wPaymentID.Disable()
					wPaymentID.Refresh()
				}

				if address.Arguments.HasValue(
					rpc.RPC_COMMENT,
					rpc.DataString,
				) {
					comment := address.Arguments[address.Arguments.Index(
						rpc.RPC_COMMENT,
						rpc.DataString,
					)].Value
					tx.Comment = comment.(string)
					wMessage.Text = comment.(string)
					if comment.(string) != string_ {
						wMessage.Disable()
					}
					wMessage.Refresh()
				}

				if tx.Ringsize == 0 {
					wRings.SetSelected(anonSetRecommended)
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

	/*
		// TODO
		wAll := widget.NewCheck(" All", func(b bool) {
			if b {
				tx.Amount = engram.Disk.GetAccount().Balance_Mature
				wAmount.SetText(walletapi.FormatMoney(tx.Amount))
			} else {
				tx.Amount = 0
				wAmount.SetText(string_)
			}
		})
	*/

	wAmount.Validator = func(s string) error {
		if s == string_ {
			tx.Amount = 0
			wAmount.SetValidationError(
				errors.New(invalidTxAmount),
			)
			btnSend.Disable()
		} else {
			balance, _ := engram.Disk.Get_Balance()
			entry, err := globals.ParseAmount(s)
			if err != nil {
				tx.Amount = 0
				wAmount.SetValidationError(
					errors.New(invalidTxAmount),
				)
				btnSend.Disable()
				return errors.New(invalidTxAmount)
			}

			if entry == 0 {
				tx.Amount = 0
				wAmount.SetValidationError(
					errors.New(invalidTxAmount),
				)
				btnSend.Disable()
				return errors.New(invalidTxAmount)
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
				wAmount.SetValidationError(
					errors.New(
						insufficientFunds),
				)
			}
			return nil
		}
		return errors.New(invalidTxAmount)
	}

	wAmount.SetValidationError(nil)

	wRings.PlaceHolder = selectAnonymity
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
				wReceiver.SetValidationError(
					errors.New(invalidAddress),
				)
				wReceiver.Refresh()
			}
		}
	}

	sendHeading := canvas.NewText(sendMoney, colors.Green)
	sendHeading.TextSize = 22
	sendHeading.Alignment = fyne.TextAlignCenter
	sendHeading.TextStyle = fyne.TextStyle{Bold: true}

	optionalLabel := canvas.NewText(optionalBanner, colors.Gray)
	optionalLabel.TextSize = 11
	optionalLabel.Alignment = fyne.TextAlignCenter
	optionalLabel.TextStyle = fyne.TextStyle{Bold: true}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2, 2))

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2, 2))

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkCancel := widget.NewHyperlinkWithStyle(
		stringCancel,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)

	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(
		fyne.NewSize(
			ui.Width,
			260,
		),
	)

	rect300 := canvas.NewRectangle(color.Transparent)
	rect300.SetMinSize(
		fyne.NewSize(
			ui.Width,
			30,
		),
	)

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(
		fyne.NewSize(
			10, 5,
		),
	)

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
	session.Domain = appService

	wSpacer := widget.NewLabel(singlespace)
	frame := &iframe{}

	btnCreate := widget.NewButton(stringCreate, nil)

	wPaymentID := widget.NewEntry()

	wReceiver := widget.NewEntry()
	wReceiver.Text = engram.Disk.GetAddress().String()
	wReceiver.Disable()

	tx.Address, _ = globals.ParseValidateAddress(engram.Disk.GetAddress().String())

	wReceiver.SetPlaceHolder(receiverContact)
	wReceiver.SetValidationError(nil)

	wAmount := widget.NewEntry()
	wAmount.SetPlaceHolder(stringAmount)

	wMessage := widget.NewEntry()
	wMessage.SetPlaceHolder(stringMessage)
	wMessage.Validator = func(s string) (err error) {
		bytes := []byte(s)
		if len(bytes) <= 130 {
			tx.Comment = s
		} else {
			err = errors.New(msgTooLong)
			wMessage.SetValidationError(err)
		}

		return
	}

	wAmount.Validator = func(s string) error {
		if s == string_ {
			tx.Amount = 0
			wAmount.SetValidationError(
				errors.New(invalidTxAmount),
			)
			btnCreate.Disable()
		} else {
			amount, err := globals.ParseAmount(s)
			if err != nil {
				tx.Amount = 0
				wAmount.SetValidationError(
					errors.New(invalidTxAmount),
				)
				btnCreate.Disable()
				return errors.New(invalidTxAmount)
			}
			wAmount.SetValidationError(nil)
			tx.Amount = amount
			btnCreate.Enable()

			return nil
		}
		return errors.New(invalidTxAmount)
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
			if wReceiver.Text != string_ {
				btnCreate.Enable()
				wPaymentID.SetValidationError(nil)
				return
			} else {
				err = errors.New(emptyPaymentID)
				wPaymentID.SetValidationError(err)
				return
			}
		}
	}
	wPaymentID.SetPlaceHolder(payIDSlashServicePort)

	sendHeading := canvas.NewText(
		newServiceAddress,
		colors.Green,
	)
	sendHeading.TextSize = 22
	sendHeading.Alignment = fyne.TextAlignCenter
	sendHeading.TextStyle = fyne.TextStyle{Bold: true}

	optionalLabel := canvas.NewText(optionalBanner, colors.Gray)
	optionalLabel.TextSize = 11
	optionalLabel.Alignment = fyne.TextAlignCenter
	optionalLabel.TextStyle = fyne.TextStyle{Bold: true}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2, 2))

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2, 2))

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkCancel := widget.NewHyperlinkWithStyle(
		stringCancel,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)

	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(
		fyne.NewSize(
			ui.Width,
			260,
		),
	)

	rect300 := canvas.NewRectangle(color.Transparent)
	rect300.SetMinSize(
		fyne.NewSize(
			ui.Width,
			30,
		),
	)

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(
		fyne.NewSize(10, 5))

	btnCreate.OnTapped = func() {
		var err error
		if tx.Address != nil && tx.PaymentID != 0 {
			if wAmount.Text != string_ {
				_, err = globals.ParseAmount(wAmount.Text)
			}

			if err == nil {
				header := canvas.NewText(
					createServiceAddress,
					colors.Gray,
				)
				header.TextSize = 14
				header.Alignment = fyne.TextAlignCenter
				header.TextStyle = fyne.TextStyle{Bold: true}

				subHeader := canvas.NewText(
					successfullyCreated,
					colors.Account,
				)
				subHeader.TextSize = 22
				subHeader.Alignment = fyne.TextAlignCenter
				subHeader.TextStyle = fyne.TextStyle{Bold: true}

				labelAddress := canvas.NewText(
					integratedBanner,
					colors.Gray,
				)
				labelAddress.TextSize = 12
				labelAddress.Alignment = fyne.TextAlignCenter
				labelAddress.TextStyle = fyne.TextStyle{Bold: true}

				btnCopy := widget.NewButton(copyServiceAddress, nil)

				valueAddress := widget.NewRichTextFromMarkdown(string_)
				valueAddress.Wrapping = fyne.TextWrapBreak

				address := engram.Disk.GetRandomIAddress8()
				address.Arguments = nil
				address.Arguments = append(
					address.Arguments,
					rpc.Argument{
						Name:     rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
						DataType: rpc.DataUint64,
						Value:    uint64(1),
					},
				)
				address.Arguments = append(
					address.Arguments,
					rpc.Argument{
						Name:     rpc.RPC_VALUE_TRANSFER,
						DataType: rpc.DataUint64,
						Value:    tx.Amount,
					},
				)
				address.Arguments = append(
					address.Arguments,
					rpc.Argument{
						Name:     rpc.RPC_DESTINATION_PORT,
						DataType: rpc.DataUint64,
						Value:    tx.PaymentID,
					},
				)
				address.Arguments = append(
					address.Arguments,
					rpc.Argument{
						Name:     rpc.RPC_COMMENT,
						DataType: rpc.DataString,
						Value:    tx.Comment,
					},
				)

				err := address.Arguments.Validate_Arguments()
				if err != nil {
					fmt.Printf(errService, err)
					subHeader.Text = stringError
					subHeader.Refresh()
					btnCopy.Disable()
				} else {
					fmt.Printf(
						newIntegrated,
						address.String(),
					)
					fmt.Printf(
						integratedArgs,
						address.Arguments,
					)

					valueAddress.ParseMarkdown(string_ + address.String())
					valueAddress.Refresh()
				}

				btnCopy.OnTapped = func() {
					session.Window.Clipboard().SetContent(address.String())
				}

				linkClose := widget.NewHyperlinkWithStyle(
					goBack,
					nil,
					fyne.TextAlignCenter,
					fyne.TextStyle{
						Bold: true,
					},
				)
				linkClose.OnTapped = func() {
					overlay := session.Window.Canvas().Overlays()
					overlay.Top().Hide()
					overlay.Remove(overlay.Top())
					overlay.Remove(overlay.Top())
				}

				span := canvas.NewRectangle(color.Transparent)
				span.SetMinSize(
					fyne.NewSize(
						ui.Width, 10))

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
				wReceiver.SetValidationError(
					errors.New(invalidAddress),
				)
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
	resizeWindow(
		ui.MaxWidth,
		ui.MaxHeight)
	a.Settings().SetTheme(themes.alt)

	session.Domain = appRegister
	session.Language = -1
	session.Error = string_
	session.Name = string_
	session.Password = string_
	session.PasswordConfirm = string_

	languages := mnemonics.Language_List()

	errorText := canvas.NewText(
		singlespace,
		colors.Green,
	)
	errorText.TextSize = 12
	errorText.Alignment = fyne.TextAlignCenter

	btnCreate := widget.NewButton(stringCreate, nil)
	btnCreate.Disable()

	linkCancel := widget.NewHyperlinkWithStyle(
		returntoLogin,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkCancel.OnTapped = func() {
		session.Domain = appMain
		session.Error = string_
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutMain())
		removeOverlays()
	}

	btnCopySeed := widget.NewButton(copyRecovery, nil)
	btnCopyAddress := widget.NewButton(copyAddress, nil)

	if !a.Driver().Device().IsMobile() {
		session.Window.Canvas().SetOnTypedKey(
			func(k *fyne.KeyEvent) {
				if session.Domain != appRegister {
					return
				}

				if k.Name == fyne.KeyReturn {
					errorText.Text = string_
					errorText.Refresh()
					create()
					errorText.Text = session.Error
					errorText.Refresh()
				}
			},
		)
	}

	wPassword := NewReturnEntry()
	wPassword.Password = true
	wPassword.OnChanged = func(s string) {
		session.Error = string_
		errorText.Text = string_
		errorText.Refresh()
		session.Password = s

		if len(session.Password) > 0 &&
			session.Password == session.PasswordConfirm &&
			!findAccount() &&
			session.Language != -1 {
			btnCreate.Enable()
			btnCreate.Refresh()
		} else {
			btnCreate.Disable()
			btnCreate.Refresh()
		}
	}
	wPassword.SetPlaceHolder(stringPassword)
	wPassword.Wrapping = fyne.TextWrap(fyne.TextTruncateClip)

	wPasswordConfirm := NewReturnEntry()
	wPasswordConfirm.Password = true
	wPasswordConfirm.OnChanged = func(s string) {
		session.Error = string_
		errorText.Text = string_
		errorText.Refresh()
		session.PasswordConfirm = s

		if len(session.Password) > 0 &&
			session.Password == session.PasswordConfirm &&
			!findAccount() &&
			session.Language != -1 {
			btnCreate.Enable()
			btnCreate.Refresh()
		} else {
			btnCreate.Disable()
			btnCreate.Refresh()
		}
	}
	wPasswordConfirm.SetPlaceHolder(confirmPassword)
	wPasswordConfirm.Wrapping = fyne.TextWrap(fyne.TextTruncateClip)

	wAccount := widget.NewEntry()
	wAccount.SetPlaceHolder(accountName)
	wAccount.Validator = func(s string) (err error) {
		session.Error = string_
		errorText.Text = string_
		errorText.Refresh()

		if len(s) > 25 {
			err = errors.New(accountTooLong)
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

		if getTestnet() {
			session.Path = filepath.Join(AppPath(), stringtestnet, s+".db")
		} else {
			session.Path = filepath.Join(AppPath(), stringmainnet, s+".db")
		}
		session.Name = s

		if findAccount() {
			err = errors.New(accountExists)
			errorText.Text = err.Error()
			errorText.Color = colors.Red
			errorText.Refresh()
			return
		} else {
			errorText.Text = string_
			errorText.Refresh()
		}

		if len(session.Password) > 0 &&
			session.Password == session.PasswordConfirm &&
			!findAccount() &&
			session.Language != -1 {
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

		if len(session.Password) > 0 &&
			session.Password == session.PasswordConfirm &&
			!findAccount() &&
			session.Language != -1 {
			btnCreate.Enable()
			btnCreate.Refresh()
		} else {
			btnCreate.Disable()
			btnCreate.Refresh()
		}
	}
	wLanguage.PlaceHolder = selectLanguage

	wSpacer := widget.NewLabel(singlespace)
	heading := canvas.NewText(
		newAccount,
		colors.Green,
	)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	heading2 := canvas.NewText(
		stringRecovery,
		colors.Green,
	)
	heading2.TextSize = 22
	heading2.Alignment = fyne.TextAlignCenter
	heading2.TextStyle = fyne.TextStyle{Bold: true}

	rectStatus := canvas.NewRectangle(color.Transparent)
	rectStatus.SetMinSize(
		fyne.NewSize(
			10,
			10,
		),
	)

	rectHeader := canvas.NewRectangle(color.Transparent)
	rectHeader.SetMinSize(
		fyne.NewSize(
			ui.Width,
			10,
		),
	)

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(
		fyne.NewSize(
			ui.Width,
			5,
		),
	)

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

	body := widget.NewLabel(recoveryWarning)
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
	scrollBox.SetMinSize(
		fyne.NewSize(
			ui.Width,
			ui.Height*0.70,
		),
	)

	btnCreate.OnTapped = func() {
		if findAccount() {
			errorText.Text = accountExists
			errorText.Color = colors.Red
			errorText.Refresh()
			return
		} else {
			errorText.Text = string_
			errorText.Refresh()
		}

		address, seed, err := create()
		if err != nil {
			errorText.Text = session.Error
			errorText.Refresh()
			return
		}

		formatted := strings.Split(
			seed,
			singlespace,
		)

		rect := canvas.NewRectangle(
			color.RGBA{
				21,
				27,
				36,
				255,
			},
		)
		rect.SetMinSize(
			fyne.NewSize(
				ui.Width,
				25,
			),
		)

		for i := 0; i < len(formatted); i++ {
			pos := fmt.Sprintf(
				"%d",
				i+1,
			)
			word := strings.ReplaceAll(
				formatted[i],
				singlespace,
				string_,
			)
			grid.Add(container.NewStack(
				rect,
				container.NewHBox(
					widget.NewLabel(singlespace),
					widget.NewLabelWithStyle(
						pos,
						fyne.TextAlignCenter,
						fyne.TextStyle{
							Bold: true,
						},
					),
					layout.NewSpacer(),
					widget.NewLabelWithStyle(
						word,
						fyne.TextAlignLeading,
						fyne.TextStyle{
							Bold: true,
						},
					),
					widget.NewLabel(singlespace),
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
	resizeWindow(
		ui.MaxWidth,
		ui.MaxHeight)
	a.Settings().SetTheme(themes.main)

	session.Domain = appRestore
	session.Language = -1
	session.Error = string_
	session.Name = string_
	session.Password = string_
	session.PasswordConfirm = string_

	var seed [25]string

	scrollBox := container.NewVScroll(nil)

	errorText := canvas.NewText(singlespace, colors.Green)
	errorText.TextSize = 12
	errorText.Alignment = fyne.TextAlignCenter

	btnCreate := widget.NewButton(stringRecover, nil)
	btnCreate.Disable()

	linkReturn := widget.NewHyperlinkWithStyle(
		returntoLogin,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkReturn.OnTapped = func() {
		session.Domain = appMain
		session.Error = string_
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutMain())
		removeOverlays()
	}

	btnCopyAddress := widget.NewButton(copyAddress, nil)

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
		session.Error = string_
		errorText.Text = string_
		errorText.Refresh()
		session.Password = s

		if len(session.Password) > 0 &&
			session.Password == session.PasswordConfirm &&
			session.Name != string_ {

		} else {
			btnCreate.Disable()
			btnCreate.Refresh()
		}
	}
	wPassword.SetPlaceHolder(stringPassword)
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
		session.Error = string_
		errorText.Text = string_
		errorText.Refresh()
		session.PasswordConfirm = s

		if len(session.Password) > 0 &&
			session.Password == session.PasswordConfirm &&
			session.Name != string_ {

		} else {
			btnCreate.Disable()
			btnCreate.Refresh()
		}
	}
	wPasswordConfirm.SetPlaceHolder(confirmPassword)
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
		wLanguage.PlaceHolder = selectLanguage
	*/

	wAccount.SetPlaceHolder(accountName)
	wAccount.Wrapping = fyne.TextWrap(fyne.TextTruncateClip)
	wAccount.Validator = func(s string) (err error) {
		session.Error = string_
		errorText.Text = string_
		errorText.Refresh()

		if len(s) > 30 {
			err = errors.New(accountLength)
			wAccount.SetText(err.Error())
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

		if getTestnet() {
			session.Path = filepath.Join(AppPath(), stringtestnet) + string(filepath.Separator) + s + ".db"
		} else {
			session.Path = filepath.Join(AppPath(), stringmainnet) + string(filepath.Separator) + s + ".db"
		}
		session.Name = s

		if findAccount() {
			err = errors.New(accountExists)
			errorText.Text = err.Error()
			errorText.Refresh()
			return
		}

		if len(session.Password) > 0 &&
			session.Password == session.PasswordConfirm &&
			session.Name != string_ {

		} else {
			btnCreate.Disable()
			btnCreate.Refresh()
		}
		return nil
	}

	wSpacer := widget.NewLabel(singlespace)
	heading := canvas.NewText(
		recoverAccount,
		colors.Green,
	)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	heading2 := canvas.NewText(
		stringSuccess,
		colors.Green,
	)
	heading2.TextSize = 22
	heading2.Alignment = fyne.TextAlignCenter
	heading2.TextStyle = fyne.TextStyle{Bold: true}

	rectStatus := canvas.NewRectangle(color.Transparent)
	rectStatus.SetMinSize(
		fyne.NewSize(
			10,
			10,
		),
	)

	rectHeader := canvas.NewRectangle(color.Transparent)
	rectHeader.SetMinSize(
		fyne.NewSize(
			ui.Width,
			10,
		),
	)

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(
		fyne.NewSize(
			ui.Width,
			5,
		),
	)

	status.Connection.FillColor = colors.Gray
	status.Cyberdeck.FillColor = colors.Gray
	status.Sync.FillColor = colors.Gray

	grid := container.NewVBox()
	grid.Objects = nil

	word1 := NewMobileEntry()
	word1.PlaceHolder = seedWord1
	word1.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New(invalidSeedWord)
		}
		seed[0] = s
		return nil
	}
	word1.OnFocusGained = func() {
		offset := word1.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			scrollBox.Offset = fyne.NewPos(
				0,
				offset,
			)
			scrollBox.Refresh()
		}
	}

	word2 := NewMobileEntry()
	word2.PlaceHolder = seedWord2
	word2.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New(invalidSeedWord)
		}
		seed[1] = s

		var list []string
		for s := range seed {
			if seed[s] != string_ {
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
			scrollBox.Offset = fyne.NewPos(
				0,
				offset,
			)
			scrollBox.Refresh()
		}
	}

	word3 := NewMobileEntry()
	word3.PlaceHolder = seedWord3
	word3.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New(invalidSeedWord)
		}
		seed[2] = s

		var list []string
		for s := range seed {
			if seed[s] != string_ {
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
	word4.PlaceHolder = seedWord4
	word4.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New(invalidSeedWord)
		}
		seed[3] = s

		var list []string
		for s := range seed {
			if seed[s] != string_ {
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
			scrollBox.Offset = fyne.NewPos(
				0,
				offset,
			)
			scrollBox.Refresh()
		}
	}

	word5 := NewMobileEntry()
	word5.PlaceHolder = seedWord5
	word5.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New(invalidSeedWord)
		}
		seed[4] = s

		var list []string
		for s := range seed {
			if seed[s] != string_ {
				list = append(
					list,
					seed[s],
				)
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
			scrollBox.Offset = fyne.NewPos(
				0,
				offset,
			)
			scrollBox.Refresh()
		}
	}

	word6 := NewMobileEntry()
	word6.PlaceHolder = seedWord6
	word6.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New(invalidSeedWord)
		}
		seed[5] = s

		var list []string
		for s := range seed {
			if seed[s] != string_ {
				list = append(
					list,
					seed[s],
				)
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
			fmt.Printf(scrollBefore,
				scrollBox.Offset.Y,
			)
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
			fmt.Printf(scrollAfter,
				scrollBox.Offset.Y,
			)
		}
		fmt.Printf(
			offsetFloat,
			offset,
		)
	}

	word7 := NewMobileEntry()
	word7.PlaceHolder = seedWord7
	word7.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New(invalidSeedWord)
		}
		seed[6] = s

		var list []string
		for s := range seed {
			if seed[s] != string_ {
				list = append(
					list,
					seed[s],
				)
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
			fmt.Printf(
				scrollBefore,
				scrollBox.Offset.Y,
			)
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
			fmt.Printf(
				scrollAfter,
				scrollBox.Offset.Y,
			)
		}
		fmt.Printf(
			offsetFloat,
			offset,
		)
	}

	word8 := NewMobileEntry()
	word8.PlaceHolder = seedWord8
	word8.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New(invalidSeedWord)
		}
		seed[7] = s

		var list []string
		for s := range seed {
			if seed[s] != string_ {
				list = append(
					list,
					seed[s],
				)
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
			scrollBox.Offset = fyne.NewPos(
				0,
				offset,
			)
			scrollBox.Refresh()
		}
	}

	word9 := NewMobileEntry()
	word9.PlaceHolder = seedWord9
	word9.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New(invalidSeedWord)
		}
		seed[8] = s

		var list []string
		for s := range seed {
			if seed[s] != string_ {
				list = append(
					list,
					seed[s],
				)
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
			scrollBox.Offset = fyne.NewPos(
				0,
				offset,
			)
			scrollBox.Refresh()
		}
	}

	word10 := NewMobileEntry()
	word10.PlaceHolder = seedWord10
	word10.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New(invalidSeedWord)
		}
		seed[9] = s

		var list []string
		for s := range seed {
			if seed[s] != string_ {
				list = append(
					list,
					seed[s],
				)
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
			scrollBox.Offset = fyne.NewPos(
				0,
				offset,
			)
			scrollBox.Refresh()
		}
	}

	word11 := NewMobileEntry()
	word11.PlaceHolder = seedWord11
	word11.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New(invalidSeedWord)
		}
		seed[10] = s

		var list []string
		for s := range seed {
			if seed[s] != string_ {
				list = append(
					list,
					seed[s],
				)
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
			scrollBox.Offset = fyne.NewPos(
				0,
				offset,
			)
			scrollBox.Refresh()
		}
	}

	word12 := NewMobileEntry()
	word12.PlaceHolder = seedWord12
	word12.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New(invalidSeedWord)
		}
		seed[11] = s

		var list []string
		for s := range seed {
			if seed[s] != string_ {
				list = append(
					list,
					seed[s],
				)
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
			scrollBox.Offset = fyne.NewPos(
				0,
				offset,
			)
			scrollBox.Refresh()
		}
	}

	word13 := NewMobileEntry()
	word13.PlaceHolder = seedWord13
	word13.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New(invalidSeedWord)
		}
		seed[12] = s

		var list []string
		for s := range seed {
			if seed[s] != string_ {
				list = append(
					list,
					seed[s],
				)
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
	word14.PlaceHolder = seedWord14
	word14.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New(invalidSeedWord)
		}
		seed[13] = s

		var list []string
		for s := range seed {
			if seed[s] != string_ {
				list = append(
					list,
					seed[s],
				)
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
			scrollBox.Offset = fyne.NewPos(
				0,
				offset,
			)
			scrollBox.Refresh()
		}
	}

	word15 := NewMobileEntry()
	word15.PlaceHolder = seedWord15
	word15.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New(invalidSeedWord)
		}
		seed[14] = s

		var list []string
		for s := range seed {
			if seed[s] != string_ {
				list = append(
					list,
					seed[s],
				)
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
			scrollBox.Offset = fyne.NewPos(
				0,
				offset,
			)
			scrollBox.Refresh()
		}
	}

	word16 := NewMobileEntry()
	word16.PlaceHolder = seedWord16
	word16.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New(invalidSeedWord)
		}
		seed[15] = s

		var list []string
		for s := range seed {
			if seed[s] != string_ {
				list = append(
					list,
					seed[s],
				)
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
			scrollBox.Offset = fyne.NewPos(
				0,
				offset,
			)
			scrollBox.Refresh()
		}
	}

	word17 := NewMobileEntry()
	word17.PlaceHolder = seedWord17
	word17.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New(invalidSeedWord)
		}
		seed[16] = s

		var list []string
		for s := range seed {
			if seed[s] != string_ {
				list = append(
					list,
					seed[s],
				)
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
			scrollBox.Offset = fyne.NewPos(
				0,
				offset,
			)
			scrollBox.Refresh()
		}
	}

	word18 := NewMobileEntry()
	word18.PlaceHolder = seedWord18
	word18.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New(invalidSeedWord)
		}
		seed[17] = s

		var list []string
		for s := range seed {
			if seed[s] != string_ {
				list = append(
					list,
					seed[s],
				)
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
			scrollBox.Offset = fyne.NewPos(
				0,
				offset,
			)
			scrollBox.Refresh()
		}
	}

	word19 := NewMobileEntry()
	word19.PlaceHolder = seedWord19
	word19.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New(invalidSeedWord)
		}
		seed[18] = s

		var list []string
		for s := range seed {
			if seed[s] != string_ {
				list = append(
					list,
					seed[s],
				)
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
	word20.PlaceHolder = seedWord20
	word20.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New(invalidSeedWord)
		}
		seed[19] = s
		return nil
	}
	word20.OnFocusGained = func() {
		offset := word20.Position().Y
		if offset-scrollBox.Offset.Y > scrollBox.MinSize().Height {
			scrollBox.Offset = fyne.NewPos(
				0,
				offset,
			)
			scrollBox.Refresh()
		}
	}

	word21 := NewMobileEntry()
	word21.PlaceHolder = "Seed Word 21"
	word21.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New(invalidSeedWord)
		}
		seed[20] = s

		var list []string
		for s := range seed {
			if seed[s] != string_ {
				list = append(
					list,
					seed[s],
				)
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
			scrollBox.Offset = fyne.NewPos(
				0,
				offset,
			)
			scrollBox.Refresh()
		}
	}

	word22 := NewMobileEntry()
	word22.PlaceHolder = seedWord22
	word22.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New(invalidSeedWord)
		}
		seed[21] = s

		var list []string
		for s := range seed {
			if seed[s] != string_ {
				list = append(
					list,
					seed[s],
				)
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
			scrollBox.Offset = fyne.NewPos(
				0,
				offset,
			)
			scrollBox.Refresh()
		}
	}

	word23 := NewMobileEntry()
	word23.PlaceHolder = seedWord23
	word23.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New(invalidSeedWord)
		}
		seed[22] = s

		var list []string
		for s := range seed {
			if seed[s] != string_ {
				list = append(
					list,
					seed[s],
				)
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
			scrollBox.Offset = fyne.NewPos(
				0,
				offset,
			)
			scrollBox.Refresh()
		}
	}

	word24 := NewMobileEntry()
	word24.PlaceHolder = seedWord24
	word24.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New(invalidSeedWord)
		}
		seed[23] = s

		var list []string
		for s := range seed {
			if seed[s] != string_ {
				list = append(
					list,
					seed[s],
				)
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
			scrollBox.Offset = fyne.NewPos(
				0,
				offset,
			)
			scrollBox.Refresh()
		}
	}

	word25 := NewMobileEntry()
	word25.PlaceHolder = seedWord25
	word25.Validator = func(s string) error {
		if !checkSeedWord(s) {
			btnCreate.Disable()
			return errors.New(invalidSeedWord)
		}
		seed[24] = s

		var list []string
		for s := range seed {
			if seed[s] != string_ {
				list = append(
					list,
					seed[s],
				)
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
			fmt.Printf(scrollBefore, scrollBox.Offset.Y)
			scrollBox.Offset = fyne.NewPos(0, offset)
			scrollBox.Refresh()
			fmt.Printf(scrollAfter, scrollBox.Offset.Y)
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

	body := widget.NewLabel(accountRecoverySuccess)
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

	scrollBox.SetMinSize(
		fyne.NewSize(
			ui.Width,
			ui.Height*0.60,
		),
	)

	btnCreate.OnTapped = func() {
		if engram.Disk != nil {
			closeWallet()
		}

		var err error

		if findAccount() {
			err = errors.New(accountExists)
			errorText.Text = err.Error()
			errorText.Color = colors.Red
			errorText.Refresh()
			return
		} else {
			errorText.Text = string_
			errorText.Refresh()
		}

		getTestnet()

		var words string

		for i := 0; i < 25; i++ {
			words += seed[i] + singlespace
		}

		language, _, err := mnemonics.Words_To_Key(words)

		temp, err := walletapi.Create_Encrypted_Wallet_From_Recovery_Words(
			session.Path,
			session.Password,
			words,
		)
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
		session.Path = string_
		session.Name = string_
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
	rect1.SetMinSize(
		fyne.NewSize(
			ui.Width,
			1,
		),
	)

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
	session.Domain = appExplorer

	var data []string
	var listData binding.StringList
	var listBox *widget.List

	frame := &iframe{}
	rectLeft := canvas.NewRectangle(color.Transparent)
	rectLeft.SetMinSize(
		fyne.NewSize(
			ui.Width*0.40,
			35,
		),
	)
	rectRight := canvas.NewRectangle(color.Transparent)
	rectRight.SetMinSize(
		fyne.NewSize(
			ui.Width*0.58,
			35,
		),
	)
	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(
		fyne.NewSize(
			ui.Width,
			ui.Height*0.47,
		),
	)
	rectWidth := canvas.NewRectangle(color.Transparent)
	rectWidth.SetMinSize(
		fyne.NewSize(
			ui.Width,
			10,
		),
	)

	heading := canvas.NewText(
		assetExplorer,
		colors.Green,
	)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(
		fyne.NewSize(
			6,
			5,
		),
	)

	results := canvas.NewText(
		string_,
		colors.Green,
	)
	results.TextSize = 13

	listData = binding.BindStringList(&data)
	listBox = widget.NewListWithData(listData,
		func() fyne.CanvasObject {
			return container.NewStack(
				container.NewHBox(
					container.NewStack(
						rectLeft,
						widget.NewLabel(string_),
					),
					container.NewStack(
						rectRight,
						widget.NewLabel(string_),
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

			split := strings.Split(
				str,
				threesemicolons,
			)

			co.(*fyne.Container).
				Objects[0].(*fyne.Container).
				Objects[0].(*fyne.Container).
				Objects[1].(*widget.Label).
				SetText(split[0])
			co.(*fyne.Container).
				Objects[0].(*fyne.Container).
				Objects[1].(*fyne.Container).
				Objects[1].(*widget.Label).
				SetText(split[1])
			//co.(*fyne.Container).Objects[3].(*fyne.Container).Objects[1].(*widget.Label).SetText(split[3])
		})

	menu := widget.NewSelect(
		[]string{
			myAssets,
			searchSCID,
		},
		nil,
	)
	menu.PlaceHolder = selectOne

	menuLabel := canvas.NewText(
		moreOptionsBanner,
		colors.Gray,
	)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	entrySCID := widget.NewEntry()
	entrySCID.PlaceHolder = searchSCID

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2,
			2,
		),
	)

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2,
			2,
		),
	)

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	btnSearch := widget.NewButton(
		stringSearch,
		nil,
	)
	btnSearch.OnTapped = func() {

	}

	linkBack := widget.NewHyperlinkWithStyle(
		backtoDash,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkBack.OnTapped = func() {
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutDashboard())
		removeOverlays()
	}

	btnMyAssets := widget.NewButton(
		myAssets,
		nil,
	)
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

	results.Text = fmt.Sprintf(

		centeredResults,

		found,
	)
	results.Color = colors.Green
	results.Refresh()

	listData.Set(nil)

	if session.Offline {
		results.Text = disabledOffline
		results.Color = colors.Gray
		results.Refresh()
	} else if gnomon.Index == nil {
		results.Text = disabledGnomon
		results.Color = colors.Gray
		results.Refresh()
	}

	entrySCID.OnChanged = func(s string) {
		if entrySCID.Text != string_ && len(s) == 64 {
			result := gnomon.Index.GravDBBackend.GetSCIDVariableDetailsAtTopoheight(
				s,
				engram.Disk.Get_Daemon_TopoHeight(),
			)

			if len(result) == 0 {
				_, err := getTxData(s)
				if err != nil {
					return
				}
			}

			showLoadingOverlay()

			err := StoreEncryptedValue(
				explorerHistory,
				[]byte(s),
				[]byte(string_),
			)
			if err != nil {
				fmt.Printf(
					errorSavingSearch,
					err,
				)
				return
			}

			scid := crypto.HashHexToHash(s)

			bal, _, err := engram.Disk.GetDecryptedBalanceAtTopoHeight(
				scid,
				-1,
				engram.Disk.GetAddress().String(),
			)

			title, desc, _, _, _ := getContractHeader(scid)

			if title == string_ {
				title = scid.String()
			}

			if len(title) > 18 {
				title = title[0:18] + threeperiods
			}

			if desc == string_ {
				desc = notAvaliable
			}

			if len(desc) > 40 {
				desc = desc[0:40] + threeperiods
			}

			assetData = append(
				data,
				globals.FormatMoney(bal)+
					threesemicolons+
					title+
					threesemicolons+
					desc+
					sixsemicolons+
					scid.String(),
			)
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

				entrySCID.Text = string_
				entrySCID.Refresh()

				results.Text = fmt.Sprintf(
					centeredResults,
					found)
				results.Color = colors.Green
				results.Refresh()
			*/

			entrySCID.SetText(string_)
			session.LastDomain = session.Window.Content()
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutAssetManager(s))
			removeOverlays()
		}
	}

	go func() {
		if engram.Disk != nil && gnomon.Index != nil {
			for gnomon.Index.LastIndexedHeight < int64(engram.Disk.Get_Daemon_Height()) {
				if session.Domain != appExplorer {
					break
				}
				results.Text = fmt.Sprintf(
					gnomonCenter,
					gnomon.Index.LastIndexedHeight,
					int64(engram.Disk.Get_Daemon_Height()),
				)
				results.Color = colors.Yellow
				results.Refresh()
				time.Sleep(time.Second * 1)
			}

			results.Text = fmt.Sprintf(loadingHistory)
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

			tree, err := ss.GetTree(explorerHistory)
			if err != nil {
				return
			}

			c := tree.Cursor()

			for k, _, err := c.First(); err == nil; k, _, err = c.Next() {
				scid := crypto.HashHexToHash(string(k))

				bal, _, err := engram.Disk.GetDecryptedBalanceAtTopoHeight(
					scid,
					-1,
					engram.Disk.GetAddress().String(),
				)
				if err != nil {
					bal = 0
				}

				title, desc, _, _, _ := getContractHeader(scid)

				if title == string_ {
					title = scid.String()
				}

				if len(title) > 18 {
					title = title[0:18] + threeperiods
				}

				if desc == string_ {
					desc = notAvaliable
				}

				if len(desc) > 40 {
					desc = desc[0:40] + threeperiods
				}

				assetData = append(
					data,
					globals.FormatMoney(bal)+
						threesemicolons+
						title+
						threesemicolons+
						desc+
						sixsemicolons+
						scid.String(),
				)
				listData.Set(assetData)
				found += 1
			}
		}

		results.Text = fmt.Sprintf(
			centeredSearch,
			found,
		)
		results.Color = colors.Green
		results.Refresh()

		listData.Set(assetData)

		listBox.OnSelected = func(id widget.ListItemID) {
			split := strings.Split(
				assetData[id],
				threesemicolons,
			)
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

	return layout
}

func layoutMyAssets() fyne.CanvasObject {
	var data []string
	var listData binding.StringList
	var listBox *widget.List

	frame := &iframe{}
	rectLeft := canvas.NewRectangle(color.Transparent)
	rectLeft.SetMinSize(
		fyne.NewSize(
			ui.Width*0.40,
			35,
		),
	)
	rectRight := canvas.NewRectangle(color.Transparent)
	rectRight.SetMinSize(
		fyne.NewSize(
			ui.Width*0.59,
			35,
		),
	)
	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(
		fyne.NewSize(
			ui.Width,
			ui.Height*0.55,
		),
	)
	rectWidth := canvas.NewRectangle(color.Transparent)
	rectWidth.SetMinSize(
		fyne.NewSize(
			ui.MaxWidth, 10,
		),
	)

	heading := canvas.NewText(
		myAssets,
		colors.Green,
	)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(
		fyne.NewSize(
			6,
			5,
		),
	)

	results := canvas.NewText(
		string_,
		colors.Green,
	)
	results.TextSize = 13

	listData = binding.BindStringList(&data)
	listBox = widget.NewListWithData(listData,
		func() fyne.CanvasObject {
			return container.NewStack(
				container.NewHBox(
					container.NewStack(
						rectLeft,
						widget.NewLabel(string_),
					),
					container.NewStack(
						rectRight,
						widget.NewLabel(string_),
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

			split := strings.Split(str, threesemicolons)

			co.(*fyne.Container).
				Objects[0].(*fyne.Container).
				Objects[0].(*fyne.Container).
				Objects[1].(*widget.Label).
				SetText(split[0])
			co.(*fyne.Container).
				Objects[0].(*fyne.Container).
				Objects[1].(*fyne.Container).
				Objects[1].(*widget.Label).
				SetText(split[1])
			//co.(*fyne.Container).Objects[3].(*fyne.Container).Objects[1].(*widget.Label).SetText(split[3])
		})

	menuLabel := canvas.NewText(moreOptionsBanner, colors.Gray)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	entrySCID := widget.NewEntry()
	entrySCID.PlaceHolder = searchSCID

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2,
			2,
		),
	)

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2,
			2,
		),
	)

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkBack := widget.NewHyperlinkWithStyle(
		backtoExplorer,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkBack.OnTapped = func() {
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutAssetExplorer())
		removeOverlays()
	}

	btnRescan := widget.NewButton(
		rescanBlockchain,
		nil,
	)
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
		results.Text = disabledTrackingOffline
		results.Color = colors.Gray
		results.Refresh()
	} else if gnomon.Index == nil {
		results.Text = disabledTrackingGnomon
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

			results.Text = gatheringSCIDs
			results.Color = colors.Yellow
			results.Refresh()

			for gnomon.Index.LastIndexedHeight < int64(engram.Disk.Get_Daemon_Height()) {
				results.Text = fmt.Sprintf(
					gnomonCenter,
					gnomon.Index.LastIndexedHeight,
					int64(engram.Disk.Get_Daemon_Height()),
				)
				results.Color = colors.Yellow
				results.Refresh()
				time.Sleep(time.Second * 1)
			}

			results.Text = fmt.Sprintf(loadingScan)
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

			tree, err := ss.GetTree(myAssets)
			if err != nil {
				return
			}

			c := tree.Cursor()

			for k, _, err := c.First(); err == nil; k, _, err = c.Next() {
				scid := string(k)

				hash := crypto.HashHexToHash(scid)

				bal, _, err := engram.Disk.GetDecryptedBalanceAtTopoHeight(
					hash,
					-1,
					engram.Disk.GetAddress().String(),
				)
				if err != nil {
					return
				} else {
					title, desc, _, _, _ := getContractHeader(hash)

					if title == string_ {
						title = scid
					}

					if len(title) > 18 {
						title = title[0:18] + threeperiods
					}

					if desc == string_ {
						desc = notAvaliable
					}

					if len(desc) > 40 {
						desc = desc[0:40] + threeperiods
					}

					balance := globals.FormatMoney(bal)
					assetData = append(data,
						balance+
							threesemicolons+
							title+
							threesemicolons+
							desc+
							sixsemicolons+
							scid,
					)
					listData.Set(assetData)
					owned += 1
				}
			}

			rescan := func() {
				btnRescan.Disable()
				assetTotal = 0
				assetCount = 0

				t := time.Now()
				timeNow := string(
					t.Format(
						time.RFC822,
					),
				)
				StoreEncryptedValue(
					assetScan,
					byteLastScan,
					[]byte(timeNow),
				)

				results.Text = fmt.Sprintf(indexingResults)
				results.Color = colors.Yellow
				results.Refresh()

				owned = 0

				assetData = []string{}
				listBox.UnselectAll()
				listData.Set(assetData)

				assetList = gnomon.Index.GravDBBackend.GetAllOwnersAndSCIDs()

				for len(assetList) < 5 {
					fmt.Printf(
						gnomonAssetScan,
						gnomon.Index.LastIndexedHeight,
						engram.Disk.Get_Daemon_Height(),
						len(assetList),
					)
					results.Color = colors.Yellow
					assetList = gnomon.Index.GravDBBackend.GetAllOwnersAndSCIDs()
					time.Sleep(time.Second * 5)
				}

				results.Text = fmt.Sprintf(indexingComplete)
				results.Color = colors.Yellow
				results.Refresh()

				assetList = gnomon.Index.GravDBBackend.GetAllOwnersAndSCIDs()

				contracts := []crypto.Hash{}

				for sc := range assetList {
					scid := crypto.HashHexToHash(sc)

					if !scid.IsZero() {
						assetCount += 1
						contracts = append(
							contracts,
							scid,
						)
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

						desc := string_
						title := string_

						assetTotal += 1

						results.Text = scanningSCIDs + fmt.Sprintf(
							"%d / %d",
							assetTotal,
							assetCount,
						)
						results.Color = colors.Yellow
						results.Refresh()

						balance := globals.FormatMoney(0)

						bal, _, err := engram.Disk.GetDecryptedBalanceAtTopoHeight(scid, -1, engram.Disk.GetAddress().String())
						if err != nil {
							return
						} else {
							balance = globals.FormatMoney(bal)

							if bal != zerobal {
								err = StoreEncryptedValue(
									myAssets,
									[]byte(scid.String()),
									[]byte(balance),
								)
								if err != nil {
									fmt.Printf(
										errFailedStore,
										err,
									)
								}

								title, desc, _, _, _ = getContractHeader(scid)

								if title == string_ {
									title = scid.String()
								}

								if len(title) > 20 {
									title = title[0:20] + threeperiods
								}

								if desc == string_ {
									desc = notAvaliable
								}

								if len(desc) > 40 {
									desc = desc[0:40] + threeperiods
								}

								owned += 1
								assetData = append(
									assetData,
									balance+
										threesemicolons+
										title+
										threesemicolons+
										desc+
										sixsemicolons+
										scid.String(),
								)
								listData.Set(assetData)
								fmt.Printf(
									fountAssets,
									scid.String(),
								)
							}
						}
					}(i)

					lastJob += 1
				}

				wg.Wait()

				if lastJob < len(contracts) {
					goto parse
				}

				results.Text = fmt.Sprintf(
					centeredOwned,
					owned,
					timeNow,
				)
				results.Color = colors.Green
				results.Refresh()

				listData.Set(assetData)
				btnRescan.Enable()
			}

			btnRescan.OnTapped = rescan

			lastScan, _ := GetEncryptedValue(
				assetScan,
				byteLastScan,
			)

			if len(assetData) == 0 && len(lastScan) == 0 {
				rescan()
			}

			if len(lastScan) > 0 {
				results.Text = fmt.Sprintf(
					centeredOwnedScanned,
					owned,
					lastScan,
				)
			} else {
				results.Text = fmt.Sprintf(
					centeredOwned,
					owned,
				)
			}

			results.Color = colors.Green
			results.Refresh()

			listData.Set(assetData)

			listBox.OnSelected = func(id widget.ListItemID) {
				split := strings.Split(
					assetData[id],
					threesemicolons,
				)

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
				session.Window.SetContent(
					layoutAssetManager(
						split[4],
					),
				)
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
	session.Domain = appManager

	wSpacer := widget.NewLabel(singlespace)

	frame := &iframe{}

	rectBox := canvas.NewRectangle(color.Transparent)
	rectBox.SetMinSize(
		fyne.NewSize(
			ui.MaxWidth*0.99,
			ui.MaxHeight*0.58,
		),
	)
	rectWidth90 := canvas.NewRectangle(color.Transparent)
	rectWidth90.SetMinSize(
		fyne.NewSize(
			ui.Width, 10,
		),
	)

	heading := canvas.NewText(
		assetsManager, colors.Green,
	)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(
		fyne.NewSize(
			6,
			5,
		),
	)

	labelSigner := canvas.NewText(
		scAUTHOR, colors.Gray,
	)
	labelSigner.TextSize = 14
	labelSigner.Alignment = fyne.TextAlignLeading
	labelSigner.TextStyle = fyne.TextStyle{Bold: true}

	labelOwner := canvas.NewText(
		scOWNER,
		colors.Gray,
	)
	labelOwner.TextSize = 14
	labelOwner.Alignment = fyne.TextAlignLeading
	labelOwner.TextStyle = fyne.TextStyle{Bold: true}

	labelSCID := canvas.NewText(
		scID,
		colors.Gray,
	)
	labelSCID.TextSize = 14
	labelSCID.Alignment = fyne.TextAlignLeading
	labelSCID.TextStyle = fyne.TextStyle{Bold: true}

	labelBalance := canvas.NewText(
		assetBALANCE,
		colors.Gray,
	)
	labelBalance.TextSize = 14
	labelBalance.Alignment = fyne.TextAlignLeading
	labelBalance.TextStyle = fyne.TextStyle{Bold: true}

	labelTransfer := canvas.NewText(
		transferASSET,
		colors.Gray,
	)
	labelTransfer.TextSize = 14
	labelTransfer.Alignment = fyne.TextAlignLeading
	labelTransfer.TextStyle = fyne.TextStyle{Bold: true}

	labelExecute := canvas.NewText(
		executeACTION,
		colors.Gray,
	)
	labelExecute.TextSize = 14
	labelExecute.Alignment = fyne.TextAlignLeading
	labelExecute.TextStyle = fyne.TextStyle{Bold: true}

	menuLabel := canvas.NewText(
		moreOptionsBanner,
		colors.Gray,
	)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	entryAddress := widget.NewEntry()
	entryAddress.PlaceHolder = usernameAddress

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2,
			2,
		),
	)

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2,
			2,
		),
	)

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	sc := widget.NewLabel(scid)
	sc.Wrapping = fyne.TextWrap(fyne.TextWrapWord)

	hash := crypto.HashHexToHash(scid)
	name, desc, icon, owner, code := getContractHeader(hash)

	if owner == string_ {
		owner = doubledashes
	}

	signer := doubledashes

	result, err := getTxData(scid)
	if err != nil {
		signer = doubledashes
	} else {
		signer = result.Txs[0].Signer
	}

	labelSeparator := widget.NewRichTextFromMarkdown(string_)
	labelSeparator.Wrapping = fyne.TextWrapOff
	labelSeparator.ParseMarkdown(threedashes)

	labelSeparator2 := widget.NewRichTextFromMarkdown(string_)
	labelSeparator2.Wrapping = fyne.TextWrapOff
	labelSeparator2.ParseMarkdown(threedashes)

	labelSeparator3 := widget.NewRichTextFromMarkdown(string_)
	labelSeparator3.Wrapping = fyne.TextWrapOff
	labelSeparator3.ParseMarkdown(threedashes)

	labelSeparator4 := widget.NewRichTextFromMarkdown(string_)
	labelSeparator4.Wrapping = fyne.TextWrapOff
	labelSeparator4.ParseMarkdown(threedashes)

	labelSeparator5 := widget.NewRichTextFromMarkdown(string_)
	labelSeparator5.Wrapping = fyne.TextWrapOff
	labelSeparator5.ParseMarkdown(threedashes)

	labelSeparator6 := widget.NewRichTextFromMarkdown(string_)
	labelSeparator6.Wrapping = fyne.TextWrapOff
	labelSeparator6.ParseMarkdown(threedashes)

	labelName := widget.NewRichTextFromMarkdown(name)
	labelName.Wrapping = fyne.TextWrapOff
	labelName.ParseMarkdown(doublehashes + name)

	labelDesc := widget.NewRichTextFromMarkdown(desc)
	labelDesc.Wrapping = fyne.TextWrapWord
	labelDesc.ParseMarkdown(desc)

	textSigner := widget.NewRichTextFromMarkdown(owner)
	textSigner.Wrapping = fyne.TextWrapWord
	textSigner.ParseMarkdown(signer)

	textOwner := widget.NewRichTextFromMarkdown(owner)
	textOwner.Wrapping = fyne.TextWrapWord
	textOwner.ParseMarkdown(owner)

	btnSend := widget.NewButton(sendAsset, nil)

	entryAddress.Validator = func(s string) error {
		btnSend.Text = sendAsset
		btnSend.Refresh()
		_, err := globals.ParseValidateAddress(s)
		if err != nil {
			go func() {
				exists, _, err := checkUsername(s, -1)
				if err != nil && !exists {
					btnSend.Disable()
					entryAddress.SetValidationError(
						errors.New(invalidContact),
					)
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
	entryAmount.PlaceHolder = assetAmount
	entryAmount.Validator = func(s string) error {
		if s != string_ {
			amount, err := globals.ParseAmount(s)
			if err != nil {
				btnSend.Disable()
				entryAmount.SetValidationError(
					errors.New(invalidAmount),
				)
				return err
			} else {
				bal, _, err := engram.Disk.GetDecryptedBalanceAtTopoHeight(
					hash,
					-1,
					engram.Disk.GetAddress().String(),
				)
				if err != nil {
					btnSend.Disable()
					entryAmount.SetValidationError(
						errors.New(errorParsingAssetBal),
					)
					return err
				} else {
					if amount > bal || amount == 0 {
						err = errors.New(insufficientAssetBal)
						btnSend.Text = insufficientTxAmount
						btnSend.Disable()
						entryAmount.SetValidationError(err)
						return err
					}
				}
			}
		}

		btnSend.Text = sendAsset
		btnSend.Enable()
		entryAmount.SetValidationError(nil)

		return nil
	}

	var zerobal uint64

	balance := canvas.NewText(
		fmt.Sprintf(
			"  %d",
			zerobal,
		),
		colors.Green,
	)
	balance.TextSize = 20
	balance.TextStyle = fyne.TextStyle{Bold: true}

	btnSend.OnTapped = func() {
		btnSend.Text = settingTransfer
		btnSend.Disable()
		btnSend.Refresh()

		txid, err := transferAsset(
			hash,
			entryAddress.Text,
			entryAmount.Text,
		)
		if err != nil {
			entryAddress.Text = string_
			entryAddress.Refresh()
			entryAmount.Text = string_
			entryAmount.Refresh()
			btnSend.Text = txFailed
			btnSend.Disable()
			btnSend.Refresh()
		} else {
			entryAddress.Text = string_
			entryAddress.Refresh()
			entryAmount.Text = string_
			entryAmount.Refresh()
			btnSend.Text = txConfirming
			btnSend.Disable()
			btnSend.Refresh()

			go func() {
				walletapi.WaitNewHeightBlock()
				sHeight := walletapi.Get_Daemon_Height()

				for session.Domain == appManager {
					result := engram.Disk.Get_Payments_TXID(txid.String())

					if result.TXID != txid.String() {
						time.Sleep(time.Second * 1)
					} else {
						break
					}
				}

				// If we go DEFAULT_CONFIRMATION_TIMEOUT blocks without exiting 'Confirming...' loop, display failed to transfer and break
				if walletapi.Get_Daemon_Height() > sHeight+int64(DEFAULT_CONFIRMATION_TIMEOUT) {
					entryAddress.Text = string_
					entryAddress.Refresh()
					entryAmount.Text = string_
					entryAmount.Refresh()
					btnSend.Text = txFailed
					btnSend.Disable()
					btnSend.Refresh()
					return
				}

				// If daemon height has incremented, print retry counters into button space
				if walletapi.Get_Daemon_Height()-sHeight > 0 {
					btnSend.Text = fmt.Sprintf(
						txConfirmingSomeofSome,
						walletapi.Get_Daemon_Height()-sHeight,
						DEFAULT_CONFIRMATION_TIMEOUT,
					)
					btnSend.Refresh()
				}

				bal, _, err := engram.Disk.GetDecryptedBalanceAtTopoHeight(
					hash,
					-1,
					engram.Disk.GetAddress().String(),
				)
				if err == nil {
					err = StoreEncryptedValue(
						myAssets,
						[]byte(hash.String()),
						[]byte(globals.FormatMoney(bal)),
					)
					if err != nil {
						fmt.Printf(
							errStoring,
							hash,
						)
					}
					balance.Text = doublespace + globals.FormatMoney(bal)
					balance.Refresh()
				}

				if bal != zerobal {
					btnSend.Text = sendAsset
					btnSend.Enable()
					btnSend.Refresh()
				} else {
					btnSend.Text = dontOwnAsset
					btnSend.Disable()
					btnSend.Refresh()
				}
			}()
		}
	}

	bal, _, err := engram.Disk.GetDecryptedBalanceAtTopoHeight(
		hash,
		-1,
		engram.Disk.GetAddress().String(),
	)
	if err == nil {
		balance.Text = doublespace + globals.FormatMoney(bal)
		balance.Refresh()

		if bal == zerobal {
			entryAddress.Disable()
			entryAmount.Disable()
			btnSend.Text = dontOwnAsset
			btnSend.Disable()
		}
	}

	linkBack := widget.NewHyperlinkWithStyle(
		backtoExplorer,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)
	linkBack.OnTapped = func() {
		removeOverlays()
		capture := session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(session.LastDomain)
		session.Domain = appExplorer
		session.LastDomain = capture
	}

	var image *canvas.Image
	image = canvas.NewImageFromResource(resourceBlockGrayPng)
	image.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2,
			ui.Width*0.2,
		),
	)

	image.FillMode = canvas.ImageFillContain

	if icon != string_ {
		var path fyne.Resource
		path, err = fyne.LoadResourceFromURLString(icon)
		if err != nil {
			image.Resource = resourceBlockGrayPng
		} else {
			image.Resource = path
		}

		image.SetMinSize(
			fyne.NewSize(
				ui.Width*0.2, ui.Width*0.2))
		image.FillMode = canvas.ImageFillContain
		image.Refresh()
	}

	if name == string_ {
		labelName.ParseMarkdown(nameNotProvided)
	}

	if desc == string_ {
		labelDesc.ParseMarkdown(noDescription)
	}

	if bal != zerobal {
		btnSend.Text = sendAsset
		btnSend.Enable()
	} else {
		btnSend.Text = dontOwnAsset
		btnSend.Disable()
	}
	btnSend.Refresh()

	linkCopySigner := widget.NewHyperlinkWithStyle(
		copyAddress,
		nil,
		fyne.TextAlignLeading,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkCopySigner.OnTapped = func() {
		session.Window.Clipboard().SetContent(signer)
	}

	linkCopyOwner := widget.NewHyperlinkWithStyle(
		copyAddress,
		nil,
		fyne.TextAlignLeading,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkCopyOwner.OnTapped = func() {
		session.Window.Clipboard().SetContent(owner)
	}

	linkMessageAuthor := widget.NewHyperlinkWithStyle(
		msgAuthor,
		nil,
		fyne.TextAlignLeading,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkMessageAuthor.OnTapped = func() {
		if signer != string_ && signer != doubledashes {
			messages.Contact = signer
			session.Window.Canvas().SetContent(layoutTransition())
			removeOverlays()
			session.Window.Canvas().SetContent(layoutPM())
		}
	}

	linkMessageOwner := widget.NewHyperlinkWithStyle(
		msgOwner,
		nil,
		fyne.TextAlignLeading,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkMessageOwner.OnTapped = func() {
		if owner != string_ && owner != doubledashes {
			messages.Contact = owner
			session.Window.Canvas().SetContent(layoutTransition())
			removeOverlays()
			session.Window.Canvas().SetContent(layoutPM())
		}
	}

	linkCopySCID := widget.NewHyperlinkWithStyle(
		copySCID,
		nil,
		fyne.TextAlignLeading,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkCopySCID.OnTapped = func() {
		session.Window.Clipboard().SetContent(scid)
	}

	linkView := widget.NewHyperlinkWithStyle(
		viewExplorer,
		nil,
		fyne.TextAlignLeading,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkView.OnTapped = func() {
		if engram.Disk.GetNetwork() {
			link, _ := url.Parse(DEFAULT_EXPLORER_URL + "/tx/" + scid)
			_ = fyne.CurrentApp().OpenURL(link)
		} else {
			link, _ := url.Parse(DEFAULT_TESTNET_EXPLORER_URL + "/tx/" + scid)
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
			fmt.Printf(
				notExported,
				contract.Functions[f].Name,
			)
		} else if contract.Functions[f].Name == stringInitialize ||
			contract.Functions[f].Name == stringInitializePrivate {
			fmt.Printf(
				isInit,
				contract.Functions[f].Name,
			)
		} else {
			data = append(
				data,
				contract.Functions[f].Name,
			)
		}
	}

	data = append(data, singlespace)

	var paramList []fyne.Widget
	var dero_amount uint64
	var asset_amount uint64

	functionList := widget.NewSelect(data, nil)
	functionList.OnChanged = func(s string) {
		if s == singlespace {
			functionList.ClearSelected()
			return
		}

		var params []dvm.Variable

		overlay := session.Window.Canvas().Overlays()

		for f := range contract.Functions {
			if contract.Functions[f].Name == s {
				params = contract.Functions[f].Params

				header := canvas.NewText(
					executeCONTRACTFUNCTION,
					colors.Gray,
				)
				header.TextSize = 14
				header.Alignment = fyne.TextAlignCenter
				header.TextStyle = fyne.TextStyle{Bold: true}

				funcName := canvas.NewText(
					s,
					colors.Account,
				)
				funcName.TextSize = 22
				funcName.Alignment = fyne.TextAlignCenter
				funcName.TextStyle = fyne.TextStyle{Bold: true}

				linkClose := widget.NewHyperlinkWithStyle(
					stringClose,
					nil,
					fyne.TextAlignCenter,
					fyne.TextStyle{
						Bold: true,
					},
				)
				linkClose.OnTapped = func() {
					dero_amount = 0
					asset_amount = 0
					overlay.Top().Hide()
					overlay.Remove(overlay.Top())
					overlay.Remove(overlay.Top())
				}

				span := canvas.NewRectangle(color.Transparent)
				span.SetMinSize(
					fyne.NewSize(
						ui.Width, 10))

				overlay.Add(
					container.NewStack(
						&iframe{},
						canvas.NewRectangle(colors.DarkMatter),
					),
				)

				entryDEROValue := widget.NewEntry()
				entryDEROValue.PlaceHolder = deroAmountNumbers
				entryDEROValue.Validator = func(s string) error {
					dero_amount, err = globals.ParseAmount(s)
					if err != nil {
						entryDEROValue.SetValidationError(err)
						return err
					}

					return nil
				}

				entryAssetValue := widget.NewEntry()
				entryAssetValue.PlaceHolder = assetAmount
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

						if strings.Contains(
							contract.Functions[f].Lines[l][i],
							deroVALUE) &&
							!existsDEROValue {
							paramList = append(
								paramList,
								entryDEROValue,
							)
							paramsContainer.Add(d)
							paramsContainer.Refresh()
						}

						if strings.Contains(
							contract.Functions[f].Lines[l][i],
							assetVALUE) &&
							!existsAssetValue {
							paramList = append(
								paramList,
								entryAssetValue,
							)
							paramsContainer.Add(a)
							paramsContainer.Refresh()
							break
						}
					}
				}

				btnExecute := widget.NewButton(stringExecute, nil)

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
						entry.PlaceHolder = params[p].Name + numbersOnly
					}
					entry.Validator = func(s string) error {
						for p := range params {
							if params[p].Type == 0x5 {
								if params[p].Name == entry.PlaceHolder {
									fmt.Printf(
										boxString,
										params[p].Name,
										s,
									)
									params[p].ValueString = s
								}
							} else if params[p].Type == 0x4 {
								if params[p].Name+numbersOnly == entry.PlaceHolder {
									amount, err := globals.ParseAmount(s)
									if err != nil {
										fmt.Printf(
											boxErr,
											params[p].Name,
											err,
										)
										entry.SetValidationError(err)
										return err
									} else {
										fmt.Printf(
											boxAmount,
											params[p].Name,
											amount,
										)
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

					btnExecute.Text = executing
					btnExecute.Disable()
					btnExecute.Refresh()

					err = executeContractFunction(
						hash,
						dero_amount,
						asset_amount,
						funcName.Text,
						funcType,
						params,
					)
					if err != nil {
						btnExecute.Text = errorExecuting
						btnExecute.Disable()
						btnExecute.Refresh()
					} else {
						btnExecute.Text = executionSuccess
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

	return layout
}

func layoutTransfers() fyne.CanvasObject {
	session.Domain = appTransfers

	wSpacer := widget.NewLabel(singlespace)
	sendTitle := canvas.NewText(
		transfersBanner,
		colors.Gray,
	)
	sendTitle.TextStyle = fyne.TextStyle{Bold: true}
	sendTitle.TextSize = 16

	sendDesc := canvas.NewText(
		string_,
		colors.Gray,
	)
	sendDesc.TextSize = 18
	sendDesc.Alignment = fyne.TextAlignCenter
	sendDesc.TextStyle = fyne.TextStyle{Bold: true}

	sendHeading := canvas.NewText(
		savedTransfers,
		colors.Green,
	)
	sendHeading.TextSize = 22
	sendHeading.Alignment = fyne.TextAlignCenter
	sendHeading.TextStyle = fyne.TextStyle{Bold: true}

	rectStatus := canvas.NewRectangle(color.Transparent)
	rectStatus.SetMinSize(
		fyne.NewSize(
			10,
			10,
		),
	)
	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(
		fyne.NewSize(
			ui.Width,
			20,
		),
	)
	frame := &iframe{}
	rect.SetMinSize(
		fyne.NewSize(
			ui.Width,
			30,
		),
	)
	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(
		fyne.NewSize(
			10,
			5,
		),
	)
	rect.SetMinSize(
		fyne.NewSize(
			10,
			10,
		),
	)
	rectEmpty := canvas.NewRectangle(color.Transparent)
	rectEmpty.SetMinSize(
		fyne.NewSize(
			10,
			10,
		),
	)
	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(
		fyne.NewSize(
			ui.Width,
			35,
		),
	)
	rectListBox := canvas.NewRectangle(color.Transparent)
	rectListBox.SetMinSize(
		fyne.NewSize(
			ui.Width,
			ui.Height*0.43,
		),
	)

	menuLabel := canvas.NewText(
		moreOptionsBanner,
		colors.Gray,
	)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2,
			2,
		),
	)

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2, 2,
		),
	)

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkBack := widget.NewHyperlinkWithStyle(
		backtoDash,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkBack.OnTapped = func() {
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutDashboard())
		removeOverlays()
	}

	var pendingList []string

	for i := 0; i < len(tx.Pending); i++ {
		pendingList = append(
			pendingList,
			strconv.Itoa(i)+
				singlecoma+
				globals.FormatMoney(tx.Pending[i].Amount)+
				singlecoma+
				tx.Pending[i].Destination,
		)
	}

	data := binding.BindStringList(&pendingList)

	scrollBox := widget.NewListWithData(data,
		func() fyne.CanvasObject {
			c := container.NewStack(
				rectList,
				container.NewHBox(
					canvas.NewText(
						string_,
						colors.Account,
					),
					layout.NewSpacer(),
					canvas.NewText(
						string_,
						colors.Account,
					),
				),
			)
			return c
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			dat := di.(binding.String)
			str, err := dat.Get()
			if err != nil {
				fmt.Printf(
					errGeneral,
					err,
				)
			}
			dataItem := strings.SplitN(str, singlecoma, 3)
			dest := dataItem[2]
			dest = threespaces + dest[0:4] + centeredthreeperiods + dest[len(dataItem[2])-10:]
			co.(*fyne.Container).
				Objects[1].(*fyne.Container).
				Objects[0].(*canvas.Text).
				Text = dest
			co.(*fyne.Container).
				Objects[1].(*fyne.Container).
				Objects[0].(*canvas.Text).
				TextSize = 17
			co.(*fyne.Container).
				Objects[1].(*fyne.Container).
				Objects[0].(*canvas.Text).
				TextStyle.Bold = true
			co.(*fyne.Container).
				Objects[1].(*fyne.Container).
				Objects[2].(*canvas.Text).
				Text = dataItem[1] + threespaces
			co.(*fyne.Container).
				Objects[1].(*fyne.Container).
				Objects[2].(*canvas.Text).
				TextSize = 17
			co.(*fyne.Container).
				Objects[1].(*fyne.Container).
				Objects[2].(*canvas.Text).
				TextStyle.Bold = true
		})

	scrollBox.OnSelected = func(id widget.ListItemID) {
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutTransfersDetail(id))
	}

	btnSend := widget.NewButton(sendTransfers, nil)

	btnClear := widget.NewButton(stringClear, func() {
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
		btnSend.Text = disabledOfflineMode
		btnSend.Disable()
	}

	btnSend.OnTapped = func() {
		overlay := session.Window.Canvas().Overlays()

		header := canvas.NewText(
			accountVerification,
			colors.Gray,
		)
		header.TextSize = 14
		header.Alignment = fyne.TextAlignCenter
		header.TextStyle = fyne.TextStyle{Bold: true}

		subHeader := canvas.NewText(
			confirmPassword,
			colors.Account,
		)
		subHeader.TextSize = 22
		subHeader.Alignment = fyne.TextAlignCenter
		subHeader.TextStyle = fyne.TextStyle{Bold: true}

		linkClose := widget.NewHyperlinkWithStyle(
			stringCancel,
			nil,
			fyne.TextAlignCenter,
			fyne.TextStyle{
				Bold: true,
			},
		)
		linkClose.OnTapped = func() {
			overlay := session.Window.Canvas().Overlays()
			overlay.Top().Hide()
			overlay.Remove(overlay.Top())
			overlay.Remove(overlay.Top())
		}

		btnSubmit := widget.NewButton(stringSubmit, nil)

		entryPassword := NewReturnEntry()
		entryPassword.Password = true
		entryPassword.PlaceHolder = stringPassword
		entryPassword.OnChanged = func(s string) {
			if s == string_ {
				btnSubmit.Text = stringSubmit
				btnSubmit.Disable()
				btnSubmit.Refresh()
			} else {
				btnSubmit.Text = stringSubmit
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
					btnSend.Text = settinguptransfer
					btnSend.Disable()
					btnSend.Refresh()
					txid, err := sendAllTransfers()
					if err != nil {
						btnSend.Text = sendTransfers
						btnSend.Enable()
						btnSend.Refresh()
						return
					}

					go func() {
						btnClear.Disable()
						btnSend.Text = txConfirming
						btnSend.Refresh()

						walletapi.WaitNewHeightBlock()
						sHeight := walletapi.Get_Daemon_Height()

						for session.Domain == appTransfers {
							result := engram.Disk.Get_Payments_TXID(txid.String())

							if result.TXID == txid.String() {
								btnSend.Text = transferSuccessful
								btnSend.Refresh()

								break
							}

							// If we go DEFAULT_CONFIRMATION_TIMEOUT blocks without exiting 'Confirming...' loop, display failed to transfer and break
							if walletapi.Get_Daemon_Height() > sHeight+int64(DEFAULT_CONFIRMATION_TIMEOUT) {
								btnSend.Text = txFailed
								btnSend.Disable()
								btnSend.Refresh()
								break
							}

							// If daemon height has incremented, print retry counters into button space
							if walletapi.Get_Daemon_Height()-sHeight > 0 {
								btnSend.Text = fmt.Sprintf(
									txConfirmingSomeofSome,
									walletapi.Get_Daemon_Height()-sHeight,
									DEFAULT_CONFIRMATION_TIMEOUT,
								)
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
				btnSubmit.Text = invalidPassword
				btnSubmit.Disable()
				btnSubmit.Refresh()
			}
		}

		btnSubmit.Disable()

		entryPassword.OnReturn = btnSubmit.OnTapped

		span := canvas.NewRectangle(color.Transparent)
		span.SetMinSize(
			fyne.NewSize(
				ui.Width, 10))

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
						widget.NewLabel(string_),
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

	session.Window.Canvas().SetOnTypedKey(
		func(k *fyne.KeyEvent) {
			if session.Domain != appTransfers {
				return
			}

			if k.Name == fyne.KeyDown {
				session.Dashboard = stringmain
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
	wSpacer := widget.NewLabel(singlespace)

	rectWidth := canvas.NewRectangle(color.Transparent)
	rectWidth.SetMinSize(
		fyne.NewSize(
			ui.MaxWidth*0.99, 10))

	rectWidth90 := canvas.NewRectangle(color.Transparent)
	rectWidth90.SetMinSize(
		fyne.NewSize(
			ui.Width, 10))

	frame := &iframe{}

	heading := canvas.NewText(
		transferdetailBanner,
		colors.Gray,
	)
	heading.TextSize = 16
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(
		fyne.NewSize(6, 5))

	labelDestination := canvas.NewText(
		receiverAddress,
		colors.Gray,
	)
	labelDestination.TextSize = 14
	labelDestination.Alignment = fyne.TextAlignLeading
	labelDestination.TextStyle = fyne.TextStyle{Bold: true}

	labelAmount := canvas.NewText(
		amount,
		colors.Gray,
	)
	labelAmount.TextSize = 14
	labelAmount.Alignment = fyne.TextAlignLeading
	labelAmount.TextStyle = fyne.TextStyle{Bold: true}

	labelService := canvas.NewText(
		serviceAddress,
		colors.Gray,
	)
	labelService.TextSize = 14
	labelService.Alignment = fyne.TextAlignLeading
	labelService.TextStyle = fyne.TextStyle{Bold: true}

	labelDestPort := canvas.NewText(
		destinationPort,
		colors.Gray,
	)
	labelDestPort.TextSize = 14
	labelDestPort.TextStyle = fyne.TextStyle{Bold: true}

	labelSourcePort := canvas.NewText(
		sourcePort,
		colors.Gray,
	)
	labelSourcePort.TextSize = 14
	labelSourcePort.TextStyle = fyne.TextStyle{Bold: true}

	labelFees := canvas.NewText(
		txFees,
		colors.Gray,
	)
	labelFees.TextSize = 14
	labelFees.TextStyle = fyne.TextStyle{Bold: true}

	labelPayload := canvas.NewText(
		payLoad,
		colors.Gray,
	)
	labelPayload.TextSize = 14
	labelPayload.TextStyle = fyne.TextStyle{Bold: true}

	labelReply := canvas.NewText(
		replyAddress,
		colors.Gray,
	)
	labelReply.TextSize = 14
	labelReply.TextStyle = fyne.TextStyle{Bold: true}

	labelSeparator := widget.NewRichTextFromMarkdown(string_)
	labelSeparator.Wrapping = fyne.TextWrapOff
	labelSeparator.ParseMarkdown(threedashes)

	labelSeparator2 := widget.NewRichTextFromMarkdown(string_)
	labelSeparator2.Wrapping = fyne.TextWrapOff
	labelSeparator2.ParseMarkdown(threedashes)

	labelSeparator3 := widget.NewRichTextFromMarkdown(string_)
	labelSeparator3.Wrapping = fyne.TextWrapOff
	labelSeparator3.ParseMarkdown(threedashes)

	labelSeparator4 := widget.NewRichTextFromMarkdown(string_)
	labelSeparator4.Wrapping = fyne.TextWrapOff
	labelSeparator4.ParseMarkdown(threedashes)

	labelSeparator5 := widget.NewRichTextFromMarkdown(string_)
	labelSeparator5.Wrapping = fyne.TextWrapOff
	labelSeparator5.ParseMarkdown(threedashes)

	menuLabel := canvas.NewText(
		moreOptionsBanner,
		colors.Gray,
	)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2, 2))

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2, 2))

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	details := tx.Pending[index]

	valueDestination := widget.NewRichTextFromMarkdown(doubledashes)
	valueDestination.Wrapping = fyne.TextWrapBreak

	valueType := widget.NewRichTextFromMarkdown(doubledashes)
	valueType.Wrapping = fyne.TextWrapOff

	if details.Destination != string_ {
		address, _ := globals.ParseValidateAddress(details.Destination)
		if address.IsIntegratedAddress() {
			valueDestination.ParseMarkdown(address.BaseAddress().String())
			valueType.ParseMarkdown(header3Service)
		} else {
			valueDestination.ParseMarkdown(details.Destination)
			valueType.ParseMarkdown(header3Normal)
		}
	}

	valueReply := widget.NewRichTextFromMarkdown(doubledashes)
	valueReply.Wrapping = fyne.TextWrapBreak

	if details.Payload_RPC.HasValue(
		rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
		rpc.DataString,
	) {
		if details.Payload_RPC.Value(
			rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
			rpc.DataString,
		).(string) != string_ {
			valueReply.ParseMarkdown(
				string_ + details.Payload_RPC.Value(
					rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
					rpc.DataString,
				).(string),
			)
		}
	}

	valuePayload := widget.NewRichTextFromMarkdown(doubledashes)
	valuePayload.Wrapping = fyne.TextWrapBreak

	if details.Payload_RPC.HasValue(
		rpc.RPC_COMMENT,
		rpc.DataString,
	) {
		if details.Payload_RPC.Value(
			rpc.RPC_COMMENT,
			rpc.DataString,
		).(string) != string_ {
			valuePayload.ParseMarkdown(
				string_ + details.Payload_RPC.Value(
					rpc.RPC_COMMENT,
					rpc.DataString,
				).(string),
			)
		}
	}

	valueAmount := canvas.NewText(
		string_,
		colors.Account,
	)
	valueAmount.TextSize = 22
	valueAmount.TextStyle = fyne.TextStyle{Bold: true}
	valueAmount.Text = doublespace + globals.FormatMoney(details.Amount)

	valueDestPort := canvas.NewText(
		string_,
		colors.Account,
	)
	valueDestPort.TextSize = 22
	valueDestPort.TextStyle = fyne.TextStyle{Bold: true}

	if details.Payload_RPC.HasValue(
		rpc.RPC_DESTINATION_PORT,
		rpc.DataUint64,
	) {
		port := fmt.Sprintf(
			"%d",
			details.Payload_RPC.Value(
				rpc.RPC_DESTINATION_PORT,
				rpc.DataUint64,
			),
		)
		valueDestPort.Text = doublespace + port
	} else {
		valueDestPort.Text = zero
	}

	linkBack := widget.NewHyperlinkWithStyle(
		backtoTransfers,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkBack.OnTapped = func() {
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutTransfers())
	}

	btnDelete := widget.NewButton(cancelTransfer, nil)
	btnDelete.OnTapped = func() {
		if len(tx.Pending) > index+1 {
			tx.Pending = append(
				tx.Pending[:index],
				tx.Pending[index+1:]...,
			)
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

	return layout
}

func layoutTransition() fyne.CanvasObject {
	frame := &iframe{}
	resizeWindow(
		ui.MaxWidth,
		ui.MaxHeight)

	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(
		fyne.NewSize(
			ui.Width*0.45,
			ui.Width*0.45,
		),
	)

	if res.loading == nil {
		res.loading, _ = x.NewAnimatedGifFromResource(resourceLoadingGif)
		res.loading.SetMinSize(
			fyne.NewSize(
				ui.Width*0.45,
				ui.Width*0.45,
			),
		)
		res.loading.Resize(
			fyne.NewSize(
				ui.Width*0.45,
				ui.Width*0.45,
			),
		)
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
	stopGnomon()
	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(
		fyne.NewSize(
			ui.Width,
			10,
		),
	)
	rectScroll := canvas.NewRectangle(color.Transparent)
	rectScroll.SetMinSize(
		fyne.NewSize(
			ui.Width,
			ui.Height*0.65,
		),
	)
	frame := &iframe{}
	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(
		fyne.NewSize(
			10,
			5,
		),
	)

	heading := canvas.NewText(
		mySettings,
		colors.Green,
	)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	labelNetwork := canvas.NewText(
		stringNETWORK,
		colors.Gray,
	)
	labelNetwork.TextStyle = fyne.TextStyle{Bold: true}
	labelNetwork.TextSize = 14

	labelNode := canvas.NewText(
		stringCONNECTION,
		colors.Gray,
	)
	labelNode.TextStyle = fyne.TextStyle{Bold: true}
	labelNode.TextSize = 14

	labelSecurity := canvas.NewText(
		stringSECURITY,
		colors.Gray,
	)
	labelSecurity.TextStyle = fyne.TextStyle{Bold: true}
	labelSecurity.TextSize = 14

	labelGnomon := canvas.NewText(
		stringGNOMON,
		colors.Gray,
	)
	labelGnomon.TextStyle = fyne.TextStyle{Bold: true}
	labelGnomon.TextSize = 14

	textGnomon := widget.NewRichTextWithText(gnomongMsg)
	textGnomon.Wrapping = fyne.TextWrapWord

	textCyberdeck := widget.NewRichTextWithText(usrpassMsg)
	textCyberdeck.Wrapping = fyne.TextWrapWord

	btnRestore := widget.NewButton(
		restoreDefaults,
		nil,
	)
	btnDelete := widget.NewButton(clearData, nil)

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
			err = errors.New(invalidHost)
			entryAddress.SetValidationError(err)
		}

		return
	}
	entryAddress.PlaceHolder = DEFAULT_DISCOVER_DAEMON
	entryAddress.SetText(getDaemon())
	entryAddress.Refresh()

	selectNodes := widget.NewSelect(nil, nil)
	selectNodes.PlaceHolder = selectNode
	if session.Testnet {
		selectNodes.Options = []string{
			DEFAULT_REMOTE_TESTNET_DAEMON,
			DEFAULT_LOCAL_TESTNET_DAEMON,
		}
	} else {
		selectNodes.Options = []string{
			DEFAULT_REMOTE_DAEMON,
			DEFAULT_LOCAL_DAEMON,
		}
	}
	selectNodes.OnChanged = func(s string) {
		if s != string_ {
			err := setDaemon(s)
			if err == nil {
				entryAddress.Text = s
				entryAddress.Refresh()
			}
			selectNodes.ClearSelected()
		}
	}

	labelScan := widget.NewRichTextFromMarkdown(scanBlocks)
	labelScan.Wrapping = fyne.TextWrapWord

	entryScan := widget.NewEntry()
	entryScan.PlaceHolder = numberOfBlocks
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

	radioNetwork := widget.NewRadioGroup(
		[]string{
			stringmainnet,
			stringtestnet,
		},
		nil,
	)
	radioNetwork.Horizontal = false
	radioNetwork.OnChanged = func(s string) {
		if s == stringtestnet {
			setTestnet(true)
			selectNodes.Options = []string{
				DEFAULT_REMOTE_TESTNET_DAEMON,
				DEFAULT_LOCAL_TESTNET_DAEMON,
			}
		} else {
			setTestnet(false)
			selectNodes.Options = []string{
				DEFAULT_REMOTE_DAEMON,
				DEFAULT_LOCAL_DAEMON,
			}
		}

		selectNodes.Refresh()
	}

	net, _ := GetValue(
		stringsettings,
		[]byte(stringNETWORK),
	)

	if string(net) == stringtestnet {
		radioNetwork.SetSelected(stringtestnet)
	} else {
		radioNetwork.SetSelected(stringmainnet)
	}

	radioNetwork.Refresh()

	entryUser := widget.NewEntry()
	entryUser.PlaceHolder = stringUsername
	entryUser.SetText(cyberdeck.user)

	entryPass := widget.NewEntry()
	entryPass.PlaceHolder = stringPassword
	entryPass.Password = true
	entryPass.SetText(cyberdeck.pass)

	entryUser.OnChanged = func(s string) {
		cyberdeck.user = s
	}

	entryPass.OnChanged = func(s string) {
		cyberdeck.pass = s
	}

	checkGnomon := widget.NewCheck(enableGnomon, nil)
	checkGnomon.OnChanged = func(b bool) {
		if b {
			StoreValue(
				stringsettings,
				bytegnomon,
				byte1,
			)
			checkGnomon.Checked = true
			gnomon.Active = 1
		} else {
			StoreValue(
				stringsettings,
				bytegnomon,
				byte0,
			)
			checkGnomon.Checked = false
			gnomon.Active = 0
		}
	}

	gmn, err := GetValue(stringsettings, bytegnomon)
	if err != nil {
		gnomon.Active = 1
		StoreValue(
			stringsettings,
			bytegnomon,
			byte1,
		)
		checkGnomon.Checked = true
	}

	if string(gmn) == string1 {
		checkGnomon.Checked = true
	} else {
		checkGnomon.Checked = false
	}

	labelBack := widget.NewHyperlinkWithStyle(
		returntoLogin,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
	labelBack.OnTapped = func() {
		if radioNetwork.Selected == stringtestnet {
			setTestnet(true)
		} else {
			setTestnet(false)
		}
		setDaemon(entryAddress.Text)

		initSettings()

		resizeWindow(
			ui.MaxWidth,
			ui.MaxHeight)
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutMain())
		removeOverlays()
	}

	btnRestore.OnTapped = func() {
		setTestnet(false)
		setDaemon(DEFAULT_REMOTE_DAEMON)
		setAuthMode(stringtrue)
		setGnomon(string1)

		resizeWindow(
			ui.MaxWidth,
			ui.MaxHeight)
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutSettings())
		removeOverlays()
	}

	statusText := canvas.NewText(
		string_,
		colors.Account,
	)
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
		statusText.Text = gnomonDeleted
		statusText.Refresh()
	}

	formSettings := container.NewVBox(
		labelNetwork,
		rectSpacer,
		radioNetwork,
		widget.NewLabel(string_),
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
		widget.NewLabel(string_),
		labelSecurity,
		rectSpacer,
		textCyberdeck,
		rectSpacer,
		entryUser,
		rectSpacer,
		entryPass,
		rectSpacer,
		widget.NewLabel(string_),
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

	scrollBox.SetMinSize(
		fyne.NewSize(
			ui.MaxWidth, ui.Height*0.68))

	gridItem1 := container.NewCenter(
		container.NewVBox(
			widget.NewLabel(string_),
			heading,
			widget.NewLabel(string_),
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
		widget.NewLabel(singlespace),
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
	session.Domain = appMessages

	if !walletapi.Connected {
		session.Window.SetContent(layoutSettings())
	}

	title := canvas.NewText(
		messagesBanner,
		colors.Gray,
	)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 16

	heading := canvas.NewText(
		myContacts,
		colors.Green,
	)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	menuLabel := canvas.NewText(
		moreOptionsBanner,
		colors.Gray,
	)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	checkLimit := widget.NewCheck(showOnlyRecent, nil)
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
	sep.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2,
			2,
		),
	)

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2,
			2,
		),
	)

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkBack := widget.NewHyperlinkWithStyle(
		backtoDash,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkBack.OnTapped = func() {
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutDashboard())
		removeOverlays()
	}

	rectStatus := canvas.NewRectangle(color.Transparent)
	rectStatus.SetMinSize(
		fyne.NewSize(
			10,
			10,
		),
	)
	rectEmpty := canvas.NewRectangle(color.Transparent)
	rectEmpty.SetMinSize(
		fyne.NewSize(
			10,
			10,
		),
	)
	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(
		fyne.NewSize(
			ui.Width,
			20,
		),
	)
	frame := &iframe{}
	rect.SetMinSize(
		fyne.NewSize(
			ui.Width,
			30,
		),
	)
	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(
		fyne.NewSize(
			10,
			5,
		),
	)
	rect.SetMinSize(
		fyne.NewSize(
			10,
			10,
		),
	)
	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(
		fyne.NewSize(
			ui.Width,
			35,
		),
	)
	rectListBox := canvas.NewRectangle(color.Transparent)
	rectListBox.SetMinSize(
		fyne.NewSize(
			ui.Width,
			ui.Height*0.40,
		),
	)

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

	messages.Box = widget.NewListWithData(
		list,
		func() fyne.CanvasObject {
			c := container.NewVBox(
				widget.NewLabel(string_),
			)
			return c
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			dat := di.(binding.String)
			str, err := dat.Get()
			if err != nil {
				return
			}
			dataItem := strings.Split(str, threetilde)
			short := dataItem[0]
			address := short[len(short)-10:]
			username := dataItem[1]

			if username == string_ {
				co.(*fyne.Container).
					Objects[0].(*widget.Label).
					SetText(threeperiods + address)
			} else {
				co.(*fyne.Container).
					Objects[0].(*widget.Label).
					SetText(username)
			}
			co.(*fyne.Container).
				Objects[0].(*widget.Label).
				Wrapping = fyne.TextWrapWord
			co.(*fyne.Container).
				Objects[0].(*widget.Label).
				TextStyle.Bold = false
			co.(*fyne.Container).
				Objects[0].(*widget.Label).
				Alignment = fyne.TextAlignLeading
		})

	messages.Box.OnSelected = func(id widget.ListItemID) {
		messages.Box.UnselectAll()
		split := strings.Split(data[id], threetilde)
		if split[1] == string_ {
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
	entrySearch.PlaceHolder = searchContact
	entrySearch.OnChanged = func(s string) {
		s = strings.ToLower(s)
		searchList = []string{}
		if s == string_ {
			data = temp
			list.Reload()
		} else {
			for _, d := range temp {
				tempd := strings.ToLower(d)
				split := strings.Split(
					tempd,
					threetilde,
				)

				if split[1] == string_ {
					if strings.Contains(
						split[0],
						s,
					) {
						searchList = append(
							searchList,
							d,
						)
					}
				} else {
					if strings.Contains(
						split[1],
						s,
					) {
						searchList = append(
							searchList,
							d,
						)
					}
				}
			}

			data = searchList
			list.Reload()
		}
	}

	btnSend := widget.NewButton(newMessage, func() {
		_, err := globals.ParseValidateAddress(messages.Contact)
		if err != nil {
			_, err := engram.Disk.NameToAddress(messages.Contact)
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
	entryDest.PlaceHolder = usernameAddress
	entryDest.Validator = func(s string) error {
		if len(s) > 0 {
			_, err := globals.ParseValidateAddress(s)
			if err != nil {
				btnSend.Disable()
				_, err := engram.Disk.NameToAddress(s)
				if err != nil {
					btnSend.Disable()
					return errors.New(invalidContact)
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

		return errors.New(invalidContact)
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

	session.Window.Canvas().SetOnTypedKey(
		func(k *fyne.KeyEvent) {
			if session.Domain != appMessages {
				return
			}

			if k.Name == fyne.KeyUp {
				session.Dashboard = stringmain

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
	session.Domain = appMessagesContact

	if !walletapi.Connected {
		session.Window.SetContent(layoutSettings())
	}

	getPrimaryUsername()

	contactAddress := string_

	_, err := globals.ParseValidateAddress(messages.Contact)
	if err != nil {
		_, err := engram.Disk.NameToAddress(messages.Contact)
		if err == nil {
			contactAddress = messages.Contact
		}
	} else {
		short := messages.Contact[len(messages.Contact)-10:]
		contactAddress = threeperiods + short
	}

	title := canvas.NewText(
		messagesBanner,
		colors.Gray,
	)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 16

	heading := canvas.NewText(
		contactAddress,
		colors.Green,
	)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	lastActive := canvas.NewText(
		string_,
		colors.Gray,
	)
	lastActive.TextSize = 12
	lastActive.Alignment = fyne.TextAlignCenter
	lastActive.TextStyle = fyne.TextStyle{Bold: false}

	menuLabel := canvas.NewText(
		moreOptionsBanner,
		colors.Gray,
	)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2,
			2,
		),
	)

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2,
			2,
		),
	)

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkBack := widget.NewHyperlinkWithStyle(
		backtoMessages,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkBack.OnTapped = func() {
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutMessages())
		removeOverlays()
	}

	rectStatus := canvas.NewRectangle(color.Transparent)
	rectStatus.SetMinSize(
		fyne.NewSize(
			10,
			10,
		),
	)
	rectEmpty := canvas.NewRectangle(color.Transparent)
	rectEmpty.SetMinSize(
		fyne.NewSize(
			10,
			10,
		),
	)
	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(
		fyne.NewSize(
			ui.Width*0.7,
			30,
		),
	)
	frame := &iframe{}
	subframe := canvas.NewRectangle(color.Transparent)
	subframe.SetMinSize(
		fyne.NewSize(
			ui.Width,
			ui.Height*0.50,
		),
	)
	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(
		fyne.NewSize(
			10,
			5,
		),
	)
	rect.SetMinSize(
		fyne.NewSize(
			10,
			10,
		),
	)
	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(
		fyne.NewSize(
			ui.Width,
			35,
		),
	)
	rectListBox := canvas.NewRectangle(color.Transparent)
	rectListBox.SetMinSize(
		fyne.NewSize(
			ui.Width*0.42,
			30,
		),
	)
	rectOutbound := canvas.NewRectangle(color.Transparent)
	rectOutbound.SetMinSize(
		fyne.NewSize(
			ui.Width*0.166,
			30,
		),
	)

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
			if data[d].Payload_RPC.Has(
				rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
				rpc.DataString,
			) {
				if data[d].Payload_RPC.Value(
					rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
					rpc.DataString,
				).(string) == string_ {

				} else {
					t := data[d].Time
					time := string(t.Format(time.RFC822))
					comment := data[d].Payload_RPC.Value(
						rpc.RPC_COMMENT,
						rpc.DataString,
					).(string)
					links := getTextURL(comment)

					for i := range links {
						if comment == links[i] {
							if len(links[i]) > 25 {
								comment = `[ ` + links[i][0:25] + threeperiods + ` ](` + links[i] + `)`
							} else {
								comment = `[ ` + links[i] + ` ](` + links[i] + `)`
							}
						} else {
							linkText := string_
							split := strings.Split(
								comment,
								links[i],
							)
							if len(links[i]) > 25 {
								linkText = links[i][0:25] + threeperiods
							} else {
								linkText = links[i]
							}
							comment = `` + split[0] + `[link]` + split[1] +
								"\n\n›" + `[ ` + linkText + ` ](` + links[i] + `)`
						}
					}
					messages.Data = append(
						messages.Data,
						data[d].Payload_RPC.Value(
							rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
							rpc.DataString,
						).(string)+
							foursemicolons+
							comment+
							foursemicolons+
							time,
					)
				}
			}
		} else {
			t := data[d].Time
			time := string(t.Format(time.RFC822))
			comment := data[d].Payload_RPC.Value(
				rpc.RPC_COMMENT,
				rpc.DataString,
			).(string)
			links := getTextURL(comment)

			for i := range links {
				if comment == links[i] {
					if len(links[i]) > 25 {
						comment = `[ ` + links[i][0:25] + threeperiods + ` ](` + links[i] + `)`
					} else {
						comment = `[ ` + links[i] + ` ](` + links[i] + `)`
					}
				} else {
					linkText := string_
					split := strings.Split(
						comment,
						links[i],
					)
					if len(links[i]) > 25 {
						linkText = links[i][0:25] + threeperiods
					} else {
						linkText = links[i]
					}
					comment = `` + split[0] + `[link]` + split[1] +
						"\n\n›" + `[ ` + linkText + ` ](` + links[i] + `)`
				}
			}
			messages.Data = append(
				messages.Data,
				engram.Disk.GetAddress().String()+
					foursemicolons+
					comment+
					foursemicolons+
					time,
			)
		}
	}

	if len(data) > 0 {
		for m := range messages.Data {
			var sender string
			split := strings.Split(
				messages.Data[m],
				foursemicolons,
			)
			mdata := widget.NewRichTextFromMarkdown(string_)
			mdata.Wrapping = fyne.TextWrapWord
			datetime := canvas.NewText(
				string_,
				colors.Green,
			)
			datetime.TextSize = 11
			boxColor := colors.Flint
			rect := canvas.NewRectangle(boxColor)
			rect.SetMinSize(
				fyne.NewSize(
					ui.Width*0.80, 30))
			rect.CornerRadius = 5.0
			rect5 := canvas.NewRectangle(color.Transparent)
			rect5.SetMinSize(
				fyne.NewSize(5, 5))

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

			lastActive.Text = lastUpdated + time.Now().Format(time.RFC822)
			lastActive.Refresh()

			chats.Add(e)
			chats.Refresh()
			chatbox.Refresh()
			chatbox.ScrollToBottom()
		}
	}

	btnSend := widget.NewButton(stringSend, nil)
	btnSend.Disable()

	entry := widget.NewEntry()
	entry.MultiLine = false
	entry.Wrapping = fyne.TextWrap(fyne.TextTruncateClip)
	entry.PlaceHolder = stringMessage
	entry.OnChanged = func(s string) {
		messages.Message = s
		contact := messages.Contact
		check, err := engram.Disk.NameToAddress(messages.Contact)
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

		err = checkMessagePack(
			messages.Message,
			session.Username,
			contact,
		)
		if err != nil {
			btnSend.Text = messageTooLong
			btnSend.Disable()
			btnSend.Refresh()
			return
		} else {
			if messages.Message == string_ {
				btnSend.Text = stringSend
				btnSend.Disable()
				btnSend.Refresh()
			} else {
				btnSend.Text = stringSend
				btnSend.Enable()
				btnSend.Refresh()
			}
		}
	}

	btnSend.OnTapped = func() {
		if messages.Message == string_ {
			return
		}
		contact := string_
		_, err := globals.ParseValidateAddress(messages.Contact)
		if err != nil {
			check, err := engram.Disk.NameToAddress(messages.Contact)
			if err != nil {
				fmt.Printf(
					failedToSend,
					err,
				)
				btnSend.Text = failedtoVerify
				btnSend.Disable()
				btnSend.Refresh()
				return
			}
			contact = check
		} else {
			contact = messages.Contact
		}

		txid, err := sendMessage(
			messages.Message,
			session.Username,
			contact,
		)
		if err != nil {
			fmt.Printf(
				failedToSend,
				err,
			)
			btnSend.Text = failttoSend
			btnSend.Disable()
			btnSend.Refresh()
			return
		}

		fmt.Printf(
			successDispatch,
			messages.Contact,
		)
		btnSend.Text = txConfirming
		btnSend.Disable()
		btnSend.Refresh()
		messages.Message = string_
		entry.Text = string_
		entry.Refresh()

		go func() {
			walletapi.WaitNewHeightBlock()
			sHeight := walletapi.Get_Daemon_Height()
			var success bool
			for session.Domain == appMessagesContact {
				result := engram.Disk.Get_Payments_TXID(txid.String())

				if result.TXID != txid.String() {
					time.Sleep(time.Second * 1)
				} else {
					success = true
				}

				// If we go DEFAULT_CONFIRMATION_TIMEOUT blocks without exiting 'Confirming...' loop, display failed to transfer and break
				if walletapi.Get_Daemon_Height() > sHeight+int64(DEFAULT_CONFIRMATION_TIMEOUT) {
					btnSend.Text = failttoSend
					btnSend.Disable()
					btnSend.Refresh()
					break
				}

				// If daemon height has incremented, print retry counters into button space
				if walletapi.Get_Daemon_Height()-sHeight > 0 {
					btnSend.Text = fmt.Sprintf(
						txConfirmingSomeofSome,
						walletapi.Get_Daemon_Height()-sHeight,
						DEFAULT_CONFIRMATION_TIMEOUT,
					)
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
		if session.Domain != appMessagesContact {
			return
		}

		if k.Name == fyne.KeyUp {
			session.Dashboard = appMessages
			messages.Contact = string_
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutMessages())
			removeOverlays()
		} else if k.Name == fyne.KeyEscape {
			session.Dashboard = appMessages
			messages.Contact = string_
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
	session.Domain = appCyberdeck
	wSpacer := widget.NewLabel(singlespace)
	title := canvas.NewText(
		cyberdeckBanner,
		colors.Gray,
	)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 16

	heading := canvas.NewText(
		myContacts,
		colors.Green,
	)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(
		fyne.NewSize(
			ui.Width,
			20,
		),
	)
	frame := &iframe{}
	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(
		fyne.NewSize(
			10,
			5,
		),
	)

	menuLabel := canvas.NewText(
		moreOptionsBanner,
		colors.Gray,
	)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2,
			2,
		),
	)

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2,
			2,
		),
	)

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkBack := widget.NewHyperlinkWithStyle(
		backtoDash,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkBack.OnTapped = func() {
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutDashboard())
		removeOverlays()
	}

	shardText := canvas.NewText(session.Username, colors.Green)
	shardText.TextStyle = fyne.TextStyle{Bold: true}
	shardText.TextSize = 22

	shortShard := canvas.NewText(appConnections, colors.Gray)
	shortShard.TextStyle = fyne.TextStyle{Bold: true}
	shortShard.TextSize = 12

	linkColor := colors.Green

	if cyberdeck.server == nil {
		session.Link = stringBlocked
		linkColor = colors.Gray
	}

	cyberdeck.status = canvas.NewText(session.Link, linkColor)
	cyberdeck.status.TextSize = 22
	cyberdeck.status.TextStyle = fyne.TextStyle{Bold: true}

	serverStatus := canvas.NewText(appConnections, colors.Gray)
	serverStatus.TextSize = 12
	serverStatus.Alignment = fyne.TextAlignCenter
	serverStatus.TextStyle = fyne.TextStyle{Bold: true}

	linkCenter := container.NewCenter(
		cyberdeck.status,
	)

	cyberdeck.userText = widget.NewEntry()
	cyberdeck.userText.PlaceHolder = stringUsername
	cyberdeck.userText.OnChanged = func(s string) {
		if len(s) > 1 {
			cyberdeck.user = s
		}
	}

	cyberdeck.passText = widget.NewEntry()
	cyberdeck.passText.Password = true
	cyberdeck.passText.PlaceHolder = stringPassword
	cyberdeck.passText.OnChanged = func(s string) {
		if len(s) > 1 {
			cyberdeck.pass = s
		}
	}

	cyberdeck.toggle = widget.NewButton(turnOn, nil)
	cyberdeck.toggle.OnTapped = func() {
		toggleCyberdeck()
	}

	if session.Offline {
		cyberdeck.toggle.Text = disabledinOfflineMode
		cyberdeck.toggle.Disable()
	} else {
		if cyberdeck.server != nil {
			cyberdeck.status.Text = stringAllowed
			cyberdeck.status.Color = colors.Green
			cyberdeck.toggle.Text = turnOff
			cyberdeck.userText.Disable()
			cyberdeck.passText.Disable()
		} else {
			cyberdeck.status.Text = stringBlocked
			cyberdeck.status.Color = colors.Gray
			cyberdeck.toggle.Text = turnOn
			cyberdeck.userText.Enable()
			cyberdeck.passText.Enable()
		}
	}

	cyberdeck.userText.SetText(cyberdeck.user)
	cyberdeck.passText.SetText(cyberdeck.pass)

	linkCopy := widget.NewHyperlinkWithStyle(
		copyCreds,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkCopy.OnTapped = func() {
		session.Window.Clipboard().SetContent(
			cyberdeck.user +
				singlecolon +
				cyberdeck.pass,
		)
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
			session.Dashboard = stringmain
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
	session.Domain = appIdentity
	title := canvas.NewText(identityBanner, colors.Gray)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 16

	heading := canvas.NewText(myContacts, colors.Green)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	menuLabel := canvas.NewText(moreOptionsBanner, colors.Gray)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2,
			2,
		),
	)

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2,
			2,
		),
	)

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	frame := &iframe{}

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(
		fyne.NewSize(
			10,
			5,
		),
	)
	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(
		fyne.NewSize(
			ui.Width, 35,
		),
	)
	rectListBox := canvas.NewRectangle(color.Transparent)
	rectListBox.SetMinSize(
		fyne.NewSize(
			ui.Width,
			ui.Height*0.44,
		),
	)

	shortShard := canvas.NewText(primaryUser, colors.Gray)
	shortShard.TextStyle = fyne.TextStyle{Bold: true}
	shortShard.TextSize = 12

	idCenter := container.NewCenter(
		shortShard,
	)

	linkBack := widget.NewHyperlinkWithStyle(
		backtoDash,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
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

	userData, err := queryUsernames()
	if err != nil {
		userData, err = getUsernames()
		if err != nil {
			userData = nil
		}
	}

	userList := binding.BindStringList(&userData)

	btnReg := widget.NewButton(
		centeredRegister,
		nil,
	)
	btnReg.Disable()
	btnReg.OnTapped = func() {
		if len(session.NewUser) > 5 {
			valid, _, _ := checkUsername(session.NewUser, -1)
			if !valid {
				btnReg.Text = txConfirming
				btnReg.Disable()
				btnReg.Refresh()
				entryReg.Disable()
				err := registerUsername(session.NewUser)
				if err != nil {
					btnReg.Text = unableRegister
					btnReg.Refresh()
					fmt.Printf(
						userMsg,
						err,
					)

				} else {
					go func() {
						entryReg.Text = string_
						entryReg.Refresh()
						walletapi.WaitNewHeightBlock()
						sHeight := walletapi.Get_Daemon_Height()
						var loop bool

						for !loop {
							if session.Domain == appIdentity {
								//vars, _, _, err := gnomon.Index.RPC.GetSCVariables("0000000000000000000000000000000000000000000000000000000000000001", engram.Disk.Get_Daemon_TopoHeight(), nil, []string{session.NewUser}, nil, false)
								usernames, err := queryUsernames()
								if err != nil {
									fmt.Printf(
										errQuerying,
										err,
									)
									return
								}

								for u := range usernames {
									if usernames[u] == session.NewUser {
										fmt.Printf(
											successReg,
											session.NewUser,
										)
										_ = tx
										btnReg.Text = successfulRegister
										btnReg.Refresh()
										session.NewUser = string_
										loop = true
										session.Window.SetContent(layoutIdentity())
										break
									}
								}

								// If we go DEFAULT_CONFIRMATION_TIMEOUT blocks without exiting 'Confirming...' loop, display failed to transfer and break
								if walletapi.Get_Daemon_Height() > sHeight+int64(DEFAULT_CONFIRMATION_TIMEOUT) {
									btnReg.Text = unableRegister
									btnReg.Refresh()
									loop = true
									break
								}

								// If daemon height has incremented, print retry counters into button space
								if walletapi.Get_Daemon_Height()-sHeight > 0 {
									btnReg.Text = fmt.Sprintf(
										txConfirmingSomeofSome,
										walletapi.Get_Daemon_Height()-sHeight,
										DEFAULT_CONFIRMATION_TIMEOUT,
									)
									btnReg.Refresh()
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

	entryReg.PlaceHolder = newUser
	entryReg.Validator = func(s string) error {
		btnReg.Text = centeredRegister
		btnReg.Enable()
		btnReg.Refresh()
		session.NewUser = s
		// Name Service SCID Logic
		//	15  IF STRLEN(name) >= 64 THEN GOTO 50 // skip names misuse
		//	20  IF STRLEN(name) >= 6 THEN GOTO 40
		if len(s) > 5 && len(s) < 64 {
			valid, _, _ := checkUsername(s, -1)
			if !valid {
				btnReg.Enable()
				btnReg.Refresh()
			} else {
				btnReg.Disable()
				err := errors.New(userExists)
				entryReg.SetValidationError(err)
				btnReg.Refresh()
				return err
			}
		} else {
			btnReg.Disable()
			err := errors.New(usertooShort)
			entryReg.SetValidationError(err)
			btnReg.Refresh()
			return err
		}

		return nil
	}

	userBox := widget.NewListWithData(userList,
		func() fyne.CanvasObject {
			c := container.NewVBox(
				widget.NewLabel(string_),
			)
			return c
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			dat := di.(binding.String)
			str, err := dat.Get()
			if err != nil {
				return
			}

			co.(*fyne.Container).
				Objects[0].(*widget.Label).
				SetText(str)
			co.(*fyne.Container).
				Objects[0].(*widget.Label).
				Wrapping = fyne.TextWrapWord
			co.(*fyne.Container).
				Objects[0].(*widget.Label).
				TextStyle.Bold = false
			co.(*fyne.Container).
				Objects[0].(*widget.Label).
				Alignment = fyne.TextAlignLeading
		})

	err = getPrimaryUsername()
	if err != nil {
		session.Username = string_
	}

	textUsername := canvas.NewText(
		session.Username,
		colors.Green,
	)
	textUsername.TextStyle = fyne.TextStyle{Bold: true}
	textUsername.TextSize = 22

	if session.Username == string_ {
		textUsername.Text = threedashes
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

	session.Window.Canvas().SetOnTypedKey(
		func(k *fyne.KeyEvent) {
			if k.Name == fyne.KeyRight {
				session.Dashboard = stringmain

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

	wSpacer := widget.NewLabel(singlespace)

	rectWidth := canvas.NewRectangle(color.Transparent)
	rectWidth.SetMinSize(
		fyne.NewSize(
			ui.MaxWidth*0.99,
			10,
		),
	)

	rectWidth90 := canvas.NewRectangle(color.Transparent)
	rectWidth90.SetMinSize(
		fyne.NewSize(
			ui.Width,
			10,
		),
	)

	frame := &iframe{}

	heading := canvas.NewText(
		identitydetailBanner,
		colors.Gray,
	)
	heading.TextSize = 16
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(
		fyne.NewSize(
			6,
			5,
		),
	)

	labelUsername := canvas.NewText(
		registerUser,
		colors.Gray,
	)
	labelUsername.TextSize = 11
	labelUsername.Alignment = fyne.TextAlignCenter
	labelUsername.TextStyle = fyne.TextStyle{Bold: true}

	labelTransfer := canvas.NewText(
		transferBanner,
		colors.Gray,
	)
	labelTransfer.TextSize = 11
	labelTransfer.Alignment = fyne.TextAlignCenter
	labelTransfer.TextStyle = fyne.TextStyle{Bold: true}

	menuLabel := canvas.NewText(
		moreOptionsBanner,
		colors.Gray,
	)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2,
			2,
		),
	)

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2,
			2,
		),
	)

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkBack := widget.NewHyperlinkWithStyle(
		backtoIdentity,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkBack.OnTapped = func() {
		removeOverlays()
	}

	valueUsername := canvas.NewText(
		username,
		colors.Green,
	)
	valueUsername.TextSize = 22
	valueUsername.TextStyle = fyne.TextStyle{Bold: true}
	valueUsername.Alignment = fyne.TextAlignCenter

	btnSetPrimary := widget.NewButton(setPrimaryUser, nil)
	btnSetPrimary.OnTapped = func() {
		setPrimaryUsername(username)
		session.Username = username
		session.Window.SetContent(layoutIdentity())
		removeOverlays()
	}

	btnSend := widget.NewButton(transferUser, nil)

	inputAddress := widget.NewEntry()
	inputAddress.PlaceHolder = receiverContact
	inputAddress.Validator = func(s string) error {
		btnSend.Text = transferUser
		btnSend.Enable()
		btnSend.Refresh()
		valid, address, _ = checkUsername(s, -1)
		if !valid {
			_, err := globals.ParseValidateAddress(s)
			if err != nil {
				btnSend.Disable()
				btnSend.Refresh()
				err := errors.New(addressDoesntExist)
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
		if address != string_ && address != engram.Disk.GetAddress().String() {
			btnSend.Text = settinguptransfer
			btnSend.Disable()
			btnSend.Refresh()
			inputAddress.Disable()
			inputAddress.Refresh()
			err := transferUsername(username, address)
			if err != nil {
				address = string_
				btnSend.Text = txFailed
				btnSend.Disable()
				btnSend.Refresh()
				inputAddress.Enable()
				inputAddress.Refresh()
			} else {
				btnSend.Text = txConfirming
				btnSend.Refresh()
				go func() {
					walletapi.WaitNewHeightBlock()
					sHeight := walletapi.Get_Daemon_Height()

					for {
						found := false
						if session.Domain == appIdentity {
							usernames, err := queryUsernames()
							if err != nil {
								fmt.Printf(
									errQuerying,
									err,
								)
								return
							}

							for u := range usernames {
								if usernames[u] == username {
									found = true
								}
							}

							if !found {
								fmt.Printf(
									successTransfer,
									username,
									address,
								)
								session.Window.SetContent(layoutTransition())
								session.Window.SetContent(layoutIdentity())
								removeOverlays()
								break
							}

							// If we go DEFAULT_CONFIRMATION_TIMEOUT blocks without exiting 'Confirming...' loop, display failed to transfer and break
							if walletapi.Get_Daemon_Height() > sHeight+int64(DEFAULT_CONFIRMATION_TIMEOUT) {
								fmt.Printf(
									failTransfer,
									username,
									address,
								)
								session.Window.SetContent(layoutTransition())
								session.Window.SetContent(layoutIdentity())
								removeOverlays()
								break
							}

							// If daemon height has incremented, print retry counters into button space
							if walletapi.Get_Daemon_Height()-sHeight > 0 {
								btnSend.Text = fmt.Sprintf(
									txConfirmingSomeofSome,
									walletapi.Get_Daemon_Height()-sHeight,
									DEFAULT_CONFIRMATION_TIMEOUT,
								)
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
	rect.SetMinSize(
		fyne.NewSize(
			ui.Width*0.6,
			ui.Height*0.35,
		),
	)
	rect2 := canvas.NewRectangle(color.Transparent)
	rect2.SetMinSize(
		fyne.NewSize(
			ui.Width,
			1,
		),
	)
	frame := canvas.NewRectangle(color.Transparent)
	frame.SetMinSize(
		fyne.NewSize(
			ui.Width,
			ui.Height,
		),
	)
	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(
		fyne.NewSize(
			10,
			5,
		),
	)
	label := canvas.NewText(proofofWork, colors.Gray)
	label.TextStyle = fyne.TextStyle{Bold: true}
	label.TextSize = 12
	hashes := canvas.NewText(
		fmt.Sprintf(
			"%d",
			session.RegHashes,
		),
		colors.Account,
	)
	hashes.TextSize = 18

	go func() {
		for engram.Disk != nil {
			hashes.Text = fmt.Sprintf(
				"%d",
				session.RegHashes,
			)
			hashes.Refresh()
		}
	}()

	session.Gif, _ = x.NewAnimatedGifFromResource(resourceAnimation2Gif)
	session.Gif.SetMinSize(rect.MinSize())
	session.Gif.Resize(rect.MinSize())
	session.Gif.Start()

	waitForm := container.NewVBox(
		widget.NewLabel(string_),
		container.NewHBox(
			layout.NewSpacer(),
			title,
			layout.NewSpacer(),
		),
		widget.NewLabel(string_),
		heading,
		rectSpacer,
		sub,
		widget.NewLabel(string_),
		container.NewStack(
			session.Gif,
		),
		widget.NewLabel(string_),
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
		widget.NewLabel(string_),
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
	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(
		fyne.NewSize(
			ui.Width*0.6,
			ui.Width*0.35,
		),
	)
	frame := &iframe{}
	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(
		fyne.NewSize(
			10,
			5,
		),
	)
	wSpacer := widget.NewLabel(singlespace)

	title := canvas.NewText(
		string_,
		colors.Gray,
	)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 16
	title.Alignment = fyne.TextAlignCenter

	heading := canvas.NewText(
		string_,
		colors.Red,
	)
	heading.TextStyle = fyne.TextStyle{Bold: true}
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter

	sub := widget.NewRichTextFromMarkdown(string_)
	sub.Wrapping = fyne.TextWrapWord

	labelSettings := widget.NewHyperlinkWithStyle(
		reviewSettings,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)

	if t == 1 {
		title.Text = errorBanner
		heading.Text = connectFailure
		sub.ParseMarkdown("Connection to " + session.Daemon + " has failed. Please review your settings and try again.")
		labelSettings.Text = reviewSettings
		labelSettings.OnTapped = func() {
			session.Window.SetContent(layoutSettings())
		}
	} else if t == 2 {
		title.Text = errorBanner
		heading.Text = writeFailure
		sub.ParseMarkdown(writeFailMsg)
		labelSettings.Text = reviewSettings
		labelSettings.OnTapped = func() {
			session.Window.SetContent(layoutMain())
		}
	} else {
		title.Text = errorBanner
		heading.Text = id10tError
		sub.ParseMarkdown(systemMalfunction)
		labelSettings.Text = reviewSettings
		labelSettings.OnTapped = func() {
			session.Window.SetContent(layoutSettings())
		}
	}

	rectHeader := canvas.NewRectangle(color.Transparent)
	rectHeader.SetMinSize(
		fyne.NewSize(
			ui.Width, 1))

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
		widget.NewLabel(string_),
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

	view := string_

	header := canvas.NewText(
		txHistroy,
		colors.Green,
	)
	header.TextSize = 22
	header.TextStyle = fyne.TextStyle{Bold: true}

	details_header := canvas.NewText(
		txDetail,
		colors.Green,
	)
	details_header.TextSize = 22
	details_header.TextStyle = fyne.TextStyle{Bold: true}

	frame := &iframe{}
	rectWidth := canvas.NewRectangle(color.Transparent)
	rectWidth.SetMinSize(
		fyne.NewSize(
			ui.MaxWidth,
			10,
		),
	)
	rectWidth90 := canvas.NewRectangle(color.Transparent)
	rectWidth90.SetMinSize(
		fyne.NewSize(
			ui.Width,
			10,
		),
	)

	heading := canvas.NewText(
		historyBanner,
		colors.Gray,
	)
	heading.TextSize = 16
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(
		fyne.NewSize(
			ui.Width*0.3,
			35,
		),
	)

	rectMid := canvas.NewRectangle(color.Transparent)
	rectMid.SetMinSize(
		fyne.NewSize(
			ui.Width*0.35,
			35,
		),
	)

	results := canvas.NewText(
		string_,
		colors.Green,
	)
	results.TextSize = 13

	listData = binding.BindStringList(&data)
	listBox = widget.NewListWithData(listData,
		func() fyne.CanvasObject {
			return container.NewHBox(
				container.NewStack(
					rect,
					widget.NewLabel(string_),
				),
				container.NewStack(
					rectMid,
					widget.NewLabel(string_),
				),
				container.NewStack(
					rect,
					widget.NewLabel(string_),
				),
			)
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			dat := di.(binding.String)
			str, err := dat.Get()
			if err != nil {
				return
			}

			split := strings.Split(str, threesemicolons)

			co.(*fyne.Container).
				Objects[0].(*fyne.Container).
				Objects[1].(*widget.Label).
				SetText(split[0])
			co.(*fyne.Container).
				Objects[1].(*fyne.Container).
				Objects[1].(*widget.Label).
				SetText(split[1])
			co.(*fyne.Container).
				Objects[2].(*fyne.Container).
				Objects[1].(*widget.Label).
				SetText(split[3])
		},
	)

	menu := widget.NewSelect(
		[]string{
			stringNormal,
			stringCoinbase,
			stringMessages,
		},
		nil,
	)
	menu.PlaceHolder = selectTxType

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(
		fyne.NewSize(
			10,
			5,
		),
	)
	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(
		fyne.NewSize(
			ui.Width,
			ui.Height*0.60,
		),
	)

	menuLabel := canvas.NewText(moreOptionsBanner, colors.Gray)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2,
			2,
		),
	)

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2,
			2,
		),
	)

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkBack := widget.NewHyperlinkWithStyle(
		backtoDash,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
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
		case stringNormal:
			listBox.UnselectAll()
			results.Text = scanning
			results.Refresh()
			count := 0
			data = nil
			listData.Set(nil)
			entries = engram.Disk.Show_Transfers(
				zeroscid,
				false,
				true,
				true,
				0,
				engram.Disk.Get_Height(),
				string_,
				string_,
				0,
				0,
			)

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
							stamp = timefmt.Format(dateFormat)
							height = strconv.FormatUint(
								entries[e].Height,
								10,
							)
							amount := string_
							txid = entries[e].TXID

							if !entries[e].Incoming {
								direction = stringSent
								amount = "(" + globals.FormatMoney(entries[e].Amount) + ")"
							} else {
								direction = strinReceived
								amount = globals.FormatMoney(entries[e].Amount)
							}

							count += 1
							data = append(
								data,
								direction+
									threesemicolons+
									amount+
									threesemicolons+
									height+
									threesemicolons+
									stamp+
									threesemicolons+
									txid,
							)
						}
					}

					results.Text = fmt.Sprintf(

						centeredResults,

						count,
					)
					results.Refresh()

					listData.Set(data)

					listBox.OnSelected = func(id widget.ListItemID) {
						//var zeroscid crypto.Hash
						split := strings.Split(
							data[id],
							threesemicolons,
						)
						result := engram.Disk.Get_Payments_TXID(split[4])

						if result.TXID == string_ {
							label.Text = threedashes
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
				results.Text = fmt.Sprintf(

					centeredResults,

					count,
				)
				results.Refresh()
			}
		case stringCoinbase:
			listBox.UnselectAll()
			results.Text = scanning
			results.Refresh()
			count := 0
			data = nil
			listData.Set(nil)
			entries = engram.Disk.Show_Transfers(
				zeroscid,
				true,
				true,
				true,
				0,
				engram.Disk.Get_Height(),
				string_,
				string_,
				0,
				0,
			)

			if entries != nil {
				go func() {
					for e := range entries {
						var height string
						var direction string
						var stamp string

						entries[e].ProcessPayload()

						if entries[e].Coinbase {
							direction = stringNETWORK
							timefmt := entries[e].Time
							stamp = timefmt.Format(dateFormat)
							height = strconv.FormatUint(
								entries[e].Height,
								10,
							)
							amount := globals.FormatMoney(entries[e].Amount)
							txid = entries[e].TXID

							count += 1
							data = append(
								data,
								direction+
									threesemicolons+
									amount+
									threesemicolons+
									height+
									threesemicolons+
									stamp+
									threesemicolons+
									txid,
							)
						}
					}

					results.Text = fmt.Sprintf(
						centeredResults,
						count,
					)
					results.Refresh()

					listData.Set(data)

					listBox.OnSelected = func(id widget.ListItemID) {
						listBox.UnselectAll()
					}
					listBox.Refresh()
					listBox.ScrollToBottom()
				}()
			} else {
				results.Text = fmt.Sprintf(
					centeredResults,
					count,
				)
				results.Refresh()
			}
		case stringMessages:
			listBox.UnselectAll()
			results.Text = scanning
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
						stamp = timefmt.Format(dateFormat)

						temp := entries[e].Incoming
						if !temp {
							direction = sentLeft
						} else {
							direction = strinReceived
						}
						if entries[e].Payload_RPC.HasValue(
							rpc.RPC_COMMENT,
							rpc.DataString,
						) {
							contact := string_
							username := string_
							if entries[e].Payload_RPC.HasValue(
								rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
								rpc.DataString,
							) {
								contact = entries[e].Payload_RPC.Value(
									rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
									rpc.DataString,
								).(string)
								if len(contact) > 10 {
									username = contact[0:10] + doubleperiod
								} else {
									username = contact
								}
							}

							comment = entries[e].Payload_RPC.Value(
								rpc.RPC_COMMENT,
								rpc.DataString,
							).(string)
							if len(comment) > 10 {
								comment = comment[0:10] + doubleperiod
							}

							txid = entries[e].TXID
							count += 1
							data = append(
								data,
								direction+
									threesemicolons+
									username+
									threesemicolons+
									comment+
									threesemicolons+
									stamp+
									threesemicolons+
									txid+
									threesemicolons+
									contact,
							)
						}
					}

					results.Text = fmt.Sprintf(
						centeredResults,
						count,
					)
					results.Refresh()

					listData.Set(data)

					listBox.OnSelected = func(id widget.ListItemID) {
						split := strings.Split(
							data[id],
							threesemicolons,
						)
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
				results.Text = fmt.Sprintf(
					centeredResults,
					count,
				)
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
	wSpacer := widget.NewLabel(singlespace)

	rectWidth := canvas.NewRectangle(color.Transparent)
	rectWidth.SetMinSize(
		fyne.NewSize(
			ui.MaxWidth*0.99, 10))

	rectWidth90 := canvas.NewRectangle(color.Transparent)
	rectWidth90.SetMinSize(
		fyne.NewSize(
			ui.Width, 10))

	frame := &iframe{}

	heading := canvas.NewText(
		txdetailBanner,
		colors.Gray,
	)
	heading.TextSize = 16
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(
		fyne.NewSize(6, 5))

	labelTXID := canvas.NewText(
		txID,
		colors.Gray,
	)
	labelTXID.TextSize = 14
	labelTXID.Alignment = fyne.TextAlignLeading
	labelTXID.TextStyle = fyne.TextStyle{Bold: true}

	labelAmount := canvas.NewText(
		amount,
		colors.Gray,
	)
	labelAmount.TextSize = 14
	labelAmount.Alignment = fyne.TextAlignLeading
	labelAmount.TextStyle = fyne.TextStyle{Bold: true}

	labelDirection := canvas.NewText(
		payDirection,
		colors.Gray,
	)
	labelDirection.TextSize = 14
	labelDirection.Alignment = fyne.TextAlignLeading
	labelDirection.TextStyle = fyne.TextStyle{Bold: true}

	labelMember := canvas.NewText(
		string_,
		colors.Gray,
	)
	labelMember.TextSize = 14
	labelMember.Alignment = fyne.TextAlignLeading
	labelMember.TextStyle = fyne.TextStyle{Bold: true}

	labelProof := canvas.NewText(
		txProof,
		colors.Gray,
	)
	labelProof.TextSize = 14
	labelProof.Alignment = fyne.TextAlignLeading
	labelProof.TextStyle = fyne.TextStyle{Bold: true}

	labelDestPort := canvas.NewText(
		destinationPort,
		colors.Gray,
	)
	labelDestPort.TextSize = 14
	labelDestPort.TextStyle = fyne.TextStyle{Bold: true}

	labelSourcePort := canvas.NewText(
		sourcePort,
		colors.Gray,
	)
	labelSourcePort.TextSize = 14
	labelSourcePort.TextStyle = fyne.TextStyle{Bold: true}

	labelFees := canvas.NewText(
		txFees,
		colors.Gray,
	)
	labelFees.TextSize = 14
	labelFees.TextStyle = fyne.TextStyle{Bold: true}

	labelPayload := canvas.NewText(
		payLoad,
		colors.Gray,
	)
	labelPayload.TextSize = 14
	labelPayload.TextStyle = fyne.TextStyle{Bold: true}

	labelHeight := canvas.NewText(
		blockHeight,
		colors.Gray,
	)
	labelHeight.TextSize = 14
	labelHeight.TextStyle = fyne.TextStyle{Bold: true}

	labelReply := canvas.NewText(
		replyAddress,
		colors.Gray,
	)
	labelReply.TextSize = 14
	labelReply.TextStyle = fyne.TextStyle{Bold: true}

	labelSeparator := widget.NewRichTextFromMarkdown(string_)
	labelSeparator.Wrapping = fyne.TextWrapOff
	labelSeparator.ParseMarkdown(threedashes)

	labelSeparator2 := widget.NewRichTextFromMarkdown(string_)
	labelSeparator2.Wrapping = fyne.TextWrapOff
	labelSeparator2.ParseMarkdown(threedashes)

	labelSeparator3 := widget.NewRichTextFromMarkdown(string_)
	labelSeparator3.Wrapping = fyne.TextWrapOff
	labelSeparator3.ParseMarkdown(threedashes)

	labelSeparator4 := widget.NewRichTextFromMarkdown(string_)
	labelSeparator4.Wrapping = fyne.TextWrapOff
	labelSeparator4.ParseMarkdown(threedashes)

	labelSeparator5 := widget.NewRichTextFromMarkdown(string_)
	labelSeparator5.Wrapping = fyne.TextWrapOff
	labelSeparator5.ParseMarkdown(threedashes)

	labelSeparator6 := widget.NewRichTextFromMarkdown(string_)
	labelSeparator6.Wrapping = fyne.TextWrapOff
	labelSeparator6.ParseMarkdown(threedashes)

	labelSeparator7 := widget.NewRichTextFromMarkdown(string_)
	labelSeparator7.Wrapping = fyne.TextWrapOff
	labelSeparator7.ParseMarkdown(threedashes)

	labelSeparator8 := widget.NewRichTextFromMarkdown(string_)
	labelSeparator8.Wrapping = fyne.TextWrapOff
	labelSeparator8.ParseMarkdown(threedashes)

	labelSeparator9 := widget.NewRichTextFromMarkdown(string_)
	labelSeparator9.Wrapping = fyne.TextWrapOff
	labelSeparator9.ParseMarkdown(threedashes)

	labelSeparator10 := widget.NewRichTextFromMarkdown(string_)
	labelSeparator10.Wrapping = fyne.TextWrapOff
	labelSeparator10.ParseMarkdown(threedashes)

	menuLabel := canvas.NewText(moreOptionsBanner, colors.Gray)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2, 2))

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2, 2))

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	details := engram.Disk.Get_Payments_TXID(txid)

	stamp := string(details.Time.Format(time.RFC822))
	height := strconv.FormatUint(details.Height, 10)

	valueMember := widget.NewRichTextFromMarkdown(singlespace)
	valueMember.Wrapping = fyne.TextWrapBreak

	valueReply := widget.NewRichTextFromMarkdown(doubledashes)
	valueReply.Wrapping = fyne.TextWrapBreak

	if details.Payload_RPC.HasValue(
		rpc.RPC_REPLYBACK_ADDRESS,
		rpc.DataAddress,
	) {
		address := details.Payload_RPC.Value(
			rpc.RPC_REPLYBACK_ADDRESS,
			rpc.DataAddress,
		).(rpc.Address)
		valueReply.ParseMarkdown(string_ + address.String())
	} else if details.Payload_RPC.HasValue(
		rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
		rpc.DataString,
	) && details.DestinationPort == 1337 {
		address := details.Payload_RPC.Value(
			rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
			rpc.DataString,
		).(string)
		valueReply.ParseMarkdown(string_ + address)
	}

	valuePayload := widget.NewRichTextFromMarkdown(doubledashes)
	valuePayload.Wrapping = fyne.TextWrapBreak

	if details.Payload_RPC.HasValue(
		rpc.RPC_COMMENT,
		rpc.DataString,
	) {
		if details.Payload_RPC.Value(
			rpc.RPC_COMMENT,
			rpc.DataString,
		).(string) != string_ {
			valuePayload.ParseMarkdown(
				string_ + details.Payload_RPC.Value(
					rpc.RPC_COMMENT,
					rpc.DataString,
				).(string))
		}
	}

	valueAmount := canvas.NewText(string_, colors.Account)
	valueAmount.TextSize = 22
	valueAmount.TextStyle = fyne.TextStyle{Bold: true}

	valueDirection := canvas.NewText(string_, colors.Account)
	valueDirection.TextSize = 22
	valueDirection.TextStyle = fyne.TextStyle{Bold: true}
	if details.Incoming {
		valueDirection.Text = received
		labelMember.Text = senderAddress
		if details.Sender == string_ ||
			details.Sender == engram.Disk.GetAddress().String() {
			valueMember.ParseMarkdown(doubledashes)
		} else {
			valueMember.ParseMarkdown(string_ + details.Sender)
		}

		if details.Amount == 0 {
			valueAmount.Color = colors.Account
			valueAmount.Text = atomicZeroes
		} else {
			valueAmount.Color = colors.Green
			valueAmount.Text = plus + globals.FormatMoney(details.Amount)
		}
	} else {
		valueDirection.Text = sentRight
		labelMember.Text = receiverAddress
		valueMember.ParseMarkdown(string_ + details.Destination)

		if details.Amount == 0 {
			valueAmount.Color = colors.Account
			valueAmount.Text = atomicZeroes
		} else {
			valueAmount.Color = colors.Account
			valueAmount.Text = subtract + globals.FormatMoney(details.Amount)
		}
	}

	valueTime := canvas.NewText(
		stamp,
		colors.Account,
	)
	valueTime.TextSize = 14
	valueTime.TextStyle = fyne.TextStyle{Bold: true}

	valueFees := canvas.NewText(
		doublespace+globals.FormatMoney(details.Fees),
		colors.Account,
	)
	valueFees.TextSize = 22
	valueFees.TextStyle = fyne.TextStyle{Bold: true}

	valueHeight := canvas.NewText(
		doublespace+height,
		colors.Account,
	)
	valueHeight.TextSize = 22
	valueHeight.TextStyle = fyne.TextStyle{Bold: true}

	valueTXID := widget.NewRichTextFromMarkdown(string_)
	valueTXID.Wrapping = fyne.TextWrapBreak
	valueTXID.ParseMarkdown(string_ + txid)

	valuePort := canvas.NewText(
		string_,
		colors.Account,
	)
	valuePort.TextSize = 22
	valuePort.TextStyle = fyne.TextStyle{Bold: true}
	valuePort.Text = doublespace + strconv.FormatUint(
		details.DestinationPort,
		10,
	)

	valueSourcePort := canvas.NewText(string_, colors.Account)
	valueSourcePort.TextSize = 22
	valueSourcePort.TextStyle = fyne.TextStyle{Bold: true}
	valueSourcePort.Text = doublespace + strconv.FormatUint(
		details.SourcePort,
		10,
	)

	btnView := widget.NewButton(viewExplorer, nil)
	btnView.OnTapped = func() {
		if engram.Disk.GetNetwork() {
			link, _ := url.Parse(DEFAULT_EXPLORER_URL + "/tx/" + txid)
			_ = fyne.CurrentApp().OpenURL(link)
		} else {
			link, _ := url.Parse(DEFAULT_TESTNET_EXPLORER_URL + "/tx/" + txid)
			_ = fyne.CurrentApp().OpenURL(link)
		}
	}

	linkBack := widget.NewHyperlinkWithStyle(
		backtoHistory,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkBack.OnTapped = func() {
		overlay := session.Window.Canvas().Overlays()
		overlay.Top().Hide()
		overlay.Remove(overlay.Top())
		overlay.Remove(overlay.Top())
	}

	linkAddress := widget.NewHyperlinkWithStyle(
		copyAddress,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkAddress.OnTapped = func() {
		session.Window.Clipboard().SetContent(valueMember.String())
	}

	linkReplyAddress := widget.NewHyperlinkWithStyle(
		copyAddress,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkReplyAddress.OnTapped = func() {
		if _, ok := details.Payload_RPC.Value(
			rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
			rpc.DataString,
		).(string); ok {
			session.Window.Clipboard().SetContent(details.Payload_RPC.Value(
				rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
				rpc.DataString,
			).(string),
			)
		}
	}

	linkTXID := widget.NewHyperlinkWithStyle(
		copyTxId,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkTXID.OnTapped = func() {
		session.Window.Clipboard().SetContent(txid)
	}

	linkProof := widget.NewHyperlinkWithStyle(
		copyTxProof,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkProof.OnTapped = func() {
		session.Window.Clipboard().SetContent(details.Proof)
	}

	linkPayload := widget.NewHyperlinkWithStyle(
		copyPayload,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkPayload.OnTapped = func() {
		if _, ok := details.Payload_RPC.Value(
			rpc.RPC_COMMENT,
			rpc.DataString,
		).(string); ok {
			session.Window.Clipboard().SetContent(
				details.Payload_RPC.Value(
					rpc.RPC_COMMENT,
					rpc.DataString,
				).(string),
			)
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
						labelReply,
						rectSpacer,
						valueReply,
						container.NewHBox(
							linkReplyAddress,
							layout.NewSpacer(),
						),
						rectSpacer,
						rectSpacer,
						labelSeparator5,
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
						labelSeparator6,
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
						labelSeparator7,
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
						labelSeparator8,
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
						labelSeparator9,
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
						labelSeparator10,
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
	session.Domain = appDatapad
	title := canvas.NewText(datapadBanner, colors.Gray)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 16

	heading := canvas.NewText(string_, colors.Green)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	menuLabel := canvas.NewText(moreOptionsBanner, colors.Gray)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	entryNewPad := widget.NewEntry()
	entryNewPad.MultiLine = false
	entryNewPad.Wrapping = fyne.TextWrap(fyne.TextTruncateClip)

	btnAdd := widget.NewButton(centeredCreate, nil)
	btnAdd.Disable()
	btnAdd.OnTapped = func() {
		err := StoreEncryptedValue(
			stringDatapads,
			[]byte(entryNewPad.Text),
			[]byte(string_),
		)
		if err != nil {
			btnAdd.Text = errorCreatingDatapad
			btnAdd.Disable()
			btnAdd.Refresh()
		} else {
			session.Datapad = entryNewPad.Text
			session.Window.SetContent(layoutTransition())
			session.Window.SetContent(layoutDatapad())
			removeOverlays()
		}
	}

	entryNewPad.PlaceHolder = datapadName
	entryNewPad.Validator = func(s string) error {
		session.Datapad = s
		if len(s) > 0 {
			_, err := GetEncryptedValue(stringDatapads, []byte(s))
			if err == nil {
				btnAdd.Text = datapadExists
				btnAdd.Disable()
				btnAdd.Refresh()
				err := errors.New(userExists)
				entryNewPad.SetValidationError(err)
				return err
			} else {
				btnAdd.Text = stringCreate
				btnAdd.Enable()
				btnAdd.Refresh()
				return nil
			}
		} else {
			btnAdd.Text = stringCreate
			btnAdd.Disable()
			err := errors.New(plsNameDatapad)
			entryNewPad.SetValidationError(err)
			btnAdd.Refresh()
			return err
		}
	}
	entryNewPad.OnChanged = func(s string) {
		entryNewPad.Validate()
	}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2,
			2,
		),
	)

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2,
			2,
		),
	)

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkBack := widget.NewHyperlinkWithStyle(
		backtoDash,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkBack.OnTapped = func() {
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutDashboard())
		removeOverlays()
	}

	frame := &iframe{}

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(
		fyne.NewSize(10, 5))
	rectList := canvas.NewRectangle(color.Transparent)
	rectList.SetMinSize(
		fyne.NewSize(
			ui.Width, 35))
	rectListBox := canvas.NewRectangle(color.Transparent)
	rectListBox.SetMinSize(
		fyne.NewSize(
			ui.Width,
			350,
		),
	)

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

	tree, err := ss.GetTree(stringDatapads)
	if err != nil {
		padData = []string{}
	}

	cursor := tree.Cursor()

	for k, _, err := cursor.First(); err == nil; k, _, err = cursor.Next() {
		if string(k) != string_ {
			padData = append(
				padData,
				string(k),
			)
		}
	}

	padList := binding.BindStringList(&padData)

	padBox := widget.NewListWithData(
		padList,
		func() fyne.CanvasObject {
			c := container.NewVBox(
				widget.NewLabel(string_),
			)
			return c
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			dat := di.(binding.String)
			str, err := dat.Get()
			if err != nil {
				return
			}

			co.(*fyne.Container).
				Objects[0].(*widget.Label).
				SetText(str)
			co.(*fyne.Container).
				Objects[0].(*widget.Label).
				Wrapping = fyne.TextWrapWord
			co.(*fyne.Container).
				Objects[0].(*widget.Label).
				TextStyle.Bold = false
			co.(*fyne.Container).
				Objects[0].(*widget.Label).
				Alignment = fyne.TextAlignLeading
		},
	)

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
		container.NewCenter(
			container.NewVBox(
				title,
				rectSpacer,
			),
		),
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
	rectWidth := canvas.NewRectangle(color.Transparent)
	rectWidth.SetMinSize(
		fyne.NewSize(
			ui.MaxWidth,
			10,
		),
	)

	rectWidth90 := canvas.NewRectangle(color.Transparent)
	rectWidth90.SetMinSize(
		fyne.NewSize(
			ui.Width,
			10,
		),
	)

	rectEntry := canvas.NewRectangle(color.Transparent)
	rectEntry.SetMinSize(
		fyne.NewSize(
			ui.Width,
			ui.Height*0.56,
		),
	)

	heading := canvas.NewText(session.Datapad, colors.Green)
	heading.TextSize = 20
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(
		fyne.NewSize(
			6,
			5,
		),
	)

	menuLabel := canvas.NewText(
		moreOptionsBanner,
		colors.Gray,
	)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	selectOptions := widget.NewSelect(
		[]string{
			stringClear,
			exportPlainText,
			stringDelete,
		},
		nil,
	)
	selectOptions.PlaceHolder = selectOption

	data, err := GetEncryptedValue(
		stringDatapads,
		[]byte(session.Datapad),
	)
	if err != nil {
		data = nil
	}

	overlay := session.Window.Canvas().Overlays()

	btnSave := widget.NewButton(
		stringSave,
		nil,
	)

	entryPad := widget.NewEntry()
	entryPad.Wrapping = fyne.TextWrapWord

	selectOptions.OnChanged = func(s string) {
		if s == stringClear {
			header := canvas.NewText(
				datapadReset,
				colors.Gray,
			)
			header.TextSize = 14
			header.Alignment = fyne.TextAlignCenter
			header.TextStyle = fyne.TextStyle{Bold: true}

			subHeader := canvas.NewText(
				clearDatapad,
				colors.Account,
			)
			subHeader.TextSize = 22
			subHeader.Alignment = fyne.TextAlignCenter
			subHeader.TextStyle = fyne.TextStyle{Bold: true}

			linkClose := widget.NewHyperlinkWithStyle(
				stringCancel,
				nil,
				fyne.TextAlignCenter,
				fyne.TextStyle{
					Bold: true,
				},
			)
			linkClose.OnTapped = func() {
				overlay := session.Window.Canvas().Overlays()
				overlay.Top().Hide()
				overlay.Remove(overlay.Top())
				overlay.Remove(overlay.Top())
				selectOptions.Selected = selectOption
				selectOptions.Refresh()
			}

			btnSubmit := widget.NewButton(stringClear, nil)

			btnSubmit.OnTapped = func() {
				if session.Datapad != string_ {
					err := StoreEncryptedValue(
						stringDatapads,
						[]byte(session.Datapad),
						[]byte(string_),
					)
					if err != nil {
						fmt.Printf(
							errDatashard,
							err,
						)
						selectOptions.Selected = selectOption
						selectOptions.Refresh()
						return
					}

					selectOptions.Selected = selectOption
					selectOptions.Refresh()
					entryPad.Text = string_
					entryPad.Refresh()
				}

				overlay := session.Window.Canvas().Overlays()
				overlay.Top().Hide()
				overlay.Remove(overlay.Top())
				overlay.Remove(overlay.Top())
				selectOptions.Selected = selectOption
				selectOptions.Refresh()
			}

			span := canvas.NewRectangle(color.Transparent)
			span.SetMinSize(
				fyne.NewSize(
					ui.Width, 10))

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
							widget.NewLabel(string_),
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
		} else if s == exportPlainText {
			data := []byte(entryPad.Text)
			err := os.WriteFile(AppPath()+string(filepath.Separator)+session.Datapad+".txt", data, 0644)
			if err != nil {
				fmt.Printf(
					errDatashard,
					err,
				)
				selectOptions.Selected = selectOption
				selectOptions.Refresh()
				return
			}

			selectOptions.Selected = selectOption
			selectOptions.Refresh()
		} else if s == stringDelete {
			header := canvas.NewText(
				datapadReset,
				colors.Gray,
			)
			header.TextSize = 14
			header.Alignment = fyne.TextAlignCenter
			header.TextStyle = fyne.TextStyle{Bold: true}

			subHeader := canvas.NewText(
				deleteDatapad,
				colors.Account,
			)
			subHeader.TextSize = 22
			subHeader.Alignment = fyne.TextAlignCenter
			subHeader.TextStyle = fyne.TextStyle{Bold: true}

			linkClose := widget.NewHyperlinkWithStyle(
				stringCancel,
				nil,
				fyne.TextAlignCenter,
				fyne.TextStyle{
					Bold: true,
				},
			)
			linkClose.OnTapped = func() {
				overlay := session.Window.Canvas().Overlays()
				overlay.Top().Hide()
				overlay.Remove(overlay.Top())
				overlay.Remove(overlay.Top())
				selectOptions.Selected = selectOption
				selectOptions.Refresh()
			}

			btnSubmit := widget.NewButton(stringDelete, nil)

			btnSubmit.OnTapped = func() {
				if session.Datapad != string_ {
					err := DeleteKey(
						stringDatapads,
						[]byte(session.Datapad),
					)
					if err != nil {
						fmt.Printf(
							errDatashard,
							err,
						)
						selectOptions.Selected = selectOption
						selectOptions.Refresh()
						fmt.Printf(
							errDeletingDatapad,
							session.Datapad,
							err,
						)
					} else {
						session.Datapad = string_
						session.DatapadChanged = false
						removeOverlays()
						session.Window.SetContent(layoutTransition())
						session.Window.SetContent(layoutDatapad())
					}
				}
			}

			span := canvas.NewRectangle(color.Transparent)
			span.SetMinSize(
				fyne.NewSize(
					ui.Width, 10))

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
							widget.NewLabel(string_),
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
			session.Datapad = string_
			session.DatapadChanged = false
			overlay := session.Window.Canvas().Overlays()
			overlay.Top().Hide()
			overlay.Remove(overlay.Top())
			overlay.Remove(overlay.Top())
			selectOptions.Selected = selectOption
			selectOptions.Refresh()
		}
	}

	btnSave.OnTapped = func() {
		err = StoreEncryptedValue(
			stringDatapads,
			[]byte(session.Datapad),
			[]byte(entryPad.Text),
		)
		if err != nil {
			btnSave.Text = errorSaving
			btnSave.Disable()
			btnSave.Refresh()
		} else {
			session.DatapadChanged = false
			btnSave.Text = stringSave
			btnSave.Disable()
			heading.Text = session.Datapad
			heading.Refresh()
		}
	}

	session.DatapadChanged = false

	btnSave.Text = stringSave
	btnSave.Disable()

	entryPad.MultiLine = true
	entryPad.Text = string(data)
	entryPad.OnChanged = func(s string) {
		session.DatapadChanged = true
		heading.Text = session.Datapad + singleasterisks
		heading.Refresh()
		btnSave.Text = stringSave
		btnSave.Enable()
	}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2, 2))

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2, 2))

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkBack := widget.NewHyperlinkWithStyle(
		backtoDatapad,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkBack.OnTapped = func() {
		if session.DatapadChanged {
			header := canvas.NewText(
				datapadChanged,
				colors.Gray,
			)
			header.TextSize = 14
			header.Alignment = fyne.TextAlignCenter
			header.TextStyle = fyne.TextStyle{Bold: true}

			subHeader := canvas.NewText(
				saveDatapad,
				colors.Account,
			)
			subHeader.TextSize = 22
			subHeader.Alignment = fyne.TextAlignCenter
			subHeader.TextStyle = fyne.TextStyle{Bold: true}

			linkClose := widget.NewHyperlinkWithStyle(
				discardDatapad,
				nil,
				fyne.TextAlignCenter,
				fyne.TextStyle{
					Bold: true,
				},
			)
			linkClose.OnTapped = func() {
				session.Datapad = string_
				session.DatapadChanged = false
				removeOverlays()
			}

			btnSubmit := widget.NewButton(stringSave, nil)

			btnSubmit.OnTapped = func() {
				err = StoreEncryptedValue(
					stringDatapads,
					[]byte(session.Datapad),
					[]byte(entryPad.Text),
				)
				if err != nil {
					btnSave.Text = errorSaving
					btnSave.Disable()
					btnSave.Refresh()
				} else {
					session.Datapad = string_
					session.DatapadChanged = false
					btnSave.Text = stringSave
					removeOverlays()
				}
			}

			span := canvas.NewRectangle(color.Transparent)
			span.SetMinSize(
				fyne.NewSize(
					ui.Width, 10))

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
							widget.NewLabel(string_),
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
			session.Datapad = string_
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
	rectWidth := canvas.NewRectangle(color.Transparent)
	rectWidth.SetMinSize(
		fyne.NewSize(
			ui.MaxWidth*0.99,
			10,
		),
	)
	rectWidth90 := canvas.NewRectangle(color.Transparent)
	rectWidth90.SetMinSize(
		fyne.NewSize(
			ui.Width,
			10,
		),
	)
	rectBox := canvas.NewRectangle(color.Transparent)
	rectBox.SetMinSize(
		fyne.NewSize(
			ui.MaxWidth*0.99,
			ui.MaxHeight*0.80,
		),
	)

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(
		fyne.NewSize(
			10,
			5,
		))

	title := canvas.NewText(myaccountBanner, colors.Gray)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 16

	heading := canvas.NewText(
		engram.Disk.GetAddress().String()[0:5]+
			threeperiods+
			engram.Disk.GetAddress().String()[len(engram.Disk.GetAddress().String())-
				10:len(engram.Disk.GetAddress().String())],
		colors.Green,
	)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	labelPassword := canvas.NewText(
		newpassBanner,
		colors.Gray,
	)
	labelPassword.TextStyle = fyne.TextStyle{Bold: true}
	labelPassword.TextSize = 11
	labelPassword.Alignment = fyne.TextAlignCenter

	menuLabel := canvas.NewText(
		moreOptionsBanner,
		colors.Gray,
	)
	menuLabel.TextSize = 11
	menuLabel.Alignment = fyne.TextAlignCenter
	menuLabel.TextStyle = fyne.TextStyle{Bold: true}

	labelRecovery := canvas.NewText(
		accountrecoveryBanner,
		colors.Gray,
	)
	labelRecovery.TextSize = 11
	labelRecovery.Alignment = fyne.TextAlignCenter
	labelRecovery.TextStyle = fyne.TextStyle{Bold: true}

	sep := canvas.NewRectangle(colors.Gray)
	sep.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2,
			2,
		),
	)

	line1 := container.NewVBox(
		layout.NewSpacer(),
		sep,
		layout.NewSpacer(),
	)

	sep2 := canvas.NewRectangle(colors.Gray)
	sep2.SetMinSize(
		fyne.NewSize(
			ui.Width*0.2,
			2,
		),
	)

	line2 := container.NewVBox(
		layout.NewSpacer(),
		sep2,
		layout.NewSpacer(),
	)

	linkBack := widget.NewHyperlinkWithStyle(
		backtoDash,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkBack.OnTapped = func() {
		session.LastDomain = session.Window.Content()
		session.Window.SetContent(layoutTransition())
		session.Window.SetContent(layoutDashboard())
		removeOverlays()
	}

	linkCopyAddress := widget.NewHyperlinkWithStyle(
		copyAddress,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkCopyAddress.OnTapped = func() {
		session.Window.Clipboard().SetContent(engram.Disk.GetAddress().String())
	}

	btnClear := widget.NewButton(
		deleteDatashard,
		nil,
	)
	btnClear.OnTapped = func() {
		header := canvas.NewText(
			datashardDelete,
			colors.Gray,
		)
		header.TextSize = 14
		header.Alignment = fyne.TextAlignCenter
		header.TextStyle = fyne.TextStyle{Bold: true}

		subHeader := canvas.NewText(
			confirmDelete,
			colors.Account,
		)
		subHeader.TextSize = 22
		subHeader.Alignment = fyne.TextAlignCenter
		subHeader.TextStyle = fyne.TextStyle{Bold: true}

		linkClose := widget.NewHyperlinkWithStyle(
			stringCancel,
			nil,
			fyne.TextAlignCenter,
			fyne.TextStyle{
				Bold: true,
			},
		)
		linkClose.OnTapped = func() {
			session.Datapad = string_
			session.DatapadChanged = false
			removeOverlays()
		}

		btnSubmit := widget.NewButton(
			deleteDatashard,
			nil,
		)

		btnSubmit.OnTapped = func() {
			err := cleanWalletData()
			if err != nil {
				btnSubmit.Text = errorDelete
				btnSubmit.Disable()
				btnSubmit.Refresh()
			} else {
				btnSubmit.Text = deleteSuccess
				btnSubmit.Disable()
				btnSubmit.Refresh()
				removeOverlays()
			}
		}

		span := canvas.NewRectangle(color.Transparent)
		span.SetMinSize(
			fyne.NewSize(
				ui.Width, 10))

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
						widget.NewLabel(string_),
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

	btnSeed := widget.NewButton(
		accessRecovery,
		nil,
	)
	btnSeed.OnTapped = func() {
		overlay := session.Window.Canvas().Overlays()

		header := canvas.NewText(
			accountVerification,
			colors.Gray,
		)
		header.TextSize = 14
		header.Alignment = fyne.TextAlignCenter
		header.TextStyle = fyne.TextStyle{Bold: true}

		subHeader := canvas.NewText(
			confirmPassword,
			colors.Account,
		)
		subHeader.TextSize = 22
		subHeader.Alignment = fyne.TextAlignCenter
		subHeader.TextStyle = fyne.TextStyle{Bold: true}

		linkClose := widget.NewHyperlinkWithStyle(
			stringCancel,
			nil,
			fyne.TextAlignCenter,
			fyne.TextStyle{
				Bold: true,
			},
		)
		linkClose.OnTapped = func() {
			overlay := session.Window.Canvas().Overlays()
			overlay.Top().Hide()
			overlay.Remove(overlay.Top())
			overlay.Remove(overlay.Top())
		}

		btnConfirm := widget.NewButton(
			stringSubmit,
			nil,
		)

		entryPassword := NewReturnEntry()
		entryPassword.Password = true
		entryPassword.PlaceHolder = stringPassword
		entryPassword.OnChanged = func(s string) {
			if s == string_ {
				btnConfirm.Text = stringSubmit
				btnConfirm.Disable()
				btnConfirm.Refresh()
			} else {
				btnConfirm.Text = stringSubmit
				btnConfirm.Enable()
				btnConfirm.Refresh()
			}
		}

		btnConfirm.OnTapped = func() {
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
				btnConfirm.Text = invalidPassword
				btnConfirm.Disable()
				btnConfirm.Refresh()
			}
		}

		entryPassword.OnReturn = btnConfirm.OnTapped

		btnConfirm.Disable()

		span := canvas.NewRectangle(color.Transparent)
		span.SetMinSize(
			fyne.NewSize(
				ui.Width, 10))

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
						widget.NewLabel(string_),
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
	}

	btnChange := widget.NewButton(
		stringSubmit,
		nil,
	)
	btnChange.Disable()

	curPass := widget.NewEntry()
	curPass.Password = true
	curPass.PlaceHolder = currentPass
	curPass.OnChanged = func(s string) {
		btnChange.Text = stringSubmit
		btnChange.Enable()
		btnChange.Refresh()
	}

	newPass := widget.NewEntry()
	newPass.Password = true
	newPass.PlaceHolder = newPassword
	newPass.OnChanged = func(s string) {
		btnChange.Text = stringSubmit
		btnChange.Enable()
		btnChange.Refresh()
	}

	confirm := widget.NewEntry()
	confirm.Password = true
	confirm.PlaceHolder = confirmPassword
	confirm.OnChanged = func(s string) {
		btnChange.Text = stringSubmit
		btnChange.Enable()
		btnChange.Refresh()
	}

	btnChange.OnTapped = func() {
		if engram.Disk.Check_Password(curPass.Text) {
			if newPass.Text == confirm.Text &&
				newPass.Text != string_ {
				err := engram.Disk.Set_Encrypted_Wallet_Password(newPass.Text)
				if err != nil {
					btnChange.Text = erroPassword
					btnChange.Disable()
					btnChange.Refresh()
				} else {
					curPass.Text = string_
					curPass.Refresh()
					newPass.Text = string_
					newPass.Refresh()
					confirm.Text = string_
					confirm.Refresh()
					btnChange.Text = passUpdated
					btnChange.Disable()
					btnChange.Refresh()
					engram.Disk.Save_Wallet()
				}
			} else {
				btnChange.Text = passMismatch
				btnChange.Disable()
				btnChange.Refresh()
			}
		} else {
			btnChange.Text = passIncorrect
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
				widget.NewLabel(string_),
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
				widget.NewLabel(string_),
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
				widget.NewLabel(string_),
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
				widget.NewLabel(string_),
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
	wSpacer := widget.NewLabel(singlespace)
	heading := canvas.NewText(
		recoveryWords,
		colors.Green,
	)
	heading.TextSize = 22
	heading.Alignment = fyne.TextAlignCenter
	heading.TextStyle = fyne.TextStyle{Bold: true}

	rectStatus := canvas.NewRectangle(color.Transparent)
	rectStatus.SetMinSize(
		fyne.NewSize(10, 10))

	rectHeader := canvas.NewRectangle(color.Transparent)
	rectHeader.SetMinSize(
		fyne.NewSize(
			ui.Width, 10))

	linkCancel := widget.NewHyperlinkWithStyle(
		backtoAccount,
		nil,
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)
	linkCancel.OnTapped = func() {
		removeOverlays()
	}

	rectSpacer := canvas.NewRectangle(color.Transparent)
	rectSpacer.SetMinSize(
		fyne.NewSize(
			ui.Width, 5))

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

	body := widget.NewLabel(recoveryWarning)
	body.Wrapping = fyne.TextWrapWord
	body.Alignment = fyne.TextAlignCenter
	body.TextStyle = fyne.TextStyle{Bold: true}

	btnCopySeed := widget.NewButton(copyRecovery, nil)

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
	scrollBox.SetMinSize(
		fyne.NewSize(
			ui.MaxWidth, ui.Height*0.74))

	formatted := strings.Split(
		engram.Disk.GetSeed(),
		singlespace,
	)

	rect := canvas.NewRectangle(
		color.RGBA{
			19,
			25,
			34,
			255,
		},
	)
	rect.SetMinSize(
		fyne.NewSize(
			ui.Width,
			25,
		),
	)

	for i := 0; i < len(formatted); i++ {
		pos := fmt.Sprintf(
			"%d",
			i+1,
		)
		word := strings.ReplaceAll(
			formatted[i],
			singlespace,
			string_,
		)
		grid.Add(container.NewStack(
			rect,
			container.NewHBox(
				widget.NewLabel(singlespace),
				widget.NewLabelWithStyle(
					pos,
					fyne.TextAlignCenter,
					fyne.TextStyle{
						Bold: true,
					},
				),
				layout.NewSpacer(),
				widget.NewLabelWithStyle(
					word,
					fyne.TextAlignLeading,
					fyne.TextStyle{
						Bold: true,
					},
				),
				widget.NewLabel(singlespace),
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

	resizeWindow(
		ui.MaxWidth,
		ui.MaxHeight)
	session.Window.SetContent(layout)
	session.Window.SetFixedSize(
		false)

	go func() {
		time.Sleep(time.Second * 2)
		removeOverlays()

		ui.MaxWidth = entry.Size().Width
		ui.MaxHeight = entry.Size().Height

		ui.Width = ui.MaxWidth * 0.9
		ui.Height = ui.MaxHeight
		ui.Padding = ui.MaxWidth * 0.05

		resizeWindow(
			ui.MaxWidth,
			ui.MaxHeight)
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
