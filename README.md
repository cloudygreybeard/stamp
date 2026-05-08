# stamp

Insert timestamps into filenames.

`stamp` renames (or copies) files by inserting a formatted timestamp into
the filename. The timestamp source, format, and template are all
configurable through flags, environment variables, config files, or named
presets.

```
stamp report.md
  report.md -> report.2026-04-26T07:32:00+1000.md
```

## Quick start

```bash
# Stamp a file with its last-modified time (the default)
stamp report.md

# Stamp several files at once
stamp *.log

# Preview what would happen without touching anything
stamp --dry-run *.md

# Use a compact date instead of full ISO 8601
stamp --format "%Y%m%d" report.md
#   report.md -> report.20260426.md

# Copy instead of rename
stamp --copy report.md
#   report.md is kept, report.2026-04-26T07:32:00+1000.md is created

# Put the timestamp at the front of the filename
stamp --template prefix --separator "_" --format "%Y%m%d" photo.jpg
#   photo.jpg -> 20260426_photo.jpg
```

## Installation

### Homebrew (macOS and Linux)

```bash
brew install cloudygreybeard/tap/stamp
```

### Go install

```bash
go install github.com/cloudygreybeard/stamp@latest
```

### From source

```bash
git clone https://github.com/cloudygreybeard/stamp.git
cd stamp
make install
```

## Usage

### Specifying files

Files can come from three sources, checked in this order:

1. **Positional arguments** on the command line.
2. **A file list** with `--file` (`-f`). Use `-f -` to read the list from
   stdin explicitly.
3. **stdin**, detected automatically when input is piped.

```bash
# Positional arguments
stamp report.md photo.jpg

# File list
stamp --file list.txt

# Piped from another command
find . -name "*.log" -mtime +30 | stamp --copy --format "%Y%m%d"
```

### Timestamp source

The `--source` (`-s`) flag controls where the timestamp value comes from.
The default is `mtime`.

| Source | Description |
|--------|-------------|
| `mtime` | Last modification time (default) |
| `atime` | Last access time |
| `btime` | Birth (creation) time; available on macOS and Windows |
| `now` | Current wall-clock time |
| Any other value | Used verbatim as a literal string; `--format` is ignored |

```bash
stamp --source atime *.log
stamp --source now draft.txt
stamp --source btime photo.jpg

# Literal string (format is ignored)
stamp --source "v2" config.yaml
#   config.yaml -> config.v2.yaml
```

### Timestamp format

The `--format` (`-F`) flag accepts a strftime format string. The default is
`%Y-%m-%dT%H:%M:%S%z` (ISO 8601 with timezone offset).

```bash
stamp --format "%Y%m%d" report.md
#   report.md -> report.20260426.md

stamp --format "%Y-%m-%d" report.md
#   report.md -> report.2026-04-26.md

stamp --format "%Y%m%d-%H%M%S" report.md
#   report.md -> report.20260426-073200.md
```

