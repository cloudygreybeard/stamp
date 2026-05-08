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

package stamper

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestExpandTemplate(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "{name}{sep}{stamp}{ext}"},
		{"before-ext", "{name}{sep}{stamp}{ext}"},
		{"prefix", "{stamp}{sep}{name}{ext}"},
		{"suffix", "{name}{ext}{sep}{stamp}"},
		{"{stamp}_{name}{ext}", "{stamp}_{name}{ext}"},
		{"custom-value", "custom-value"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := expandTemplate(tt.input)
			if got != tt.want {
				t.Errorf("expandTemplate(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNewName_TemplateShortcuts(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		timestamp string
		opts      Options
		want      string
	}{
		// before-ext (default when Template is empty)
		{
			name:      "before-ext simple",
			path:      "report.md",
			timestamp: "2026-04-26T07:32:00+1000",
			want:      "report.2026-04-26T07:32:00+1000.md",
		},
		{
			name:      "before-ext with directory",
			path:      "./this_file.md",
			timestamp: "2026-04-26T07:32:00+1000",
			want:      "this_file.2026-04-26T07:32:00+1000.md",
		},
		{
			name:      "before-ext nested path",
			path:      "/home/user/docs/notes.txt",
			timestamp: "20260426",
			want:      filepath.Join("/home/user/docs", "notes.20260426.txt"),
		},
		{
			name:      "before-ext no extension",
			path:      "Makefile",
			timestamp: "2026-04-26",
			want:      "Makefile.2026-04-26",
		},
		{
			name:      "before-ext multiple dots",
			path:      "archive.tar.gz",
			timestamp: "20260426",
			want:      "archive.tar.20260426.gz",
		},
		{
			name:      "before-ext hidden file with extension",
			path:      ".config.yaml",
			timestamp: "v2",
			want:      ".config.v2.yaml",
		},
		{
			name:      "before-ext dotfile",
			path:      ".gitignore",
			timestamp: "bak",
			want:      ".gitignore.bak",
		},
		{
			name:      "before-ext explicit shortcut",
			path:      "report.md",
			timestamp: "20260426",
			opts:      Options{Template: "before-ext"},
			want:      "report.20260426.md",
		},

		// suffix
		{
			name:      "suffix simple",
			path:      "report.md",
			timestamp: "2026-04-26",
			opts:      Options{Template: "suffix"},
			want:      "report.md.2026-04-26",
		},
		{
			name:      "suffix no extension",
			path:      "Makefile",
			timestamp: "bak",
			opts:      Options{Template: "suffix"},
			want:      "Makefile.bak",
		},
		{
			name:      "suffix with directory",
			path:      "/tmp/data.csv",
			timestamp: "20260426",
			opts:      Options{Template: "suffix"},
			want:      filepath.Join("/tmp", "data.csv.20260426"),
		},

		// prefix
		{
			name:      "prefix simple",
			path:      "report.md",
			timestamp: "2026-04-26",
			opts:      Options{Template: "prefix"},
			want:      "2026-04-26.report.md",
		},
		{
			name:      "prefix with directory",
			path:      "/tmp/photo.jpg",
			timestamp: "20260426",
			opts:      Options{Template: "prefix"},
			want:      filepath.Join("/tmp", "20260426.photo.jpg"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewName(tt.path, tt.timestamp, tt.opts)
			if got != tt.want {
				t.Errorf("NewName(%q, %q) = %q, want %q", tt.path, tt.timestamp, got, tt.want)
			}
		})
	}
}

func TestNewName_Separator(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		timestamp string
		opts      Options
		want      string
	}{
		{
			name:      "underscore separator before-ext",
			path:      "report.md",
			timestamp: "20260426",
			opts:      Options{Separator: "_"},
			want:      "report_20260426.md",
		},
		{
			name:      "hyphen separator before-ext",
			path:      "report.md",
			timestamp: "20260426",
			opts:      Options{Separator: "-"},
			want:      "report-20260426.md",
		},
		{
			name:      "underscore separator prefix",
			path:      "photo.jpg",
			timestamp: "20260426",
			opts:      Options{Template: "prefix", Separator: "_"},
			want:      "20260426_photo.jpg",
		},
		{
			name:      "underscore separator suffix",
			path:      "data.csv",
			timestamp: "bak",
			opts:      Options{Template: "suffix", Separator: "_"},
			want:      "data.csv_bak",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewName(tt.path, tt.timestamp, tt.opts)
			if got != tt.want {
				t.Errorf("NewName(%q, %q) = %q, want %q", tt.path, tt.timestamp, got, tt.want)
			}
		})
	}
}

func TestNewName_Template(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		timestamp string
		opts      Options
		want      string
	}{
		{
			name:      "default-equivalent template",
			path:      "report.md",
			timestamp: "20260426",
			opts:      Options{Template: "{name}.{stamp}{ext}"},
			want:      "report.20260426.md",
		},
		{
			name:      "prefix template",
			path:      "report.md",
			timestamp: "20260426",
			opts:      Options{Template: "{stamp}_{name}{ext}"},
			want:      "20260426_report.md",
		},
		{
			name:      "suffix template",
			path:      "report.md",
			timestamp: "20260426",
			opts:      Options{Template: "{name}{ext}.{stamp}"},
			want:      "report.md.20260426",
		},
		{
			name:      "custom layout",
			path:      "IMG_1234.jpg",
			timestamp: "2026-04-26",
			opts:      Options{Template: "{stamp} - {name}{ext}"},
			want:      "2026-04-26 - IMG_1234.jpg",
		},
		{
			name:      "sep token in template",
			path:      "report.md",
			timestamp: "20260426",
			opts:      Options{Template: "{name}{sep}{stamp}{ext}", Separator: "_"},
			want:      "report_20260426.md",
		},
		{
			name:      "template with directory preserved",
			path:      "/home/user/report.md",
			timestamp: "20260426",
			opts:      Options{Template: "{stamp}_{name}{ext}"},
			want:      filepath.Join("/home/user", "20260426_report.md"),
		},
		{
			name:      "token pattern overrides default shortcut",
			path:      "report.md",
			timestamp: "20260426",
			opts:      Options{Template: "{name}.{stamp}{ext}"},
			want:      "report.20260426.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewName(tt.path, tt.timestamp, tt.opts)
			if got != tt.want {
				t.Errorf("NewName(%q, %q) = %q, want %q", tt.path, tt.timestamp, got, tt.want)
			}
		})
	}
}

