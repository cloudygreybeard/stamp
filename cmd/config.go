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

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"

	"github.com/cloudygreybeard/stamp/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage stamp configuration and presets",
	Long: `View and modify stamp's persistent configuration.

Presets are named groups of settings stored in the config file. Use the
subcommands below to create, edit, list, and delete presets, or to select
a default preset.`,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all presets",
	Args:  cobra.NoArgs,
	RunE:  runConfigList,
}

var configShowCmd = &cobra.Command{
	Use:   "show [NAME]",
	Short: "Show a preset (or the full config if no name given)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runConfigShow,
}

var configCreateCmd = &cobra.Command{
	Use:   "create NAME",
	Short: "Create a new preset",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigCreate,
}

var configEditCmd = &cobra.Command{
	Use:   "edit NAME",
	Short: "Update fields in an existing preset",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigEdit,
}

var configDeleteCmd = &cobra.Command{
	Use:   "delete NAME",
	Short: "Remove a preset",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigDelete,
}

var configUseDefaultCmd = &cobra.Command{
	Use:   "use-default NAME",
	Short: "Set the default preset",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigUseDefault,
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the resolved config file path",
	Args:  cobra.NoArgs,
	RunE:  runConfigPath,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configCreateCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configDeleteCmd)
	configCmd.AddCommand(configUseDefaultCmd)
	configCmd.AddCommand(configPathCmd)

	for _, cmd := range []*cobra.Command{configCreateCmd, configEditCmd} {
		cmd.Flags().StringP("format", "F", "", "strftime format")
		cmd.Flags().StringP("source", "s", "", "timestamp source")
		cmd.Flags().StringP("template", "t", "", "output template (shortcut or pattern)")
		cmd.Flags().String("separator", "", "separator character")
		cmd.Flags().StringP("replace", "r", "", "regex replacement pattern")
		cmd.Flags().BoolP("copy", "c", false, "copy instead of rename")
		cmd.Flags().IntP("workers", "j", 0, "concurrent workers")
	}
}

func resolveConfigPath() (string, error) {
	if cfgFile != "" {
		return cfgFile, nil
	}
	return config.DefaultPath()
}

func loadConfig() (*config.Config, string, error) {
	path, err := resolveConfigPath()
	if err != nil {
		return nil, "", err
	}

	cfg, migrated, err := config.Load(path)
	if err != nil {
		return nil, "", err
	}
	if migrated {
		fmt.Fprintln(os.Stderr, "stamp: migrated legacy profiles: key; consider running 'stamp config list' to review")
	}
	return cfg, path, nil
}

func saveConfig(path string, cfg *config.Config) error {
	return config.Save(path, cfg)
}

func runConfigList(_ *cobra.Command, _ []string) error {
	cfg, _, err := loadConfig()
	if err != nil {
		return err
	}

	if len(cfg.Presets) == 0 {
		fmt.Println("No presets configured.")
		return nil
	}

	for _, name := range cfg.PresetNames() {
		if name == cfg.Default {
			fmt.Printf("* %s (default)\n", name)
		} else {
			fmt.Printf("  %s\n", name)
		}
	}
	return nil
}

func runConfigShow(_ *cobra.Command, args []string) error {
	cfg, _, err := loadConfig()
	if err != nil {
		return err
	}

	var out interface{}
	if len(args) == 0 {
		out = cfg
	} else {
		p, ok := cfg.FindPreset(args[0])
		if !ok {
			return fmt.Errorf("preset %q not found", args[0])
		}
		out = p
	}

	data, err := yaml.Marshal(out)
	if err != nil {
		return err
	}
	fmt.Print(string(data))
	return nil
}

func runConfigCreate(cmd *cobra.Command, args []string) error {
	cfg, path, err := loadConfig()
	if err != nil {
		return err
	}

	p := presetFromFlags(cmd, args[0])
	if err := cfg.CreatePreset(p); err != nil {
		return err
	}

	if err := saveConfig(path, cfg); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Created preset %q\n", args[0])
	return nil
}

func runConfigEdit(cmd *cobra.Command, args []string) error {
	cfg, path, err := loadConfig()
	if err != nil {
		return err
	}

	patch := presetFromFlags(cmd, args[0])
	if err := cfg.UpdatePreset(args[0], patch); err != nil {
		return err
	}

	if err := saveConfig(path, cfg); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Updated preset %q\n", args[0])
	return nil
}

func runConfigDelete(_ *cobra.Command, args []string) error {
	cfg, path, err := loadConfig()
	if err != nil {
		return err
	}

	if err := cfg.DeletePreset(args[0]); err != nil {
		return err
	}

	if err := saveConfig(path, cfg); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Deleted preset %q\n", args[0])
	return nil
}

func runConfigUseDefault(_ *cobra.Command, args []string) error {
	cfg, path, err := loadConfig()
	if err != nil {
		return err
	}

	if err := cfg.SetDefault(args[0]); err != nil {
		return err
	}

	if err := saveConfig(path, cfg); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Default preset set to %q\n", args[0])
	return nil
}

func runConfigPath(_ *cobra.Command, _ []string) error {
	path, err := resolveConfigPath()
	if err != nil {
		return err
	}
	fmt.Println(path)
	return nil
}

func presetFromFlags(cmd *cobra.Command, name string) config.Preset {
	p := config.Preset{Name: name}

	if cmd.Flags().Changed("format") {
		p.Format, _ = cmd.Flags().GetString("format")
	}
	if cmd.Flags().Changed("source") {
		p.Source, _ = cmd.Flags().GetString("source")
	}
	if cmd.Flags().Changed("template") {
		p.Template, _ = cmd.Flags().GetString("template")
	}
	if cmd.Flags().Changed("separator") {
		p.Separator, _ = cmd.Flags().GetString("separator")
	}
	if cmd.Flags().Changed("replace") {
		p.Replace, _ = cmd.Flags().GetString("replace")
	}
	if cmd.Flags().Changed("copy") {
		p.Copy, _ = cmd.Flags().GetBool("copy")
	}
	if cmd.Flags().Changed("workers") {
		p.Workers, _ = cmd.Flags().GetInt("workers")
	}

	return p
}
