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

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	original := &Config{
		Default: "compact",
		Presets: []Preset{
			{Name: "compact", Format: "%Y%m%d"},
			{Name: "backup", Format: "%Y-%m-%dT%H:%M:%S%z", Template: "suffix", Copy: true},
		},
	}

	if err := Save(path, original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, migrated, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if migrated {
		t.Error("expected migrated=false for new-format file")
	}
	if loaded.Default != original.Default {
		t.Errorf("Default = %q, want %q", loaded.Default, original.Default)
	}
	if len(loaded.Presets) != len(original.Presets) {
		t.Fatalf("len(Presets) = %d, want %d", len(loaded.Presets), len(original.Presets))
	}
	for i, want := range original.Presets {
		got := loaded.Presets[i]
		if got.Name != want.Name || got.Format != want.Format {
			t.Errorf("Presets[%d] = %+v, want %+v", i, got, want)
		}
	}
}

func TestLoadNonexistent(t *testing.T) {
	cfg, migrated, err := Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if migrated {
		t.Error("expected migrated=false")
	}
	if len(cfg.Presets) != 0 {
		t.Errorf("expected empty presets, got %d", len(cfg.Presets))
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(":::invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	_, _, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestFindPreset(t *testing.T) {
	cfg := &Config{
		Presets: []Preset{
			{Name: "a", Format: "%Y"},
			{Name: "b", Format: "%Y%m%d"},
		},
	}

	p, ok := cfg.FindPreset("b")
	if !ok {
		t.Fatal("expected to find preset b")
	}
	if p.Format != "%Y%m%d" {
		t.Errorf("Format = %q, want %%Y%%m%%d", p.Format)
	}

	_, ok = cfg.FindPreset("missing")
	if ok {
		t.Error("expected not to find preset missing")
	}
}

func TestPresetNames(t *testing.T) {
	cfg := &Config{
		Presets: []Preset{
			{Name: "zebra"},
			{Name: "alpha"},
			{Name: "middle"},
		},
	}
	names := cfg.PresetNames()
	want := []string{"alpha", "middle", "zebra"}
	if len(names) != len(want) {
		t.Fatalf("len = %d, want %d", len(names), len(want))
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("names[%d] = %q, want %q", i, names[i], want[i])
		}
	}
}

func TestCreatePreset(t *testing.T) {
	cfg := &Config{}

	if err := cfg.CreatePreset(Preset{Name: "new", Format: "%Y"}); err != nil {
		t.Fatalf("CreatePreset: %v", err)
	}
	if len(cfg.Presets) != 1 {
		t.Fatalf("len = %d, want 1", len(cfg.Presets))
	}

	err := cfg.CreatePreset(Preset{Name: "new", Format: "%Y%m%d"})
	if err == nil {
		t.Fatal("expected error for duplicate name")
	}

	err = cfg.CreatePreset(Preset{Name: ""})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestUpdatePreset(t *testing.T) {
	cfg := &Config{
		Presets: []Preset{
			{Name: "x", Format: "%Y", Template: "prefix"},
		},
	}

	if err := cfg.UpdatePreset("x", Preset{Format: "%Y%m%d"}); err != nil {
		t.Fatalf("UpdatePreset: %v", err)
	}

	p, _ := cfg.FindPreset("x")
	if p.Format != "%Y%m%d" {
		t.Errorf("Format = %q, want %%Y%%m%%d", p.Format)
	}
	if p.Template != "prefix" {
		t.Errorf("Template = %q, want prefix (should be unchanged)", p.Template)
	}

	if err := cfg.UpdatePreset("missing", Preset{Format: "%Y"}); err == nil {
		t.Fatal("expected error for missing preset")
	}
}

func TestDeletePreset(t *testing.T) {
	cfg := &Config{
		Default: "a",
		Presets: []Preset{
			{Name: "a"},
			{Name: "b"},
		},
	}

	if err := cfg.DeletePreset("a"); err != nil {
		t.Fatalf("DeletePreset: %v", err)
	}
	if len(cfg.Presets) != 1 {
		t.Fatalf("len = %d, want 1", len(cfg.Presets))
	}
	if cfg.Default != "" {
		t.Errorf("Default = %q, want empty (should be cleared when default is deleted)", cfg.Default)
	}

	if err := cfg.DeletePreset("missing"); err == nil {
		t.Fatal("expected error for missing preset")
	}
}

func TestSetDefault(t *testing.T) {
	cfg := &Config{
		Presets: []Preset{{Name: "a"}, {Name: "b"}},
	}

	if err := cfg.SetDefault("b"); err != nil {
		t.Fatalf("SetDefault: %v", err)
	}
	if cfg.Default != "b" {
		t.Errorf("Default = %q, want b", cfg.Default)
	}

	if err := cfg.SetDefault("missing"); err == nil {
		t.Fatal("expected error for missing preset")
	}
}

func TestResolvePreset(t *testing.T) {
	cfg := &Config{
		Default: "a",
		Presets: []Preset{
			{Name: "a", Format: "%Y"},
			{Name: "b", Format: "%Y%m%d"},
		},
	}

	p, err := cfg.ResolvePreset("b")
	if err != nil {
		t.Fatalf("explicit: %v", err)
	}
	if p.Format != "%Y%m%d" {
		t.Errorf("explicit: Format = %q, want %%Y%%m%%d", p.Format)
	}

	p, err = cfg.ResolvePreset("")
	if err != nil {
		t.Fatalf("default: %v", err)
	}
	if p.Format != "%Y" {
		t.Errorf("default: Format = %q, want %%Y", p.Format)
	}

	_, err = cfg.ResolvePreset("missing")
	if err == nil {
		t.Fatal("expected error for missing preset")
	}

	cfg.Default = ""
	p, err = cfg.ResolvePreset("")
	if err != nil {
		t.Fatalf("no default: %v", err)
	}
	if p.Name != "" {
		t.Errorf("no default: expected zero Preset, got %+v", p)
	}
}

func TestMigrateProfiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	legacy := `profiles:
  backup:
    format: "%Y-%m-%dT%H:%M:%S%z"
    placement: after-ext
    copy: true
  compact:
    format: "%Y%m%d"
  photo:
    format: "%Y-%m-%d"
    source: btime
    template: "{stamp}_{name}{ext}"
`
	if err := os.WriteFile(path, []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, migrated, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !migrated {
		t.Error("expected migrated=true for legacy profiles")
	}
	if len(cfg.Presets) != 3 {
		t.Fatalf("len(Presets) = %d, want 3", len(cfg.Presets))
	}

	// Sorted alphabetically by migrateProfiles.
	if cfg.Presets[0].Name != "backup" {
		t.Errorf("Presets[0].Name = %q, want backup", cfg.Presets[0].Name)
	}
	if cfg.Presets[0].Template != "suffix" {
		t.Errorf("backup.Template = %q, want suffix (migrated from after-ext)", cfg.Presets[0].Template)
	}
	if !cfg.Presets[0].Copy {
		t.Error("backup.Copy = false, want true")
	}

	if cfg.Presets[1].Name != "compact" {
		t.Errorf("Presets[1].Name = %q, want compact", cfg.Presets[1].Name)
	}

	if cfg.Presets[2].Name != "photo" {
		t.Errorf("Presets[2].Name = %q, want photo", cfg.Presets[2].Name)
	}
	if cfg.Presets[2].Template != "{stamp}_{name}{ext}" {
		t.Errorf("photo.Template = %q, want {stamp}_{name}{ext}", cfg.Presets[2].Template)
	}
}

func TestMigrateProfilesIgnoredWhenPresetsExist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	// File has both profiles and presets; profiles should be ignored.
	both := `presets:
  - name: existing
    format: "%Y"
profiles:
  old:
    format: "%Y%m%d"
`
	if err := os.WriteFile(path, []byte(both), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, migrated, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if migrated {
		t.Error("expected migrated=false when presets already exist")
	}
	if len(cfg.Presets) != 1 || cfg.Presets[0].Name != "existing" {
		t.Errorf("Presets = %+v, want only existing", cfg.Presets)
	}
}

func TestSaveCreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "config.yaml")

	cfg := &Config{Presets: []Preset{{Name: "test"}}}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("file mode = %o, want 0600", info.Mode().Perm())
	}
}

func TestDefaultPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/custom/config")
	p, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	want := filepath.Join("/custom/config", "stamp", "config.yaml")
	if p != want {
		t.Errorf("DefaultPath = %q, want %q", p, want)
	}

	t.Setenv("XDG_CONFIG_HOME", "")
	p, err = DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	home, _ := os.UserHomeDir()
	want = filepath.Join(home, ".config", "stamp", "config.yaml")
	if p != want {
		t.Errorf("DefaultPath = %q, want %q", p, want)
	}
}