See the [strftime directives](#strftime-directives) table for the full list
of supported specifiers.

### Template

The `--template` (`-t`) flag controls where the timestamp appears in the
output filename. It accepts either a **named shortcut** or a **token
pattern**.

**Named shortcuts** (used with `--separator`, default `.`):

| Shortcut | Example |
|----------|---------|
| `before-ext` (default) | `report.md` -> `report.20260426.md` |
| `suffix` | `report.md` -> `report.md.20260426` |
| `prefix` | `report.md` -> `20260426.report.md` |

```bash
stamp --template suffix backup.sql
#   backup.sql -> backup.sql.2026-04-26T07:32:00+1000

stamp --template prefix photo.jpg
#   photo.jpg -> 2026-04-26T07:32:00+1000.photo.jpg
```

**Token patterns** give full control over the output filename. Available
placeholders:

| Placeholder | Value |
|-------------|-------|
| `{name}` | Filename without extension |
| `{ext}` | File extension including the leading dot |
| `{stamp}` | Formatted timestamp string |
| `{sep}` | The configured separator |

```bash
stamp --template "{stamp} - {name}{ext}" --format "%Y-%m-%d" notes.md
#   notes.md -> 2026-04-26 - notes.md

stamp --template "backup-{name}-{stamp}{ext}" --format "%Y%m%d" config.yaml
#   config.yaml -> backup-config-20260426.yaml

stamp --template "{name}{sep}{stamp}{ext}" --separator "_" data.csv
#   data.csv -> data_2026-04-26T07:32:00+1000.csv
```

### Separator

The `--separator` flag changes the character between filename components
when using a named shortcut. The default is `.` (dot). When a full token
pattern is used, the separator is whatever you place in the pattern
literally.

```bash
stamp --separator "_" --format "%Y%m%d" report.md
#   report.md -> report_20260426.md

stamp --template prefix --separator "_" --format "%Y%m%d" photo.jpg
#   photo.jpg -> 20260426_photo.jpg
```

### Pattern replacement

Use `--replace` (`-r`) with a regex to match and replace part of an existing
filename with the timestamp. When the pattern matches it takes priority over
`--template`. If the pattern does not match, `stamp` falls through to the
configured template.

```bash
# Re-stamp a previously stamped file
stamp --replace "\d{4}-\d{2}-\d{2}" --format "%Y-%m-%d" report.2024-01-15.md
#   report.2024-01-15.md -> report.2026-04-26.md

# Replace a version tag with a literal
stamp --replace "v\d+" --source "v3" config.v2.yaml
#   config.v2.yaml -> config.v3.yaml

# No match: falls through to default template
stamp --replace "\d{8}" report.md
#   report.md -> report.2026-04-26T07:32:00+1000.md
```

### Copy mode

By default `stamp` renames the original file. Pass `--copy` (`-c`) to keep
the original and create a stamped copy.

```bash
stamp --copy report.md
#   report.md is preserved
#   report.2026-04-26T07:32:00+1000.md is created
```

If the destination path already exists, `stamp` reports an error and skips
that file.

### Dry run

Use `--dry-run` (`-n`) to preview the result without modifying anything.

```bash
stamp --dry-run *.md
#   would rename: notes.md -> notes.2026-04-26T07:32:00+1000.md
#   would rename: report.md -> report.2026-04-26T07:32:00+1000.md
```

### Concurrency

When processing many files, `stamp` runs multiple workers in parallel. The
`--workers` (`-j`) flag controls the concurrency limit.

```bash
# Auto (default): uses one worker per CPU core, capped to the file count
stamp *.log

# Explicit limit
stamp --workers 16 *.log

# Sequential processing
stamp --workers 1 *.log
```

Output order always matches input order regardless of how many workers are
used. Set `--workers 0` (or omit the flag) for automatic sizing.

### Presets

Named presets bundle multiple settings in the config file. Invoke one with
`--preset` (`-P`). CLI flags and environment variables override preset
values.

```bash
stamp --preset backup *.txt
stamp --preset compact report.md
```

See [Configuration](#configuration) for how to define and manage presets.

## Configuration

Settings are resolved in this order (highest priority first):

1. CLI flags
2. Environment variables
3. `--preset NAME` (explicit)
4. Default preset (from config file)
5. Built-in defaults

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--format` | `-F` | strftime format for the timestamp | `%Y-%m-%dT%H:%M:%S%z` |
| `--source` | `-s` | Timestamp source: `mtime`, `atime`, `btime`, `now`, or literal | `mtime` |
| `--copy` | `-c` | Copy files instead of renaming | `false` |
| `--dry-run` | `-n` | Preview changes without applying | `false` |
| `--file` | `-f` | Read filenames from a file (`-` for stdin) | |
| `--template` | `-t` | Shortcut (`before-ext`, `prefix`, `suffix`) or token pattern | `before-ext` |
| `--separator` | | Separator for named shortcuts | `.` |
| `--replace` | `-r` | Regex in basename to replace with timestamp | |
| `--preset` | `-P` | Named preset from config file | |
| `--workers` | `-j` | Concurrent workers (`0` = auto, based on CPU count) | `0` |
| `--config` | | Path to config file | `~/.config/stamp/config.yaml` |

### Environment variables

Set any flag via an environment variable with the `STAMP_` prefix. Hyphens
in flag names become underscores.

```bash
export STAMP_FORMAT="%Y%m%d"
export STAMP_SOURCE=now
export STAMP_COPY=true
export STAMP_TEMPLATE=prefix
export STAMP_SEPARATOR="_"
export STAMP_WORKERS=8
```

### Config file

`stamp` reads `~/.config/stamp/config.yaml` (following the XDG Base
Directory Specification). Use `--config` to specify a different path. Run
`stamp config path` to see the resolved location.

```yaml
default: compact

presets:
  - name: compact
    format: "%Y%m%d"

  - name: backup
    format: "%Y-%m-%dT%H:%M:%S%z"
    template: suffix
    copy: true

  - name: archive
    format: "%Y%m%d-%H%M%S"
    template: prefix
    separator: "_"
    workers: 8

  - name: photo
    format: "%Y-%m-%d"
    source: btime
    template: "{stamp}_{name}{ext}"

  - name: re-stamp
    format: "%Y-%m-%d"
    replace: "\\d{4}-\\d{2}-\\d{2}"
```

A preset can set any of: `format`, `source`, `copy`, `template`,
`separator`, `replace`, `workers`.

### Managing presets with `stamp config`

Instead of editing the config file by hand, use the `stamp config`
subcommands:

```bash
# Create a preset
stamp config create compact --format "%Y%m%d"

# Create another with more options
stamp config create backup --format "%Y-%m-%dT%H:%M:%S%z" --template suffix --copy

# Set a default preset
stamp config use-default compact

# List all presets (default marked with *)
stamp config list
#   backup
# * compact (default)

# Show a single preset
stamp config show compact

# Edit an existing preset
stamp config edit backup --workers 4

# Delete a preset
stamp config delete backup

# Print the config file path
stamp config path
```

### Migrating from profiles

If you used a previous version of `stamp` with `profiles:` in
`~/.stamp.yaml`, the old format is detected automatically and loaded. A
deprecation notice is printed to stderr. To migrate:

1. Run `stamp config list` to confirm your presets were loaded.
2. Copy the file to the new location:
   `mkdir -p ~/.config/stamp && cp ~/.stamp.yaml ~/.config/stamp/config.yaml`
3. Edit `~/.config/stamp/config.yaml` to use the new format (rename
   `profiles:` to `presets:` as an array of objects, rename `placement:` to
   `template:`, rename `after-ext` to `suffix`).

## strftime directives

| Directive | Description | Example |
|-----------|-------------|---------|
| `%Y` | 4-digit year | `2026` |
| `%y` | 2-digit year | `26` |
| `%C` | Century | `20` |
| `%m` | Month, zero-padded (01--12) | `04` |
| `%d` | Day of month, zero-padded (01--31) | `26` |
| `%e` | Day of month, space-padded | ` 5` |
| `%H` | Hour, 24h zero-padded (00--23) | `07` |
| `%I` | Hour, 12h zero-padded (01--12) | `07` |
| `%M` | Minute (00--59) | `32` |
| `%S` | Second (00--59) | `00` |
| `%p` | AM or PM | `AM` |
| `%P` | am or pm | `am` |
| `%z` | UTC offset | `+1000` |
| `%Z` | Timezone abbreviation | `AEST` |
| `%a` | Abbreviated weekday | `Sun` |
| `%A` | Full weekday | `Sunday` |
| `%b` / `%h` | Abbreviated month | `Apr` |
| `%B` | Full month name | `April` |
| `%j` | Day of year (001--366) | `116` |
| `%u` | Weekday number, Mon=1 (1--7) | `7` |
| `%w` | Weekday number, Sun=0 (0--6) | `0` |
| `%G` | ISO week-numbering year | `2026` |
| `%V` | ISO week number (01--53) | `17` |
| `%s` | Unix epoch seconds | `1777321920` |
| `%N` | Nanoseconds | `000000000` |
| `%F` | `%Y-%m-%d` | `2026-04-26` |
| `%T` | `%H:%M:%S` | `07:32:00` |
| `%R` | `%H:%M` | `07:32` |
| `%%` | Literal `%` | `%` |

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | All files processed successfully |
| 1 | One or more files could not be stamped, or invalid arguments |

Errors for individual files are printed to stderr. Successfully stamped
files are printed to stdout.

## Development

Prerequisites: Go 1.25+, [golangci-lint](https://golangci-lint.run/)
(for linting).

```bash
make build      # Build the binary
make test       # Run tests with race detector
make vet        # Run go vet
make lint       # Run golangci-lint
make fmt        # Format source code
make cover      # Generate and open HTML coverage report
make check      # Run vet + test (no external tools required)
make ci         # Run the full CI suite locally (vet, test, lint)
make snapshot   # Build a snapshot release with GoReleaser
make clean      # Remove build artifacts
make deps       # Download and tidy dependencies
make help       # Show all targets
```

## License

Apache 2.0. See [LICENSE](LICENSE).
