package backup

import (
	"cmp"
	"context"
	"log/slog"
	"slices"
	"strings"

	"github.com/nicklasos/govno/v2/internal/config"
	"github.com/nicklasos/govno/v2/internal/pathtpl"
	"github.com/nicklasos/govno/v2/internal/storage"
)

// prune keeps the newest vno.Keep objects matching the vno's path template,
// counting current (the key just written) as one of them, and deletes the
// rest. current is never deleted.
func (r *Runner) prune(ctx context.Context, store Storage, prefix string, vno config.Vno, vars pathtpl.Vars, current string, log *slog.Logger) error {
	re, err := pathtpl.Pattern(vno.Path, vars)
	if err != nil {
		return err
	}
	objects, err := store.List(ctx, prefix+pathtpl.StaticPrefix(vno.Path))
	if err != nil {
		return err
	}

	var others []storage.Object
	for _, o := range objects {
		rel, ok := strings.CutPrefix(o.Key, prefix)
		if !ok || o.Key == current || !re.MatchString(rel) {
			continue
		}
		others = append(others, o)
	}
	slices.SortFunc(others, func(a, b storage.Object) int {
		if c := b.LastModified.Compare(a.LastModified); c != 0 {
			return c
		}
		return cmp.Compare(b.Key, a.Key)
	})

	keepOthers := vno.Keep - 1
	if len(others) <= keepOthers {
		return nil
	}
	var stale []string
	for _, o := range others[keepOthers:] {
		stale = append(stale, o.Key)
	}

	if r.DryRun {
		log.Info("dry run: would delete old backups", "count", len(stale), "keys", stale)
		return nil
	}
	if err := store.Delete(ctx, stale); err != nil {
		return err
	}
	log.Info("deleted old backups", "count", len(stale), "keys", stale)
	return nil
}
