//	Copyright 2021-2022 DERO Foundation. All rights reserved.
//	Mikoshi v0.0.1
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
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"github.com/deroproject/graviton"
	"golang.org/x/crypto/chacha20poly1305"
)

// Get Engram's working directory
func GetDir() (result string, err error) {
	result, err = os.Getwd()
	if err != nil {
		return
	}

	if !session.Network {
		result += string(filepath.Separator) + "testnet" + string(filepath.Separator)
	} else {
		result += string(filepath.Separator) + "mainnet" + string(filepath.Separator)
	}

	return
}

// Get a datashard's path
func GetShard() (result string, err error) {
	result = ""
	wd, err := os.Getwd()

	if engram.Disk == nil {
		//if session.Domain == "app.settings" {
		result = wd + "/datashards/settings/"
		//}

		return
	} else {
		address := engram.Disk.GetAddress().String()
		result = wd + "/datashards/" + fmt.Sprintf("%x", sha1.Sum([]byte(address)))

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

	eValue, err := Encrypt(value)
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

	eValue, _ := tree.Get(key)
	if err != nil {
		return
	}

	result, err = Decrypt(eValue)
	if err != nil {
		return
	}

	return
}

// Use a key and nonce to seal the data
func EncryptWithKey(Key []byte, Data []byte) (result []byte, err error) {
	nonce := make([]byte, chacha20poly1305.NonceSize, chacha20poly1305.NonceSize)
	cipher, err := chacha20poly1305.New(Key)
	if err != nil {
		return
	}

	_, err = rand.Read(nonce)
	if err != nil {
		return
	}
	Data = cipher.Seal(Data[:0], nonce, Data, nil)

	result = append(Data, nonce...)
	return
}

// Use a key and extract 12 byte nonce from the data and unseal the data
func DecryptWithKey(Key []byte, Data []byte) (result []byte, err error) {

	// make sure data is atleast 28 byte, 16 bytes of AEAD cipher and 12 bytes of nonce
	if len(Data) < 28 {
		err = errors.New("error: invalid data")
		return
	}

	data_without_nonce := Data[0 : len(Data)-chacha20poly1305.NonceSize]

	nonce := Data[len(Data)-chacha20poly1305.NonceSize:]

	cipher, err := chacha20poly1305.New(Key)
	if err != nil {
		return
	}

	return cipher.Open(result[:0], nonce, data_without_nonce, nil)

}

// Encrypt data using the wallet's public key
// Trim the public key to 32 bytes
func Encrypt(Data []byte) (result []byte, err error) {
	keys := engram.Disk.Get_Keys()
	key := []byte(keys.Public.StringHex())
	key = key[:len(key)-34]

	return EncryptWithKey(key, Data)
}

// Decrypt data using the wallet's public key
// Trim the public key to 32 bytes
func Decrypt(Data []byte) (result []byte, err error) {
	keys := engram.Disk.Get_Keys()
	key := []byte(keys.Public.StringHex())
	key = key[:len(key)-34]

	return DecryptWithKey(key, Data)
}

func NewRandomByte(n int) (char []byte) {
	const pool = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"
	char = make([]byte, n)

	for i := 0; i < n; i++ {
		selection, _ := rand.Int(rand.Reader, big.NewInt(int64(len(pool))))

		char[i] = pool[selection.Int64()]
	}

	return
}

// Generate a 32 byte key from a wallet public key
func newSequenceKey() (key []byte, sequence []byte) {
	keys := engram.Disk.Get_Keys()
	key = []byte(keys.Public.StringHex())

	for i := len(key); i != 32; {
		if len(key) < 32 {
			key = append(key, "@"...)
			sequence = append(sequence, "@"...)
		} else {
			char := NewRandomByte(1)
			sequence = append(sequence, char...)
			key = bytes.Trim(key, string(char[0]))
		}
	}

	return
}

// Use a given sequence to decode the key
func resequence(sequence []byte) (key []byte) {
	keys := engram.Disk.Get_Keys()
	key = []byte(keys.Public.StringHex())

	for i := 0; i < len(sequence); i++ {
		key = bytes.Trim(key, string(sequence[i]))
	}

	return
}
