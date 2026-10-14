package dump

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/nicklasos/govno/v2/internal/config"
)

// fakeMysqldump writes a shell script that echoes its arguments to stdout.
func fakeMysqldump(t *testing.T, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake requires a POSIX shell")
	}
	path := filepath.Join(t.TempDir(), "mysqldump")
	script := "#!/bin/sh\n" + body + "\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDump(t *testing.T) {
	bin := fakeMysqldump(t, `echo "$@"`)
	db := config.Database{
		Name:          "shop",
		DefaultsFile:  "/etc/govno/shop.cnf",
		Host:          "db.local",
		MysqldumpArgs: []string{"--routines"},
	}

	var buf bytes.Buffer
	if err := (MySQL{Binary: bin}).Dump(context.Background(), db, &buf); err != nil {
		t.Fatal(err)
	}

	gz, err := gzip.NewReader(&buf)
	if err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(gz)
	if err != nil {
		t.Fatal(err)
	}
	want := "--defaults-file=/etc/govno/shop.cnf --single-transaction --quick --host=db.local --routines shop"
	if got := strings.TrimSpace(string(out)); got != want {
		t.Errorf("args = %q\nwant  %q", got, want)
	}
}

func TestDumpFailure(t *testing.T) {
	bin := fakeMysqldump(t, `echo partial; echo "Access denied" >&2; exit 2`)
	err := (MySQL{Binary: bin}).Dump(context.Background(), config.Database{Name: "shop"}, io.Discard)
	if err == nil {
		t.Fatal("expected error on non-zero exit")
	}
	if !strings.Contains(err.Error(), "Access denied") {
		t.Errorf("stderr not included in error: %v", err)
	}
}
