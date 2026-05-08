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
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cloudygreybeard/stamp/internal/config"
	"github.com/cloudygreybeard/stamp/internal/stamper"
)

// Version information set via ldflags at build time.
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "stamp [flags] [FILE...]",
	Short: "Insert timestamps into filenames",
	Long: `stamp renames (or copies) files by inserting a formatted timestamp
into the filename.

Files are specified as positional arguments, read from a file with --file,
or piped via stdin. The timestamp defaults to the file's last modification
time, formatted as an ISO 8601 string.

Examples:
  stamp report.md
    report.md -> report.2026-04-26T07:32:00+1000.md

  stamp --copy --format "%Y%m%d" *.log
    app.log -> app.20260426.log  (original kept)

  stamp --template prefix --separator "_" photo.jpg
    photo.jpg -> 20260506_photo.jpg

  stamp --template "{stamp} - {name}{ext}" notes.md
    notes.md -> 2026-05-06T09:00:00+1000 - notes.md

  stamp --replace "\d{4}-\d{2}-\d{2}" report.2024-01-15.md
    report.2024-01-15.md -> report.2026-05-06.md

  stamp --preset backup *.txt

  find . -name "*.bak" | stamp --dry-run`,
	Args:          cobra.ArbitraryArgs,
	RunE:          runStamp,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		"config file (default ~/.config/stamp/config.yaml)")

	rootCmd.Flags().StringP("format", "F", "%Y-%m-%dT%H:%M:%S%z",
		"strftime format for the timestamp")
	rootCmd.Flags().StringP("source", "s", "mtime",
		"timestamp source: mtime, atime, btime, now, or a literal string")
	rootCmd.Flags().BoolP("copy", "c", false,
		"copy files instead of renaming")
	rootCmd.Flags().BoolP("dry-run", "n", false,
		"show what would be done without doing it")
	rootCmd.Flags().StringP("file", "f", "",
		"read filenames from a file (- for stdin)")

	rootCmd.Flags().String("separator", ".",
		"separator between filename components and the timestamp")
	rootCmd.Flags().StringP("template", "t", "",
		"output template: shortcut (before-ext, prefix, suffix) or pattern ({name}{ext}{sep}{stamp})")
	rootCmd.Flags().StringP("replace", "r", "",
		"regex pattern in the basename to replace with the timestamp")
	rootCmd.Flags().StringP("preset", "P", "",
		"named preset from config file")
	rootCmd.Flags().IntP("workers", "j", 0,
		"number of concurrent workers (0 = auto, based on CPU count)")

	// Hidden backward-compat alias for --preset.
	rootCmd.Flags().String("profile", "", "")
	_ = rootCmd.Flags().MarkHidden("profile")

	for _, key := range []string{"format", "source", "copy", "separator", "template", "replace", "workers"} {
		_ = viper.BindPFlag(key, rootCmd.Flags().Lookup(key))
	}
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// XDG-compliant primary location.
		if p, err := config.DefaultPath(); err == nil {
			viper.SetConfigFile(p)
		}
	}

	viper.SetEnvPrefix("STAMP")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if cfgFile == "" {
			// Fall back to legacy locations when the XDG path is missing.
			viper.SetConfigName(".stamp")
			if home, err := os.UserHomeDir(); err == nil {
				viper.AddConfigPath(home)
			}
			viper.AddConfigPath(".")
			_ = viper.ReadInConfig()
		}
	}
}

// presetKeys are the option keys a preset can supply.
var presetKeys = []string{
	"format", "source", "copy", "separator", "template", "replace", "workers",
}

// applyPreset merges a resolved Preset into the active viper settings.
// CLI flags and env vars take precedence over preset values.
func applyPreset(cmd *cobra.Command, p config.Preset) {
	vals := map[string]interface{}{
		"format":    p.Format,
		"source":    p.Source,
		"copy":      p.Copy,
		"separator": p.Separator,
		"template":  p.Template,
		"replace":   p.Replace,
		"workers":   p.Workers,
	}

	for _, key := range presetKeys {
		if cmd.Flags().Changed(key) {
			continue
		}
		envKey := "STAMP_" + strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
		if _, envSet := os.LookupEnv(envKey); envSet {
			continue
		}
		v := vals[key]
		switch tv := v.(type) {
		case string:
			if tv != "" {
				viper.Set(key, tv)
			}
		case bool:
			if tv {
				viper.Set(key, tv)
			}
		case int:
			if tv != 0 {
				viper.Set(key, tv)
			}
		}
	}
}

