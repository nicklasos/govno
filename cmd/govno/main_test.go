package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	var out bytes.Buffer
	if code := run(context.Background(), []string{"-version"}, &out, io.Discard); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "govno dev") {
		t.Errorf("output %q", out.String())
	}
}

func TestRunUsage(t *testing.T) {
	var out bytes.Buffer
	if code := run(context.Background(), nil, io.Discard, &out); code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
	if !strings.Contains(out.String(), "Usage: govno") {
		t.Errorf("output %q", out.String())
	}
}

func TestRunDryRunHidesSecrets(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "govno.toml")
	err := os.WriteFile(cfg, []byte(`
[s3]
bucket = "b"
region = "us-east-1"
access_key_id = "AKIATESTKEY"
secret_access_key = "super-secret-value"

[[database]]
name = "shop"
defaults_file = "shop.cnf"
  [[database.vno]]
  name = "daily"
  path = "{db}/{weekday}.sql.gz"
`), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if code := run(context.Background(), []string{"-config", cfg, "-dry-run", "daily"}, io.Discard, &out); code != 0 {
		t.Fatalf("exit %d: %s", code, out.String())
	}
	if !strings.Contains(out.String(), "would upload") {
		t.Errorf("missing dry run output: %s", out.String())
	}
	if strings.Contains(out.String(), "super-secret-value") || strings.Contains(out.String(), "AKIATESTKEY") {
		t.Errorf("credentials leaked to logs: %s", out.String())
	}
}

func TestRunUnknownVnoFails(t *testing.T) {
	var out bytes.Buffer
	code := run(context.Background(), []string{"-config", "../../examples/govno.toml", "-dry-run", "nope"}, io.Discard, &out)
	if code != 1 {
		t.Errorf("exit %d, want 1: %s", code, out.String())
	}
}
