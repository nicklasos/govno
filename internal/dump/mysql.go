// Package dump produces gzip-compressed database dumps.
package dump

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/nicklasos/govno/v2/internal/config"
)

// MySQL runs mysqldump and gzips its output.
type MySQL struct {
	// Binary is the mysqldump executable; defaults to "mysqldump" from PATH.
	Binary string
}

// Args returns the mysqldump arguments for db.
func Args(db config.Database) []string {
	args := []string{
		"--defaults-file=" + db.DefaultsFile,
		"--single-transaction",
		"--quick",
	}
	if db.Host != "" {
		args = append(args, "--host="+db.Host)
	}
	args = append(args, db.MysqldumpArgs...)
	return append(args, db.Name)
}

// Dump writes a gzip-compressed dump of db to w. It returns an error if
// mysqldump exits non-zero, in which case the written data must be discarded.
func (m MySQL) Dump(ctx context.Context, db config.Database, w io.Writer) error {
	bin := m.Binary
	if bin == "" {
		bin = "mysqldump"
	}

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, bin, Args(db)...)
	cmd.Stderr = &stderr

	gz, err := gzip.NewWriterLevel(w, gzip.BestCompression)
	if err != nil {
		return err
	}
	cmd.Stdout = gz

	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return fmt.Errorf("mysqldump %s: %w: %s", db.Name, err, msg)
		}
		return fmt.Errorf("mysqldump %s: %w", db.Name, err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("gzip %s: %w", db.Name, err)
	}
	return nil
}
