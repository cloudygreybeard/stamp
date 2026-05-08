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

// Package stamper renames or copies files, inserting a formatted timestamp
// into the filename.
package stamper

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/cloudygreybeard/stamp/internal/strftime"
)

// Source identifies where to obtain the timestamp for a file.
type Source string

const (
	SourceMtime Source = "mtime"
	SourceAtime Source = "atime"
	SourceBtime Source = "btime"
	SourceNow   Source = "now"
)

// Options controls how files are stamped.
type Options struct {
	Format    string
	Source    Source
	Copy      bool
	DryRun    bool
	Separator string
	Template  string         // shortcut name (before-ext, prefix, suffix) or token pattern
	Replace   *regexp.Regexp // regex matched against basename; overrides Template
}

// Result describes the outcome of stamping a single file.
type Result struct {
	Original string
	Stamped  string
}

// expandTemplate resolves a named shortcut to a token pattern.
// Recognised shortcuts: before-ext (default), prefix, suffix.
// Any other value (including patterns containing "{") is returned as-is.
func expandTemplate(tmpl string) string {
	switch tmpl {
	case "prefix":
		return "{stamp}{sep}{name}{ext}"
	case "suffix":
		return "{name}{ext}{sep}{stamp}"
	case "before-ext", "":
		return "{name}{sep}{stamp}{ext}"
	default:
		return tmpl
	}
}

// NewName computes the stamped filename. Replace takes priority when its
// regex matches the basename. Otherwise Template is expanded — either from
// a named shortcut (before-ext, prefix, suffix) or a token pattern using
// {name}, {ext}, {stamp}, and {sep} placeholders.
func NewName(path, timestamp string, opts Options) string {
	dir := filepath.Dir(path)
	ext := filepath.Ext(path)
	base := filepath.Base(path)
	name := base[:len(base)-len(ext)]

	if name == "" {
		name = base
		ext = ""
	}

	sep := opts.Separator
	if sep == "" {
		sep = "."
	}

	if opts.Replace != nil && opts.Replace.MatchString(base) {
		return filepath.Join(dir, opts.Replace.ReplaceAllString(base, timestamp))
	}

	r := strings.NewReplacer(
		"{name}", name,
		"{ext}", ext,
		"{stamp}", timestamp,
		"{sep}", sep,
	)
	return filepath.Join(dir, r.Replace(expandTemplate(opts.Template)))
}

func resolveTimestamp(path string, opts Options) (string, error) {
	switch opts.Source {
	case SourceMtime, SourceAtime, SourceBtime:
		return timestampFromStat(path, opts)
	case SourceNow:
		return strftime.Format(opts.Format, time.Now()), nil
	default:
		return string(opts.Source), nil
	}
}

func timestampFromStat(path string, opts Options) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", path, err)
	}

	var t time.Time
	switch opts.Source {
	case SourceMtime:
		t = info.ModTime()
	case SourceAtime:
		at, err := accessTime(info)
		if err != nil {
			return "", fmt.Errorf("access time for %s: %w", path, err)
		}
		t = at
	case SourceBtime:
		bt, err := birthTime(info)
		if err != nil {
			return "", fmt.Errorf("birth time for %s: %w", path, err)
		}
		t = bt
	}

	return strftime.Format(opts.Format, t), nil
}

// Process stamps a single file and returns the result.
func Process(path string, opts Options) (Result, error) {
	ts, err := resolveTimestamp(path, opts)
	if err != nil {
		return Result{}, err
	}

	newPath := NewName(path, ts, opts)
	res := Result{Original: path, Stamped: newPath}

	if opts.DryRun {
		return res, nil
	}

	if _, err := os.Stat(newPath); err == nil {
		return Result{}, fmt.Errorf("destination already exists: %s", newPath)
	}

	if opts.Copy {
		if err := copyFile(path, newPath); err != nil {
			return Result{}, fmt.Errorf("copy %s to %s: %w", path, newPath, err)
		}
	} else {
		if err := os.Rename(path, newPath); err != nil {
			return Result{}, fmt.Errorf("rename %s to %s: %w", path, newPath, err)
		}
	}

	return res, nil
}

func copyFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()

	info, err := s.Stat()
	if err != nil {
		return err
	}

	d, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_EXCL, info.Mode())
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(d, s)
	closeErr := d.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
