// Command govno dumps MySQL databases to S3 on a rotating schedule.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/nicklasos/govno/v2/internal/backup"
	"github.com/nicklasos/govno/v2/internal/config"
	"github.com/nicklasos/govno/v2/internal/dump"
	"github.com/nicklasos/govno/v2/internal/storage"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("govno", flag.ContinueOnError)
	fs.SetOutput(stderr)
	configPath := fs.String("config", "", "config file (default $GOVNO_CONFIG, ~/.govno or /etc/govno/govno.toml)")
	dbFilter := fs.String("db", "", "only back up this database")
	dryRun := fs.Bool("dry-run", false, "show what would be uploaded and deleted without doing it")
	logFormat := fs.String("log-format", "text", "log format: text or json")
	showVersion := fs.Bool("version", false, "print version and exit")
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: govno [flags] <vno>\n\nRuns the backup schedule (vno) named <vno>, e.g. \"daily\".\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Fprintln(stdout, "govno", version)
		return 0
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}
	vno := fs.Arg(0)

	var handler slog.Handler
	switch *logFormat {
	case "text":
		handler = slog.NewTextHandler(stderr, nil)
	case "json":
		handler = slog.NewJSONHandler(stderr, nil)
	default:
		fmt.Fprintf(stderr, "unknown -log-format %q\n", *logFormat)
		return 2
	}
	logger := slog.New(handler)

	path := *configPath
	if path == "" {
		path = config.DefaultPath()
	}
	cfg, err := config.Load(path)
	if err != nil {
		logger.Error("invalid config", "error", err)
		return 1
	}

	hostname, err := os.Hostname()
	if err != nil {
		logger.Warn("cannot determine hostname", "error", err)
	}

	runner := &backup.Runner{
		Dumper: dump.MySQL{},
		NewStorage: func(ctx context.Context, c config.S3) (backup.Storage, error) {
			return storage.NewS3(ctx, c)
		},
		Logger:   logger,
		Hostname: hostname,
		DryRun:   *dryRun,
	}
	if err := runner.Run(ctx, cfg, vno, *dbFilter); err != nil {
		logger.Error("govno finished with errors", "error", err)
		return 1
	}
	return 0
}
