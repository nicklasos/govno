package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	cfg, err := Load("testdata/valid.toml")
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Database) != 2 {
		t.Fatalf("got %d databases, want 2", len(cfg.Database))
	}

	shop := cfg.Database[0]
	if want, _ := filepath.Abs("testdata/shop.cnf"); mustAbs(t, shop.DefaultsFile) != want {
		t.Errorf("relative defaults_file not resolved: %q", shop.DefaultsFile)
	}
	if shop.Vno[1].Keep != 12 {
		t.Errorf("keep = %d, want 12", shop.Vno[1].Keep)
	}
	if s := cfg.StorageFor(shop); s.Bucket != "global-bucket" || s.Prefix != "mysql/" {
		t.Errorf("shop storage = %+v", s)
	}

	blog := cfg.Database[1]
	if blog.DefaultsFile != "/etc/govno/blog.cnf" {
		t.Errorf("absolute defaults_file changed: %q", blog.DefaultsFile)
	}
	s := cfg.StorageFor(blog)
	if s.Bucket != "blog-bucket" || s.Region != "us-west-2" || s.Prefix != "mysql/" || s.AccessKeyID != "AKIA" {
		t.Errorf("blog storage override not merged: %+v", s)
	}
	if strings.Contains(s.String(), "secret") {
		t.Errorf("S3.String leaks credentials: %s", s)
	}
}

func TestLoadMissing(t *testing.T) {
	if _, err := Load("testdata/does-not-exist.toml"); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadLegacy(t *testing.T) {
	_, err := Load("testdata/legacy.toml")
	if err == nil || !strings.Contains(err.Error(), "Migrating from v1") {
		t.Errorf("expected migration hint, got %v", err)
	}
}

func TestLoadInvalid(t *testing.T) {
	_, err := Load("testdata/invalid.toml")
	if err == nil {
		t.Fatal("expected validation error")
	}
	for _, want := range []string{
		"defaults_file is required",
		"s3 bucket is required",
		"unknown placeholder {typo}",
		"keep must be >= 0",
		`duplicate vno "daily"`,
		"duplicate name",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error missing %q:\n%v", want, err)
		}
	}
}

func TestDefaultPathEnv(t *testing.T) {
	t.Setenv("GOVNO_CONFIG", "/tmp/custom.toml")
	if got := DefaultPath(); got != "/tmp/custom.toml" {
		t.Errorf("DefaultPath() = %q", got)
	}
}

func mustAbs(t *testing.T, p string) string {
	t.Helper()
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}
