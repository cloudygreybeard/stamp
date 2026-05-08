// Copyright 2026
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux

package stamper

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

func accessTime(info os.FileInfo) (time.Time, error) {
	stat := info.Sys().(*syscall.Stat_t)
	return time.Unix(stat.Atim.Sec, stat.Atim.Nsec), nil
}

func birthTime(_ os.FileInfo) (time.Time, error) {
	return time.Time{}, fmt.Errorf("birth time is not reliably available on Linux; use --source mtime, atime, or now")
}
