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

package strftime

import (
	"fmt"
	"testing"
	"time"
)

func TestFormat(t *testing.T) {
	// 2026-04-26 07:32:00 +10:00 (AEST, a Sunday)
	loc := time.FixedZone("AEST", 10*60*60)
	ts := time.Date(2026, 4, 26, 7, 32, 0, 123456789, loc)

	tests := []struct {
		name   string
		layout string
		want   string
	}{
		{"full year", "%Y", "2026"},
		{"short year", "%y", "26"},
		{"century", "%C", "20"},
		{"month", "%m", "04"},
		{"day", "%d", "26"},
		{"day space-padded", "%e", "26"},
		{"hour 24h", "%H", "07"},
		{"hour 12h", "%I", "07"},
		{"minute", "%M", "32"},
		{"second", "%S", "00"},
		{"AM/PM upper", "%p", "AM"},
		{"am/pm lower", "%P", "am"},
		{"timezone offset", "%z", "+1000"},
		{"timezone name", "%Z", "AEST"},
		{"weekday short", "%a", "Sun"},
		{"weekday full", "%A", "Sunday"},
		{"month short", "%b", "Apr"},
		{"month full", "%B", "April"},
		{"month short h", "%h", "Apr"},
		{"day of year", "%j", "116"},
		{"weekday iso", "%u", "7"},
		{"weekday sunday", "%w", "0"},
		{"iso week year", "%G", "2026"},
		{"iso week number", "%V", "17"},
		{"date shorthand", "%F", "2026-04-26"},
		{"time shorthand", "%T", "07:32:00"},
		{"hours:minutes", "%R", "07:32"},
		{"nanoseconds", "%N", "123456789"},
		{"literal percent", "%%", "%"},
		{"newline", "%n", "\n"},
		{"tab", "%t", "\t"},
		{"iso 8601", "%Y-%m-%dT%H:%M:%S%z", "2026-04-26T07:32:00+1000"},
		{"compact date", "%Y%m%d", "20260426"},
		{"unknown directive", "%Q", "%Q"},
		{"trailing percent", "test%", "test%"},
		{"no directives", "literal", "literal"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Format(tt.layout, ts)
			if got != tt.want {
				t.Errorf("Format(%q) = %q, want %q", tt.layout, got, tt.want)
			}
		})
	}
}

func TestFormat_UnixTimestamp(t *testing.T) {
	ts := time.Date(2026, 4, 26, 7, 32, 0, 0, time.UTC)
	got := Format("%s", ts)
	want := fmt.Sprintf("%d", ts.Unix())
	if got != want {
		t.Errorf("Format(%%s) = %q, want %q", got, want)
	}
}

func TestFormat_12HourEdgeCases(t *testing.T) {
	loc := time.UTC

	midnight := time.Date(2026, 1, 1, 0, 0, 0, 0, loc)
	if got := Format("%I %p", midnight); got != "12 AM" {
		t.Errorf("midnight: got %q, want %q", got, "12 AM")
	}

	noon := time.Date(2026, 1, 1, 12, 0, 0, 0, loc)
	if got := Format("%I %p", noon); got != "12 PM" {
		t.Errorf("noon: got %q, want %q", got, "12 PM")
	}
}