func collectFiles(cmd *cobra.Command, args []string) ([]string, error) {
	if len(args) > 0 {
		return args, nil
	}

	fileFlag, _ := cmd.Flags().GetString("file")
	if fileFlag != "" {
		return readFileList(fileFlag)
	}

	info, err := os.Stdin.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat stdin: %w", err)
	}
	if info.Mode()&os.ModeCharDevice == 0 {
		return scanLines(os.Stdin)
	}

	return nil, fmt.Errorf("no files specified; provide filenames as arguments, with --file, or via stdin")
}

func readFileList(path string) ([]string, error) {
	if path == "-" {
		return scanLines(os.Stdin)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return scanLines(f)
}

func scanLines(r io.Reader) ([]string, error) {
	var lines []string
	s := bufio.NewScanner(r)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, s.Err()
}

func buildOptions(cmd *cobra.Command) (stamper.Options, error) {
	// Resolve the preset name: --preset flag, hidden --profile alias, or
	// the configured default.
	presetName, _ := cmd.Flags().GetString("preset")
	if presetName == "" {
		presetName, _ = cmd.Flags().GetString("profile")
	}

	cfgPath, _ := resolveConfigPath()
	cfg, migrated, err := config.Load(cfgPath)
	if err != nil {
		return stamper.Options{}, fmt.Errorf("load config: %w", err)
	}
	if migrated {
		fmt.Fprintln(os.Stderr, "stamp: migrated legacy profiles: key to presets; run 'stamp config list' to review")
	}

	p, err := cfg.ResolvePreset(presetName)
	if err != nil {
		return stamper.Options{}, err
	}
	if p.Name != "" {
		applyPreset(cmd, p)
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")

	opts := stamper.Options{
		Format:    viper.GetString("format"),
		Source:    stamper.Source(viper.GetString("source")),
		Copy:      viper.GetBool("copy"),
		DryRun:    dryRun,
		Separator: viper.GetString("separator"),
		Template:  viper.GetString("template"),
	}

	if pat := viper.GetString("replace"); pat != "" {
		re, err := regexp.Compile(pat)
		if err != nil {
			return stamper.Options{}, fmt.Errorf("invalid --replace pattern: %w", err)
		}
		opts.Replace = re
	}

	return opts, nil
}

// effectiveWorkers returns the number of goroutines to use given the
// requested count and the number of files to process.  A requested value
// of 0 means auto (runtime.NumCPU, capped to the file count).
func effectiveWorkers(requested, fileCount int) int {
	if fileCount <= 0 {
		return 1
	}
	n := requested
	if n <= 0 {
		n = runtime.NumCPU()
	}
	if n > fileCount {
		n = fileCount
	}
	if n < 1 {
		n = 1
	}
	return n
}

type fileResult struct {
	path string
	res  stamper.Result
	err  error
}

func runStamp(cmd *cobra.Command, args []string) error {
	files, err := collectFiles(cmd, args)
	if err != nil {
		return err
	}

	opts, err := buildOptions(cmd)
	if err != nil {
		return err
	}

	workers := effectiveWorkers(viper.GetInt("workers"), len(files))

	results := make([]fileResult, len(files))
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for i, f := range files {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, path string) {
			defer wg.Done()
			defer func() { <-sem }()
			res, err := stamper.Process(path, opts)
			results[idx] = fileResult{path: path, res: res, err: err}
		}(i, f)
	}
	wg.Wait()

	verb := "renamed"
	if opts.Copy {
		verb = "copied"
	}
	if opts.DryRun {
		verb = "would rename"
		if opts.Copy {
			verb = "would copy"
		}
	}

	var errCount int
	for _, r := range results {
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "stamp: %s: %s\n", r.path, r.err)
			errCount++
			continue
		}
		fmt.Fprintf(os.Stdout, "%s: %s -> %s\n", verb, r.res.Original, r.res.Stamped)
	}

	if errCount > 0 {
		return fmt.Errorf("%d file(s) could not be stamped", errCount)
	}
	return nil
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
