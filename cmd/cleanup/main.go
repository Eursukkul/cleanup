package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"cleanup/internal/adapter/config"
	"cleanup/internal/adapter/filesystem"
	"cleanup/internal/domain"
	"cleanup/internal/usecase"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr, time.Now()); err != nil {
		fmt.Fprintln(os.Stderr, "cleanup:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer, now time.Time) error {
	flags := flag.NewFlagSet("cleanup", flag.ContinueOnError)
	flags.SetOutput(stderr)
	configFile := flags.String("config", "", "YAML configuration file (default: config.yaml beside executable)")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if *configFile == "" {
		executable, err := os.Executable()
		if err != nil {
			return fmt.Errorf("locate executable: %w", err)
		}
		*configFile = filepath.Join(filepath.Dir(executable), "config.yaml")
	}
	settings, err := config.Load(*configFile)
	if err != nil {
		return err
	}
	cutoff, err := domain.Cutoff(now, settings.RetentionMonths)
	if err != nil {
		return err
	}
	for _, directory := range settings.Directories {
		if err := runDirectory(ctx, directory, cutoff, settings.DryRun, stdout); err != nil {
			return fmt.Errorf("directory %q: %w", directory, err)
		}
	}
	return nil
}

func runDirectory(ctx context.Context, directory string, cutoff time.Time, dryRun bool, stdout io.Writer) error {
	store, err := filesystem.Open(directory)
	if err != nil {
		return err
	}
	defer store.Close()
	mode := "dry-run"
	if !dryRun {
		mode = "delete"
	}
	if _, err := fmt.Fprintf(stdout, "mode=%s cutoff=%s dir=%q\n", mode, cutoff.Format(time.RFC3339Nano), directory); err != nil {
		return err
	}
	result, runErr := (usecase.Cleanup{Store: store}).Run(ctx, cutoff, !dryRun, func(event usecase.Event) error {
		_, err := fmt.Fprintf(stdout, "%s path=%q bytes=%d modified=%s\n", event.Action, event.File.Path, event.File.Size, event.File.ModifiedAt.Format(time.RFC3339Nano))
		return err
	})
	_, outputErr := fmt.Fprintf(stdout, "matched=%d deleted=%d changed=%d matched_bytes=%d\n", result.Matched, result.Deleted, result.Changed, result.Bytes)
	if runErr != nil {
		return runErr
	}
	return outputErr
}
