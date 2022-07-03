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
	"crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/url"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"

	//"github.com/deroproject/derohe/astrobwt/astrobwt_fast"
	"github.com/deroproject/derohe/astrobwt/astrobwtv3"
	"github.com/deroproject/derohe/block"
	"github.com/deroproject/derohe/globals"
	"github.com/deroproject/derohe/rpc"
	"github.com/deroproject/derohe/walletapi"

	"github.com/go-logr/logr"

	"github.com/gorilla/websocket"
)

type Netrunner struct {
	Mission     int64
	Height      int64
	Blocks      uint64
	MiniBlocks  uint64
	Hashrate    string
	NWHashrate  string
	Connection  *websocket.Conn
	Label       *canvas.Text
	LabelBlocks *canvas.Text
	Account     *walletapi.Wallet_Disk
	Threads     int64
	Daemon      string
	BlockList   []string
	ScrollBox   *widget.List
	Data        binding.StringList
}

var nr Netrunner
var mutex sync.RWMutex
var job rpc.GetBlockTemplate_Result
var job_counter int64
var maxdelay int = 10000
var threads int
var iterations int = 100
var max_pow_size int = 819200 //astrobwt.MAX_LENGTH
var wallet_address string
var daemon_rpc_address string

var counter uint64
var hash_rate uint64
var Difficulty uint64
var our_height int64

var block_counter uint64
var mini_block_counter uint64

var logger logr.Logger

var Exit_In_Progress = make(chan bool)

func startRunner(w *walletapi.Wallet_Disk, d string, t int) {
	Exit_In_Progress = make(chan bool)
	fmt.Printf("[Netrunner] Started with [%d] thread(s)... Good luck!\n", t)

	globals.Arguments["--wallet-address"] = w.GetAddress().String()
	globals.Arguments["--daemon-rpc-address"] = d
	globals.Arguments["--mining-threads"] = strconv.Itoa(t)

	if !w.GetNetwork() {
		globals.Arguments["--testnet"] = true
	} else {
		globals.Arguments["--testnet"] = false
	}

	if globals.Arguments["--wallet-address"] != nil {
		addr, err := globals.ParseValidateAddress(globals.Arguments["--wallet-address"].(string))
		if err != nil {
			//logger.Error(err, "Wallet address is invalid.")
			return
		}

		wallet_address = addr.String()
	}

	if globals.Arguments["--daemon-rpc-address"] != nil {
		daemon_rpc_address = globals.Arguments["--daemon-rpc-address"].(string)
	}

	threads = runtime.GOMAXPROCS(0)
	if globals.Arguments["--mining-threads"] != nil {
		if s, err := strconv.Atoi(globals.Arguments["--mining-threads"].(string)); err == nil {
			threads = s
		} else {
			//logger.Error(err, "Mining threads argument cannot be parsed.")
		}

		if threads > runtime.GOMAXPROCS(0) {
			//logger.Info("Mining threads is more than available CPUs. This is NOT optimal", "thread_count", threads, "max_possible", runtime.GOMAXPROCS(0))
		}
	}

	//logger.Info(fmt.Sprintf("System will mine to \"%s\" with %d threads. Good Luck!!", wallet_address, threads))

	if threads < 1 || iterations < 1 || threads > 2048 {
		panic("Invalid parameters\n")
		//return
	}

	// This tiny goroutine continuously updates status as required
	go func() {
		for nr.Mission == 1 {
			last_our_height := int64(0)
			last_best_height := int64(0)

			last_counter := uint64(0)
			last_counter_time := time.Now()
			last_mining_state := false

			_ = last_mining_state

			mining := true
			for nr.Mission == 1 {
				if nr.Mission == 0 {
					return
				}

				best_height := int64(0)
				// only update prompt if needed
				if last_our_height != our_height || last_best_height != best_height || last_counter != counter {
					mining_string := ""

					if mining {
						mining_speed := float64(counter-last_counter) / (float64(uint64(time.Since(last_counter_time))) / 1000000000.0)
						last_counter = counter
						last_counter_time = time.Now()
						switch {
						case mining_speed > 1000000:
							mining_string = fmt.Sprintf("%.3f MH/s", float32(mining_speed)/1000000.0)
						case mining_speed > 1000:
							mining_string = fmt.Sprintf("%.3f KH/s", float32(mining_speed)/1000.0)
						case mining_speed > 0:
							mining_string = fmt.Sprintf("%.0f H/s", mining_speed)
						}
					}
					last_mining_state = mining

					hash_rate_string := ""

					switch {
					case hash_rate > 1000000000000:
						hash_rate_string = fmt.Sprintf("%.3f TH/s", float64(hash_rate)/1000000000000.0)
					case hash_rate > 1000000000:
						hash_rate_string = fmt.Sprintf("%.3f GH/s", float64(hash_rate)/1000000000.0)
					case hash_rate > 1000000:
						hash_rate_string = fmt.Sprintf("%.3f MH/s", float64(hash_rate)/1000000.0)
					case hash_rate > 1000:
						hash_rate_string = fmt.Sprintf("%.3f KH/s", float64(hash_rate)/1000.0)
					case hash_rate > 0:
						hash_rate_string = fmt.Sprintf("%d H/s", hash_rate)
					}

					nr.Height = our_height
					nr.Blocks = block_counter
					nr.MiniBlocks = mini_block_counter
					nr.NWHashrate = hash_rate_string
					nr.Hashrate = mining_string
					nr.Label.Text = " " + nr.Hashrate
					nr.Label.Refresh()
					nr.LabelBlocks.Text = "Blocks:  " + strconv.Itoa(int(nr.MiniBlocks+nr.Blocks))
					nr.LabelBlocks.Refresh()

					last_our_height = our_height
					last_best_height = best_height

					mb := strconv.FormatUint(nr.MiniBlocks, 10)
					fmt.Printf("[Netrunner] Hashrate: %s -- MiniBlocks: %s\n", nr.Hashrate, mb)
				}
				time.Sleep(5 * time.Second)
			}
		}
	}()

	if threads > 255 {
		//logger.Error(nil, "This program supports maximum 256 CPU cores.", "available", threads)
		threads = 255
	}

	if node.Active == 1 {
		node.Chain.SetIntegratorAddress(engram.Disk.GetAddress())
	}

	if nr.Mission == 1 {
		go getwork(wallet_address)

		for i := 0; i < threads; i++ {
			go mineblock(i)
		}
	}

	<-Exit_In_Progress

	return
}

