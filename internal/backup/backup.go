// Package backup runs the dump → upload → prune cycle for a vno.
package backup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/nicklasos/govno/v2/internal/config"
	"github.com/nicklasos/govno/v2/internal/pathtpl"
	"github.com/nicklasos/govno/v2/internal/storage"
)

// Dumper writes a compressed dump of a database.
type Dumper interface {
	Dump(ctx context.Context, db config.Database, w io.Writer) error
}

// Storage is where backups go.
type Storage interface {
	Put(ctx context.Context, key string, r io.Reader) error
	List(ctx context.Context, prefix string) ([]storage.Object, error)
	Delete(ctx context.Context, keys []string) error
}

// StorageFactory creates a Storage for the given S3 settings.
type StorageFactory func(ctx context.Context, cfg config.S3) (Storage, error)

// Runner holds the dependencies of a backup run.
type Runner struct {
	Dumper     Dumper
	NewStorage StorageFactory
	Logger     *slog.Logger
	Hostname   string
	Now        func() time.Time
	DryRun     bool

	stores map[config.S3]Storage
}

// Run backs up every database that has a vno named vnoName. If dbFilter is
// non-empty only that database is processed. A failing database does not stop
// the others; all errors are returned joined.
func (r *Runner) Run(ctx context.Context, cfg *config.Config, vnoName, dbFilter string) error {
	if r.Logger == nil {
		r.Logger = slog.New(slog.DiscardHandler)
	}
	if r.Now == nil {
		r.Now = time.Now
	}
	now := r.Now()

	var errs []error
	matched := 0
	for _, db := range cfg.Database {
		if dbFilter != "" && db.Name != dbFilter {
			continue
		}
		for _, vno := range db.Vno {
			if vno.Name != vnoName {
				continue
			}
			matched++
			if err := r.runOne(ctx, cfg.StorageFor(db), db, vno, now); err != nil {
				r.Logger.Error("backup failed", "db", db.Name, "vno", vno.Name, "error", err)
				errs = append(errs, fmt.Errorf("%s/%s: %w", db.Name, vno.Name, err))
			}
		}
	}

	if matched == 0 {
		if dbFilter != "" {
			return fmt.Errorf("no database %q with vno %q in config", dbFilter, vnoName)
		}
		return fmt.Errorf("no database has vno %q in config", vnoName)
	}
	return errors.Join(errs...)
}

func (r *Runner) runOne(ctx context.Context, s3cfg config.S3, db config.Database, vno config.Vno, now time.Time) error {
	vars := pathtpl.Vars{DB: db.Name, Vno: vno.Name, Hostname: r.Hostname}
	rendered, err := pathtpl.Render(vno.Path, now, vars)
	if err != nil {
		return err
	}
	key := s3cfg.Prefix + rendered
	log := r.Logger.With("db", db.Name, "vno", vno.Name, "bucket", s3cfg.Bucket, "key", key)

	if r.DryRun {
		log.Info("dry run: would upload")
	} else {
		store, err := r.storage(ctx, s3cfg)
		if err != nil {
			return err
		}
		start := time.Now()
		n, err := r.upload(ctx, store, db, key)
		if err != nil {
			return err
		}
		log.Info("backup uploaded", "bytes", n, "duration", time.Since(start).Round(time.Millisecond))
	}

	if vno.Keep > 0 {
		store, err := r.storage(ctx, s3cfg)
		if err != nil {
			return err
		}
		if err := r.prune(ctx, store, s3cfg.Prefix, vno, vars, key, log); err != nil {
			return fmt.Errorf("prune: %w", err)
		}
	}
	return nil
}

// upload streams the dump straight into storage. A dump error closes the pipe
// with that error, which makes Put abort without creating an object.
func (r *Runner) upload(ctx context.Context, store Storage, db config.Database, key string) (int64, error) {
	pr, pw := io.Pipe()
	cw := &countingWriter{w: pw}
	dumpErr := make(chan error, 1)
	go func() {
		err := r.Dumper.Dump(ctx, db, cw)
		pw.CloseWithError(err) // nil closes normally
		dumpErr <- err
	}()

	putErr := store.Put(ctx, key, pr)
	// Unblock the dumper if Put stopped reading early.
	_ = pr.CloseWithError(errors.New("upload stopped"))

	if err := <-dumpErr; err != nil {
		return 0, err
	}
	if putErr != nil {
		return 0, putErr
	}
	return cw.n, nil
}

func (r *Runner) storage(ctx context.Context, cfg config.S3) (Storage, error) {
	if s, ok := r.stores[cfg]; ok {
		return s, nil
	}
	s, err := r.NewStorage(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if r.stores == nil {
		r.stores = map[config.S3]Storage{}
	}
	r.stores[cfg] = s
	return s, nil
}

type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}
