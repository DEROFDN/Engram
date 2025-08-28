// Copyright 2023-2024 DERO Foundation. All rights reserved.
//
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
	"os"
	"path/filepath"
	"runtime"

	"github.com/deroproject/graviton"
)

// Get Engram's working directory
func GetDir() (result string, err error) {
	result, err = os.Getwd()
	if err != nil {
		return
	}

	if runtime.GOOS == "darwin" {
		switch session.Network {
		case NETWORK_MAINNET:
			result = filepath.Join(AppPath(), "Contents", "Resources", "mainnet") + string(filepath.Separator)
		case NETWORK_SIMULATOR:
			result = filepath.Join(AppPath(), "Contents", "Resources", "testnet_simulator") + string(filepath.Separator)
		default:
			result = filepath.Join(AppPath(), "Contents", "Resources", "testnet") + string(filepath.Separator)
		}
	} else if runtime.GOOS == "android" {
		switch session.Network {
		case NETWORK_MAINNET:
			result = filepath.Join(AppPath(), "mainnet") + string(filepath.Separator)
		case NETWORK_SIMULATOR:
			result = filepath.Join(AppPath(), "testnet_simulator") + string(filepath.Separator)
		default:
			result = filepath.Join(AppPath(), "testnet") + string(filepath.Separator)
		}
	} else if runtime.GOOS == "ios" {
		switch session.Network {
		case NETWORK_MAINNET:
			result = filepath.Join(AppPath(), "mainnet") + string(filepath.Separator)
		case NETWORK_SIMULATOR:
			result = filepath.Join(AppPath(), "testnet_simulator") + string(filepath.Separator)
		default:
			result = filepath.Join(AppPath(), "testnet") + string(filepath.Separator)
		}
	} else {
		switch session.Network {
		case NETWORK_MAINNET:
			result = filepath.Join(AppPath(), "mainnet") + string(filepath.Separator)
		case NETWORK_SIMULATOR:
			result = filepath.Join(AppPath(), "testnet_simulator") + string(filepath.Separator)
		default:
			result = filepath.Join(AppPath(), "testnet") + string(filepath.Separator)
		}
	}

	return
}

// Get a datashard's path
func GetShard() (result string, err error) {
	if engram.Disk == nil {
		result = filepath.Join(AppPath(), "datashards", "settings")
		return
	} else {
		address := engram.Disk.GetAddress().String()
		result = filepath.Join(AppPath(), "datashards", fmt.Sprintf("%x", sha1.Sum([]byte(address))))
		return
	}
}

// Encrypt a key-value and then store it in a Graviton tree
// Requires the user to have an active wallet open
func StoreEncryptedValue(t string, key []byte, value []byte) (err error) {
	if engram.Disk == nil {
		err = errors.New("error: no active account found")
		return
	}

	if t == "" {
		err = errors.New("error: missing graviton tree input")
		return
	} else if key == nil {
		err = errors.New("error: missing graviton key input")
		return
	}

	eValue, err := engram.Disk.Encrypt(value)
	if err != nil {
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

	tree, err := ss.GetTree(t)
	if err != nil {
		return
	}

	err = tree.Put(key, eValue)
	if err != nil {
		return
	}

	_, err = graviton.Commit(tree)
	if err != nil {
		return
	}

	return
}

// Store a key-value in a Graviton tree
func StoreValue(t string, key []byte, value []byte) (err error) {
	if t == "" {
		err = errors.New("error: missing graviton tree input")
		return
	} else if key == nil {
		err = errors.New("error: missing graviton key input")
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

	tree, err := ss.GetTree(t)
	if err != nil {
		return
	}

	err = tree.Put(key, value)
	if err != nil {
		return
	}

	_, err = graviton.Commit(tree)
	if err != nil {
		return
	}

	return
}

// Get a key-value from a Graviton tree
func GetValue(t string, key []byte) (result []byte, err error) {
	result = []byte("")

	if t == "" {
		err = errors.New("error: missing graviton tree input")
		return
	} else if key == nil {
		err = errors.New("error: missing graviton key input")
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

	tree, err := ss.GetTree(t)
	if err != nil {
		return
	}

	result, err = tree.Get(key)
	if err != nil {
		return
	}

	return
}

// Get an encrypted key-value from a Graviton tree and then decrypt it
// Requires the user to have an active wallet open
func GetEncryptedValue(t string, key []byte) (result []byte, err error) {
	result = []byte("")

	if t == "" {
		err = errors.New("error: missing graviton tree input")
		return
	} else if key == nil {
		err = errors.New("error: missing graviton key input")
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

	tree, err := ss.GetTree(t)
	if err != nil {
		return
	}

	eValue, err := tree.Get(key)
	if err != nil {
		return
	}

	result, err = engram.Disk.Decrypt(eValue)
	if err != nil {
		return
	}

	return
}

// Delete a key-value in a Graviton tree
func DeleteKey(t string, key []byte) (err error) {
	if t == "" {
		err = errors.New("error: missing graviton tree input")
		return
	} else if key == nil {
		err = errors.New("error: missing graviton key input")
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

	tree, err := ss.GetTree(t)
	if err != nil {
		return
	}

	err = tree.Delete(key)
	if err != nil {
		return
	}

	_, err = graviton.Commit(tree)
	if err != nil {
		return
	}

	return
}