func random_execution(wg *sync.WaitGroup, iterations int) {
	var workbuf [255]byte

	runtime.LockOSThread()
	threadaffinity()

	rand.Read(workbuf[:])

	for i := 0; i < iterations; i++ {
		_ = astrobwtv3.AstroBWTv3(workbuf[:])
	}
	wg.Done()
	runtime.UnlockOSThread()
}

var connection_mutex sync.Mutex

func getwork(wallet_address string) {
	if nr.Mission == 0 {
		nr.Connection.Close()
		return
	}
	var err error

	for nr.Mission == 1 {
		if nr.Mission == 0 {
			break
		}
		u := url.URL{Scheme: "wss", Host: daemon_rpc_address, Path: "/ws/" + wallet_address}
		//logger.Info("connecting to ", "url", u.String())
		fmt.Printf("[Netrunner] Connecting to: %s\n", u.String())

		dialer := websocket.DefaultDialer
		dialer.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
		nr.Connection, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			fmt.Printf("[Netrunner] Error connecting to server: %s\n", daemon_rpc_address)
			//logger.Info("Will try in 10 secs", "server adress", daemon_rpc_address)
			time.Sleep(10 * time.Second)

			continue
		}

		var result rpc.GetBlockTemplate_Result

	wait_for_another_job:

		if nr.Mission == 0 {
			nr.Connection.Close()
			break
		}

		if err = nr.Connection.ReadJSON(&result); err != nil {
			fmt.Printf("[Netrunner] Error connecting to server: %s\n", daemon_rpc_address)
			continue
		}

		mutex.Lock()
		job = result
		job_counter++
		mutex.Unlock()
		if job.LastError != "" {
			//logger.Error(nil, "received error", "err", job.LastError)
		}

		block_counter = job.Blocks
		mini_block_counter = job.MiniBlocks
		hash_rate = job.Difficultyuint64
		our_height = int64(job.Height)
		Difficulty = job.Difficultyuint64

		//fmt.Printf("recv: %+v diff %d\n", result, Difficulty)
		goto wait_for_another_job
	}
}

func mineblock(tid int) {
	var diff big.Int
	var work [block.MINIBLOCK_SIZE]byte
	var random_buf [12]byte

	rand.Read(random_buf[:])

	time.Sleep(5 * time.Second)

	nonce_buf := work[block.MINIBLOCK_SIZE-5:] //since slices are linked, it modifies parent
	runtime.LockOSThread()
	threadaffinity()

	var local_job_counter int64

	i := uint32(0)

	for nr.Mission == 1 {
		if nr.Mission == 0 {
			break
		}
		mutex.RLock()
		myjob := job
		local_job_counter = job_counter
		mutex.RUnlock()

		n, err := hex.Decode(work[:], []byte(myjob.Blockhashing_blob))
		if err != nil || n != block.MINIBLOCK_SIZE {
			//logger.Error(err, "Blockwork could not decoded successfully", "blockwork", myjob.Blockhashing_blob, "n", n, "job", myjob)
			time.Sleep(time.Second)
			continue
		}

		copy(work[block.MINIBLOCK_SIZE-12:], random_buf[:]) // add more randomization in the mix
		work[block.MINIBLOCK_SIZE-1] = byte(tid)

		diff.SetString(myjob.Difficulty, 10)

		if work[0]&0xf != 1 { // check version
			//logger.Error(nil, "Unknown version, please check for updates", "version", work[0]&0x1f)
			time.Sleep(time.Second)
			continue
		}

		for local_job_counter == job_counter && nr.Mission == 1 { // update job when it comes, expected rate 1 per second
			i++
			binary.BigEndian.PutUint32(nonce_buf, i)

			powhash := astrobwtv3.AstroBWTv3(work[:])
			atomic.AddUint64(&counter, 1)

			if CheckPowHashBig(powhash, &diff) == true { // note we are doing a local, NW might have moved meanwhile
				//logger.V(1).Info("Successfully found DERO miniblock (going to submit)", "difficulty", myjob.Difficulty, "height", myjob.Height)
				func() {
					defer globals.Recover(1)
					connection_mutex.Lock()
					defer connection_mutex.Unlock()
					nr.Connection.WriteJSON(rpc.SubmitBlock_Params{JobID: myjob.JobID, MiniBlockhashing_blob: fmt.Sprintf("%x", work[:])})
					fmt.Printf("[Netrunner] Saved on datashard: %s - %s\n", strconv.FormatUint(myjob.Height, 10), myjob.Blockhashing_blob)
					nr.ScrollBox.Refresh()
					nr.Data.Append(strconv.FormatUint(myjob.Height, 10) + "," + myjob.Blockhashing_blob)
				}()

			}
		}
	}
}
