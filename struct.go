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
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	x "fyne.io/x/fyne/widget"
	"github.com/civilware/Gnomon/indexer"
	"github.com/civilware/Gnomon/storage"
	"github.com/creachadair/jrpc2"
	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/rpc"
	"github.com/deroproject/derohe/transaction"
	"github.com/deroproject/derohe/walletapi"
	"github.com/deroproject/derohe/walletapi/rpcserver"
	"github.com/gorilla/websocket"
)

type (
	App struct {
		App    fyne.App
		Window fyne.Window
		Focus  bool
	}

	UI struct {
		Padding   float32
		MaxWidth  float32
		Width     float32
		MaxHeight float32
		Height    float32
	}

	Colors struct {
		Network    color.Color
		Account    color.Color
		Blue       color.Color
		Red        color.Color
		DarkGreen  color.Color
		Green      color.Color
		Gray       color.Color
		Yellow     color.Color
		DarkMatter color.Color
		Cold       color.Color
		Flint      color.Color
	}

	Navigation struct {
		PosX float32
		PosY float32
		CurX float32
		CurY float32
	}

	Session struct {
		Window            fyne.Window
		DesktopMode       bool
		Domain            string
		LastDomain        fyne.CanvasObject
		Testnet           bool
		Offline           bool
		Language          int
		ID                string
		Link              string
		Type              string
		Daemon            string
		WalletOpen        bool
		Username          string
		Datapad           string
		DatapadChanged    bool
		LastBalance       uint64
		Balance           uint64
		BalanceUSD        string
		BalanceText       *canvas.Text
		BalanceUSDText    *canvas.Text
		ModeText          *canvas.Text
		IDText            *canvas.Text
		LinkText          *canvas.Text
		StatusText        *canvas.Text
		Path              string
		Name              string
		Password          string
		PasswordConfirm   string
		DaemonHeight      uint64
		WalletHeight      uint64
		RPCServer         *rpcserver.RPCServer
		Verified          bool
		Dashboard         string
		Error             string
		NewUser           string
		Gif               *x.AnimatedGif
		RegHashes         int64
		LimitMessages     bool
		TrackRecentBlocks int64
	}

	Cyberdeck struct {
		user     string
		pass     string
		userText *widget.Entry
		passText *widget.Entry
		toggle   *widget.Button
		status   *canvas.Text
		server   *rpcserver.RPCServer
	}

	Engram struct {
		Disk *walletapi.Wallet_Disk
	}

	Theme struct {
		main eTheme
		alt  eTheme2
	}

	Gnomon struct {
		Active   int
		Index    *indexer.Indexer
		BBolt    *storage.BboltStore
		Graviton *storage.GravitonStore
		Path     string
	}

	ProofData struct {
		Receivers []string
		Amounts   []uint64
		Payloads  []string
	}

	Status struct {
		Canvas     *canvas.Text
		Message    string
		Network    *canvas.Text
		Connection *canvas.Circle
		Sync       *canvas.Circle
		Cyberdeck  *canvas.Circle
	}

	Transfers struct {
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

	Messages struct {
		Contact string
		Data    []string
		Box     *widget.List
		List    binding.ExternalStringList
		Height  uint64
		Message string
	}

	InstallContract struct {
		TXID string
	}

	Client struct {
		WS  *websocket.Conn
		RPC *jrpc2.Client
	}

	Res struct {
		bg          *canvas.Image
		bg2         *canvas.Image
		icon        *canvas.Image
		icon_sm     *canvas.Image
		load        *canvas.Image
		loading     *x.AnimatedGif
		header      *canvas.Image
		spacer      *canvas.Image
		dero        *canvas.Image
		gram        *canvas.Image
		block       *canvas.Image
		red_alert   *canvas.Image
		green_alert *canvas.Image
	}
)
