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

// Package config manages stamp's persistent configuration file, including
// named presets and the default preset selection.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"go.yaml.in/yaml/v3"
)

// Config is the top-level configuration structure, serialised as YAML.
type Config struct {
	Default string   `yaml:"default,omitempty"`
	Presets []Preset `yaml:"presets,omitempty"`
}

// Preset holds a named set of stamping options.
type Preset struct {
	Name      string `yaml:"name"`
	Format    string `yaml:"format,omitempty"`
	Source    string `yaml:"source,omitempty"`
	Template  string `yaml:"template,omitempty"`
	Separator string `yaml:"separator,omitempty"`
	Replace   string `yaml:"replace,omitempty"`
	Copy      bool   `yaml:"copy,omitempty"`
	Workers   int    `yaml:"workers,omitempty"`
}

// FindPreset returns the preset with the given name and true, or a zero
// Preset and false if not found.
func (c *Config) FindPreset(name string) (Preset, bool) {
	for _, p := range c.Presets {
		if p.Name == name {
			return p, true
		}
	}
	return Preset{}, false
}

// PresetNames returns a sorted list of all preset names.
func (c *Config) PresetNames() []string {
	names := make([]string, len(c.Presets))
	for i, p := range c.Presets {
		names[i] = p.Name
	}
	sort.Strings(names)
	return names
}

// CreatePreset adds a new preset. It returns an error if a preset with the
// same name already exists.
func (c *Config) CreatePreset(p Preset) error {
	if p.Name == "" {
		return errors.New("preset name must not be empty")
	}
	if _, exists := c.FindPreset(p.Name); exists {
		return fmt.Errorf("preset %q already exists", p.Name)
	}
	c.Presets = append(c.Presets, p)
	return nil
}

// UpdatePreset replaces an existing preset by name. Only non-zero fields
// in the supplied Preset are applied; the rest are left unchanged.
func (c *Config) UpdatePreset(name string, patch Preset) error {
	for i, p := range c.Presets {
		if p.Name != name {
			continue
		}
		if patch.Format != "" {
			p.Format = patch.Format
		}
		if patch.Source != "" {
			p.Source = patch.Source
		}
		if patch.Template != "" {
			p.Template = patch.Template
		}
		if patch.Separator != "" {
			p.Separator = patch.Separator
		}
		if patch.Replace != "" {
			p.Replace = patch.Replace
		}
		if patch.Copy {
			p.Copy = patch.Copy
		}
		if patch.Workers != 0 {
			p.Workers = patch.Workers
		}
		c.Presets[i] = p
		return nil
	}
	return fmt.Errorf("preset %q not found", name)
}

// DeletePreset removes a preset by name. If the deleted preset was the
// default, the default is cleared.
func (c *Config) DeletePreset(name string) error {
	for i, p := range c.Presets {
		if p.Name == name {
			c.Presets = append(c.Presets[:i], c.Presets[i+1:]...)
			if c.Default == name {
				c.Default = ""
			}
			return nil
		}
	}
	return fmt.Errorf("preset %q not found", name)
}

// SetDefault sets the default preset. The named preset must exist.
func (c *Config) SetDefault(name string) error {
	if _, ok := c.FindPreset(name); !ok {
		return fmt.Errorf("preset %q not found", name)
	}
	c.Default = name
	return nil
}

// ResolvePreset returns the effective preset for a given invocation.
// If explicit is non-empty it is looked up; otherwise the configured
// default is used. A zero Preset is returned when neither applies.
func (c *Config) ResolvePreset(explicit string) (Preset, error) {
	name := explicit
	if name == "" {
		name = c.Default
	}
	if name == "" {
		return Preset{}, nil
	}
	p, ok := c.FindPreset(name)
	if !ok {
		return Preset{}, fmt.Errorf("preset %q not found", name)
	}
	return p, nil
}

// DefaultPath returns the XDG-compliant config file path:
// $XDG_CONFIG_HOME/stamp/config.yaml, defaulting to ~/.config/stamp/config.yaml.
func DefaultPath() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("determine home directory: %w", err)
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "stamp", "config.yaml"), nil
}

// Load reads and parses a config file. If the file does not exist an empty
// Config is returned with no error. The second return value is true when a
// legacy profiles: key was migrated; callers should warn on stderr.
func Load(path string) (*Config, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{}, false, nil
		}
		return nil, false, err
	}

	var raw rawConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, false, fmt.Errorf("parse %s: %w", path, err)
	}

	cfg := &Config{Default: raw.Default, Presets: raw.Presets}

	migrated := false
	if len(raw.Profiles) > 0 && len(cfg.Presets) == 0 {
		cfg.Presets = migrateProfiles(raw.Profiles)
		migrated = true
	}

	return cfg, migrated, nil
}

// Save writes the config to the given path, creating parent directories
// (mode 0700) as needed.
func Save(path string, cfg *Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

// rawConfig mirrors Config but also captures the legacy profiles: key for
// backward-compatible loading.
type rawConfig struct {
	Default  string                   `yaml:"default,omitempty"`
	Presets  []Preset                 `yaml:"presets,omitempty"`
	Profiles map[string]legacyPreset  `yaml:"profiles,omitempty"`
}

// legacyPreset captures the old profile format which included a placement
// field instead of template.
type legacyPreset struct {
	Format    string `yaml:"format,omitempty"`
	Source    string `yaml:"source,omitempty"`
	Placement string `yaml:"placement,omitempty"`
	Template  string `yaml:"template,omitempty"`
	Separator string `yaml:"separator,omitempty"`
	Replace   string `yaml:"replace,omitempty"`
	Copy      bool   `yaml:"copy,omitempty"`
	Workers   int    `yaml:"workers,omitempty"`
}

func migrateProfiles(profiles map[string]legacyPreset) []Preset {
	presets := make([]Preset, 0, len(profiles))
	for name, lp := range profiles {
		p := Preset{
			Name:      name,
			Format:    lp.Format,
			Source:    lp.Source,
			Template:  lp.Template,
			Separator: lp.Separator,
			Replace:   lp.Replace,
			Copy:      lp.Copy,
			Workers:   lp.Workers,
		}
		if p.Template == "" && lp.Placement != "" {
			p.Template = migratePlacement(lp.Placement)
		}
		presets = append(presets, p)
	}
	sort.Slice(presets, func(i, j int) bool {
		return presets[i].Name < presets[j].Name
	})
	return presets
}

func migratePlacement(placement string) string {
	switch placement {
	case "after-ext":
		return "suffix"
	default:
		return placement
	}
}
