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
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Get app path
func AppPath() (result string) {
	result, _ = os.Getwd()
	if runtime.GOOS == "android" {
		result = a.Storage().RootURI().Path()
	} else if runtime.GOOS == "ios" {
		result = a.Storage().RootURI().Path()
	}

	return
}

func GetAccounts() (result []string, err error) {
	path := string_

	if !session.Testnet {
		_, err = os.Stat(
			filepath.Join(
				AppPath(),
				stringmainnet,
			))
		if err != nil {
			return
		} else {
			path = filepath.Join(AppPath(), stringmainnet) + string(filepath.Separator)
		}
	} else {
		_, err = os.Stat(
			filepath.Join(
				AppPath(),
				stringtestnet,
			),
		)
		if err != nil {
			return
		} else {
			path = filepath.Join(AppPath(), stringtestnet) + string(filepath.Separator)
		}
	}

	matches, _ := filepath.Glob(path + "*.db")
	result = []string{}

	for _, match := range matches {
		check, _ := os.Stat(match)
		if !check.IsDir() {
			if strings.Contains(match, ".db") {
				split := strings.Split(match, string(
					filepath.Separator,
				))
				pos := len(split) - 1
				match = split[pos]
				result = append(result, match)
			}
		}
	}

	if len(result) == 0 {
		// TODO: May do something here like start the user at Create/Restore Account window.
	}

	return
}

func findAccount() (result bool) {
	matches, err := filepath.Glob(session.Path)
	if err != nil {
		fmt.Printf("%s\n", err)
	}

	if len(matches) > 0 {
		result = true
	} else {
		result = false
	}

	return
}

func checkDir() (err error) {
	err = os.MkdirAll(
		filepath.Join(
			AppPath(),
			stringmainnet,
		),
		os.ModePerm,
	)
	if err != nil {
		return
	}
	err = os.MkdirAll(
		filepath.Join(
			AppPath(),
			stringtestnet,
		),
		os.ModePerm,
	)
	if err != nil {
		return
	}
	err = os.MkdirAll(
		filepath.Join(
			AppPath(),
			stringdatashards,
		),
		os.ModePerm,
	)
	if err != nil {
		return
	}

	return
}
