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
	"fmt"
	"os"
	"time"

	"fyne.io/fyne/v2/canvas"
	derodpkg "github.com/civilware/derodpkg/cmd"
	"github.com/deroproject/derohe/blockchain"
	drpc "github.com/deroproject/derohe/cmd/derod/rpc"
)

type Node struct {
	Active         int
	Chain          *blockchain.Blockchain
	Server         *drpc.RPCServer
	Work           string
	Height         int64
	TopoHeight     int64
	BestHeight     int64
	BestTopoHeight int64
	Peers          uint64
	Mempool        int
	Regpool        int
	Hashrate       string
	Miners         int
	MiniBlocks     string
	Offset         string
	OffsetNTP      string
	OffsetP2P      string
	Label          *canvas.Text
	LabelBlock     *canvas.Text
	Init           bool
	Exit           chan os.Signal
}

var node Node

// BUG: Only run this one time per app session, otherwise it will panic.
// Will work on a solution in future versions.
func startDaemon() {
	initSettings()
	initparams := make(map[string]interface{})

	// Define all input params for derod - need to be sure most/all are the default from standard command line parser as we are not using that here
	if !session.Network {
		initparams["--testnet"] = true
		initparams["--rpc-bind"] = DEFAULT_LOCAL_TESTNET_DAEMON
		initparams["--p2p-bind"] = DEFAULT_LOCAL_TESTNET_P2P
		initparams["--getwork-bind"] = DEFAULT_LOCAL_TESTNET_WORK
		initparams["--integrator-address"] = "deto1qy0ehnqjpr0wxqnknyc66du2fsxyktppkr8m8e6jvplp954klfjz2qqdzcd8p"
	} else {
		initparams["--testnet"] = false
		initparams["--rpc-bind"] = DEFAULT_LOCAL_DAEMON
		initparams["--p2p-bind"] = DEFAULT_LOCAL_P2P
		initparams["--getwork-bind"] = DEFAULT_LOCAL_WORK
		initparams["--integrator-address"] = "dero1qykyta6ntpd27nl0yq4xtzaf4ls6p5e9pqu0k2x4x3pqq5xavjsdxqgny8270"
	}
	initparams["--fastsync"] = true

	node.Chain = derodpkg.InitializeDerod(initparams)
	node.Server = derodpkg.StartDerod(node.Chain)
	node.Active = 1
	go pulse()
}

func stopDaemon() {
	node.Active = 0
	node.Server.RPCServer_Stop()
	node.Chain.Shutdown()
}

func pulse() {
	for node.Active == 1 {
		if !node.Chain.Sync {
			status.Connection.FillColor = colors.Yellow
			status.Connection.Refresh()
		} else {
			if node.Chain.Get_Height() > 0 {
				syncing := false
				n := node.Chain.Get_Top_ID()
				_, err := node.Chain.Store.Block_tx_store.ReadBlockHeight(n)
				if err != nil {
					syncing = true
				} else {
					syncing = false
				}

				fmt.Printf("[Daemon]  Height >> %d\n", node.Chain.Get_Height())

				if !syncing {
					status.Connection.FillColor = colors.Green
					status.Connection.Refresh()
				} else {
					status.Connection.FillColor = colors.Yellow
					status.Connection.Refresh()
				}
			} else {
				status.Connection.FillColor = colors.Red
				status.Connection.Refresh()
			}
		}

		time.Sleep(time.Second)
	}
}
