# Cleanup batch

A one-shot Go CLI that recursively removes regular files older than three calendar months from configured folders. Age uses **last modification time (mtime)**. Requires Go 1.25+.

## YAML configuration

Copy `config.example.yaml` to `config.yaml`, then edit the directories:

```yaml
directories:
  - /Users/your-name/Downloads
  - /Users/your-name/Documents
retention_months: 3
dry_run: true
```

```sh
go run ./cmd/cleanup --config config.yaml
make build
./bin/cleanup --config /absolute/path/config.yaml
make check
```

Preview with `dry_run: true`. Set `dry_run: false` to delete matched files. The config is loaded on **every invocation**; edits take effect on the next batch without rebuilding. This is a one-shot job, so there is no watcher or reload during a running batch.

Without `--config`, the binary reads `config.yaml` beside its executable, regardless of the terminal's working directory. For example, place the config in `bin/config.yaml` and run `./bin/cleanup`, or open the binary from Finder. An explicit relative `--config` path resolves from the working directory. With `go run`, pass `--config config.yaml` because its executable is built in a temporary folder.

`directories` must contain at least one nonblank folder. Each folder runs sequentially with its own summary. Relative paths resolve from the YAML file's folder. Omitted `retention_months` defaults to 3 and omitted `dry_run` defaults to true. Unknown/duplicate keys, invalid values and multiple YAML documents are rejected. YAML parsing uses [go.yaml.in/yaml/v3](https://pkg.go.dev/go.yaml.in/yaml/v3).

Retention accepts 1–1200 calendar months. The cutoff uses the machine's local timezone and clamps month-end dates: May 31 minus three months is February 28 (29 in a leap year). Files exactly at the cutoff are kept.

The batch walks subfolders, skips symlinks and special files, and preserves directories. Filesystem roots are rejected. `os.Root` confines filesystem operations to each selected tree. Each match and a summary are written to stdout; errors go to stderr with exit code 1. The batch stops on its first error, reports partial progress and leaves subsequent folders unprocessed. SIGINT/SIGTERM cancel the walk.

Run against a folder whose writers are paused. Before removal, the batch rechecks file type, size, and mtime and skips changed or missing files. The check and removal are separate operations; concurrent replacements after the check cannot be fully prevented. Nested mount points are traversed, so select a folder containing only data intended for this policy. Deleted files require a backup to restore.

## Clean Architecture

```text
cmd/cleanup/                  CLI, signals, output, wiring
internal/domain/              File age rule and calendar cutoff
internal/usecase/             Cleanup batch and Store port
internal/adapter/filesystem/  os.Root filesystem adapter
internal/adapter/config/      YAML loading and validation
```

The domain depends only on the standard library. The use case owns its storage interface; the filesystem adapter implements it. The CLI connects the layers. Files are handled sequentially without collecting all matches in memory.

## Schedule

Build the binary and invoke it with cron (example: daily at 02:00). Replace paths with absolute paths appropriate to the host. Cron's timezone determines execution time; the process timezone determines the cutoff.

```cron
0 2 * * * /absolute/path/bin/cleanup --config /absolute/path/config.yaml >> /absolute/path/cleanup.log 2>&1
```

Keep the binary, configuration and log outside the cleanup folder.
