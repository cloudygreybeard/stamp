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

// Package strftime formats time.Time values using C strftime directives.
package strftime

import (
	"fmt"
	"strings"
	"time"
)

// Format returns a string representation of t according to the strftime layout.
// Unrecognised directives are passed through verbatim (e.g. %Q becomes %Q).
func Format(layout string, t time.Time) string {
	var b strings.Builder
	b.Grow(len(layout) * 2)

	for i := 0; i < len(layout); i++ {
		if layout[i] != '%' {
			b.WriteByte(layout[i])
			continue
		}
		i++
		if i >= len(layout) {
			b.WriteByte('%')
			break
		}
		switch layout[i] {
		case 'Y':
			fmt.Fprintf(&b, "%04d", t.Year())
		case 'y':
			fmt.Fprintf(&b, "%02d", t.Year()%100)
		case 'C':
			fmt.Fprintf(&b, "%02d", t.Year()/100)
		case 'm':
			fmt.Fprintf(&b, "%02d", t.Month())
		case 'd':
			fmt.Fprintf(&b, "%02d", t.Day())
		case 'e':
			fmt.Fprintf(&b, "%2d", t.Day())
		case 'H':
			fmt.Fprintf(&b, "%02d", t.Hour())
		case 'I':
			h := t.Hour() % 12
			if h == 0 {
				h = 12
			}
			fmt.Fprintf(&b, "%02d", h)
		case 'M':
			fmt.Fprintf(&b, "%02d", t.Minute())
		case 'S':
			fmt.Fprintf(&b, "%02d", t.Second())
		case 'p':
			if t.Hour() < 12 {
				b.WriteString("AM")
			} else {
				b.WriteString("PM")
			}
		case 'P':
			if t.Hour() < 12 {
				b.WriteString("am")
			} else {
				b.WriteString("pm")
			}
		case 'z':
			_, offset := t.Zone()
			sign := byte('+')
			if offset < 0 {
				sign = '-'
				offset = -offset
			}
			fmt.Fprintf(&b, "%c%02d%02d", sign, offset/3600, (offset%3600)/60)
		case 'Z':
			name, _ := t.Zone()
			b.WriteString(name)
		case 'a':
			b.WriteString(t.Weekday().String()[:3])
		case 'A':
			b.WriteString(t.Weekday().String())
		case 'b', 'h':
			b.WriteString(t.Month().String()[:3])
		case 'B':
			b.WriteString(t.Month().String())
		case 'j':
			fmt.Fprintf(&b, "%03d", t.YearDay())
		case 'u':
			d := int(t.Weekday())
			if d == 0 {
				d = 7
			}
			fmt.Fprintf(&b, "%d", d)
		case 'w':
			fmt.Fprintf(&b, "%d", t.Weekday())
		case 'G':
			y, _ := t.ISOWeek()
			fmt.Fprintf(&b, "%04d", y)
		case 'V':
			_, w := t.ISOWeek()
			fmt.Fprintf(&b, "%02d", w)
		case 'F':
			fmt.Fprintf(&b, "%04d-%02d-%02d", t.Year(), t.Month(), t.Day())
		case 'T':
			fmt.Fprintf(&b, "%02d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
		case 'R':
			fmt.Fprintf(&b, "%02d:%02d", t.Hour(), t.Minute())
		case 's':
			fmt.Fprintf(&b, "%d", t.Unix())
		case 'N':
			fmt.Fprintf(&b, "%09d", t.Nanosecond())
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case '%':
			b.WriteByte('%')
		default:
			b.WriteByte('%')
			b.WriteByte(layout[i])
		}
	}
	return b.String()
}