func TestNewName_Replace(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		timestamp string
		opts      Options
		want      string
	}{
		{
			name:      "replace existing date",
			path:      "report.2024-01-15.md",
			timestamp: "2026-04-26",
			opts:      Options{Replace: regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)},
			want:      "report.2026-04-26.md",
		},
		{
			name:      "replace compact date",
			path:      "backup_20240115_data.csv",
			timestamp: "20260426",
			opts:      Options{Replace: regexp.MustCompile(`\d{8}`)},
			want:      "backup_20260426_data.csv",
		},
		{
			name:      "replace version tag",
			path:      "config.v1.yaml",
			timestamp: "v2",
			opts:      Options{Replace: regexp.MustCompile(`v\d+`)},
			want:      "config.v2.yaml",
		},
		{
			name:      "replace with directory",
			path:      "/tmp/report.2024-01-15.md",
			timestamp: "2026-04-26",
			opts:      Options{Replace: regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)},
			want:      filepath.Join("/tmp", "report.2026-04-26.md"),
		},
		{
			name:      "no match falls through to template",
			path:      "report.md",
			timestamp: "20260426",
			opts:      Options{Replace: regexp.MustCompile(`\d{8}`), Template: "prefix"},
			want:      "20260426.report.md",
		},
		{
			name:      "replace overrides template",
			path:      "report.OLD.md",
			timestamp: "NEW",
			opts: Options{
				Replace:  regexp.MustCompile(`OLD`),
				Template: "{stamp}_{name}{ext}",
			},
			want: "report.NEW.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewName(tt.path, tt.timestamp, tt.opts)
			if got != tt.want {
				t.Errorf("NewName(%q, %q) = %q, want %q", tt.path, tt.timestamp, got, tt.want)
			}
		})
	}
}

func TestProcess_Rename(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(src, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := Options{
		Format: "%Y",
		Source: "literal-ts",
	}

	res, err := Process(src, opts)
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(dir, "test.literal-ts.txt")
	if res.Stamped != want {
		t.Errorf("stamped = %q, want %q", res.Stamped, want)
	}

	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("original file should not exist after rename")
	}
	if _, err := os.Stat(want); err != nil {
		t.Error("stamped file should exist after rename")
	}
}

func TestProcess_Copy(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "test.txt")
	content := []byte("hello world")
	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatal(err)
	}

	opts := Options{
		Format: "%Y",
		Source: "copied",
		Copy:   true,
	}

	res, err := Process(src, opts)
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(dir, "test.copied.txt")
	if res.Stamped != want {
		t.Errorf("stamped = %q, want %q", res.Stamped, want)
	}

	if _, err := os.Stat(src); err != nil {
		t.Error("original file should still exist after copy")
	}

	got, err := os.ReadFile(want)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Errorf("copied content = %q, want %q", got, content)
	}
}

func TestProcess_DryRun(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(src, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := Options{
		Source: "dry",
		DryRun: true,
	}

	res, err := Process(src, opts)
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(dir, "test.dry.txt")
	if res.Stamped != want {
		t.Errorf("stamped = %q, want %q", res.Stamped, want)
	}

	if _, err := os.Stat(src); err != nil {
		t.Error("original file should still exist in dry-run mode")
	}
	if _, err := os.Stat(want); !os.IsNotExist(err) {
		t.Error("stamped file should not exist in dry-run mode")
	}
}

func TestProcess_DestinationExists(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "test.txt")
	dst := filepath.Join(dir, "test.exists.txt")

	if err := os.WriteFile(src, []byte("src"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("dst"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := Options{Source: "exists"}
	_, err := Process(src, opts)
	if err == nil {
		t.Fatal("expected error when destination exists")
	}
}

func TestProcess_MtimeSource(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(src, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := Options{
		Format: "%Y%m%d",
		Source: SourceMtime,
		DryRun: true,
	}

	res, err := Process(src, opts)
	if err != nil {
		t.Fatal(err)
	}

	if res.Stamped == "" {
		t.Error("expected non-empty stamped path")
	}
	if res.Original != src {
		t.Errorf("original = %q, want %q", res.Original, src)
	}
}

func TestProcess_NowSource(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(src, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := Options{
		Format: "%Y",
		Source: SourceNow,
		DryRun: true,
	}

	res, err := Process(src, opts)
	if err != nil {
		t.Fatal(err)
	}

	if res.Stamped == "" {
		t.Error("expected non-empty stamped path")
	}
}

func TestProcess_NonexistentFile(t *testing.T) {
	opts := Options{
		Source: SourceMtime,
		Format: "%Y",
	}
	_, err := Process("/nonexistent/file.txt", opts)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestProcess_WithTemplateShortcut(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(src, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := Options{
		Source:   "ts",
		Template: "prefix",
		DryRun:   true,
	}

	res, err := Process(src, opts)
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(dir, "ts.test.txt")
	if res.Stamped != want {
		t.Errorf("stamped = %q, want %q", res.Stamped, want)
	}
}
