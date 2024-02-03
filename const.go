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

// Constants
const (
	// ports
	DEFAULT_TESTNET_WALLET_PORT = 40403
	DEFAULT_TESTNET_DAEMON_PORT = 40402
	DEFAULT_TESTNET_WORK_PORT   = 40400
	DEFAULT_WALLET_PORT         = 10103
	DEFAULT_DAEMON_PORT         = 10102
	DEFAULT_WORK_PORT           = 10100

	// endpoints
	DEFAULT_LOCAL_IP_ADDRESS = "127.0.0.1"

	// testnet
	DEFAULT_LOCAL_TESTNET_WALLET_RPC = "127.0.0.1:40403"
	DEFAULT_LOCAL_TESTNET_DAEMON     = "127.0.0.1:40402"
	DEFAULT_LOCAL_TESTNET_P2P        = "127.0.0.1:40401"
	DEFAULT_LOCAL_TESTNET_WORK       = "0.0.0.0:40400"
	DEFAULT_REMOTE_TESTNET_DAEMON    = "testnetexplorer.dero.io:40402"
	DEFAULT_TESTNET_EXPLORER_URL     = "https://testnetexplorer.dero.io"

	// mainnet
	DEFAULT_LOCAL_WALLET_RPC = "127.0.0.1:10103"
	DEFAULT_LOCAL_DAEMON     = "127.0.0.1:10102"
	DEFAULT_LOCAL_P2P        = "127.0.0.1:10101"
	DEFAULT_DISCOVER_DAEMON  = "0.0.0.0:10102"
	DEFAULT_LOCAL_WORK       = "0.0.0.0:10100"
	DEFAULT_REMOTE_DAEMON    = "node.derofoundation.org:11012"
	DEFAULT_EXPLORER_URL     = "https://explorer.dero.io"

	// wallet
	DEFAULT_CONFIRMATION_TIMEOUT = 5
	DEFAULT_MAX_RINGSIZE         = 128

	// platform
	DERO_DEVELOPER_MAINNET_ADDRESS   = "dero1qykyta6ntpd27nl0yq4xtzaf4ls6p5e9pqu0k2x4x3pqq5xavjsdxqgny8270"
	DERO_DEVELOPER_TESTNET_ADDRESS   = "deto1qy0ehnqjpr0wxqnknyc66du2fsxyktppkr8m8e6jvplp954klfjz2qqdzcd8p"
	DERO_DEVELOPER_SIMULATOR_ADDRESS = "deto1qyvyeyzrcm2fzf6kyq7egkes2ufgny5xn77y6typhfx9s7w3mvyd5qqynr5hx"

	// daemon
	DAEMON_GET_GAS_ESTIMATE = "DERO.GetGasEstimate"
	DAEMON_GET_SC           = "DERO.GetSC"
	DAEMON_GET_TX           = "DERO.GetTransaction"
	DAEMON_NAME_TO_ADDRESS  = "DERO.NameToAddress"

	// contract functions
	DEFAULT_SC_RINGSIZE     = 2
	DEFAULT_SC_ENTRYPOINT   = "entrypoint"
	POST_STRING             = "POST"
	GET_STRING              = "GET"
	CONTENT_TYPE_STRING     = "Content-Type"
	APP_OCTET_STREAM_STRING = "application/octet-stream"
	APP_JSON_STRING         = "application/json"

	// name_service
	DERO_NAME_SERVICE_SCID    = "0000000000000000000000000000000000000000000000000000000000000001"
	TRANSFER_OWNERSHIP_STRING = "TransferOwnership"
	NAME_STRING               = "name"
	OWNER_STRING              = "owner"
	NEWOWNER_STRING           = "newowner"
	REGISTER_STRING           = "Register"
)
