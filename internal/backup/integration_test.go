//go:build integration

package backup_test

import (
	"compress/gzip"
	"context"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"

	"github.com/nicklasos/govno/v2/internal/backup"
	"github.com/nicklasos/govno/v2/internal/config"
	"github.com/nicklasos/govno/v2/internal/dump"
	"github.com/nicklasos/govno/v2/internal/storage"
)

// TestEndToEnd runs dump -> upload -> prune against an S3 API and mysqldump.
//
// By default it needs no external services: S3 is an in-process fake
// (gofakes3) and mysqldump is a stub script. Set these to use real ones:
//
//	GOVNO_IT_MYSQL_CNF   mysql client options file (user, password, host, port)
//	GOVNO_IT_MYSQL_DB    database to dump (must exist)
//	GOVNO_IT_S3_ENDPOINT S3-compatible endpoint, e.g. http://127.0.0.1:9000
//	GOVNO_IT_S3_BUCKET   bucket (must exist)
//	AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY / AWS_REGION
func TestEndToEnd(t *testing.T) {
	ctx := context.Background()

	s3cfg := config.S3{
		Bucket:          os.Getenv("GOVNO_IT_S3_BUCKET"),
		Region:          os.Getenv("AWS_REGION"),
		Endpoint:        os.Getenv("GOVNO_IT_S3_ENDPOINT"),
		AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		Prefix:          "it-" + time.Now().Format("150405") + "/",
	}
	if s3cfg.Endpoint == "" {
		s3cfg = fakeS3(t, s3cfg.Prefix)
	}

	dumper := dump.MySQL{}
	cnf, dbName := os.Getenv("GOVNO_IT_MYSQL_CNF"), os.Getenv("GOVNO_IT_MYSQL_DB")
	if cnf == "" {
		cnf, dbName = "unused.cnf", "govno_it"
		dumper.Binary = fakeMysqldump(t)
	}

	cfg := &config.Config{
		S3: s3cfg,
		Database: []config.Database{{
			Name:         dbName,
			DefaultsFile: cnf,
			Vno:          []config.Vno{{Name: "hourly", Path: "{db}/{seconds}.sql.gz", Keep: 2}},
		}},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}

	store, err := storage.NewS3(ctx, s3cfg)
	if err != nil {
		t.Fatal(err)
	}
	runner := &backup.Runner{
		Dumper:     dumper,
		NewStorage: func(context.Context, config.S3) (backup.Storage, error) { return store, nil },
	}

	// Three runs with distinct keys; keep = 2 must leave two objects.
	for range 3 {
		if err := runner.Run(ctx, cfg, "hourly", ""); err != nil {
			t.Fatal(err)
		}
		time.Sleep(1100 * time.Millisecond)
	}

	objects, err := store.List(ctx, s3cfg.Prefix)
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != 2 {
		t.Fatalf("got %d objects after pruning, want 2: %v", len(objects), objects)
	}

	client := s3Client(t, s3cfg)
	out, err := client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s3cfg.Bucket), Key: aws.String(objects[0].Key)})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = out.Body.Close() }()
	gz, err := gzip.NewReader(out.Body)
	if err != nil {
		t.Fatal(err)
	}
	sql, err := io.ReadAll(gz)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(sql), "MySQL dump") {
		t.Errorf("object does not look like a mysqldump:\n%.300s", sql)
	}

	// A failing dump must not create an object.
	bad := *cfg
	bad.Database = []config.Database{cfg.Database[0]}
	bad.Database[0].Name = "does_not_exist"
	if err := runner.Run(ctx, &bad, "hourly", ""); err == nil {
		t.Fatal("expected error for missing database")
	}
	objects, _ = store.List(ctx, s3cfg.Prefix+"does_not_exist/")
	if len(objects) != 0 {
		t.Errorf("failed dump created objects: %v", objects)
	}
}

func s3Client(t *testing.T, c config.S3) *s3.Client {
	t.Helper()
	return s3.New(s3.Options{
		Region:       c.Region,
		BaseEndpoint: aws.String(c.Endpoint),
		UsePathStyle: true,
		Credentials: aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
			return aws.Credentials{AccessKeyID: c.AccessKeyID, SecretAccessKey: c.SecretAccessKey}, nil
		}),
	})
}

// fakeS3 starts an in-memory S3 server with one bucket.
func fakeS3(t *testing.T, prefix string) config.S3 {
	t.Helper()
	backend := s3mem.New()
	const bucket = "govno-it"
	if err := backend.CreateBucket(bucket); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(gofakes3.New(backend).Server())
	t.Cleanup(srv.Close)
	return config.S3{
		Bucket:          bucket,
		Region:          "us-east-1",
		Endpoint:        srv.URL,
		AccessKeyID:     "test",
		SecretAccessKey: "test",
		Prefix:          prefix,
	}
}

// fakeMysqldump writes a stub that prints a dump header for any database
// except "does_not_exist", for which it fails like the real mysqldump.
func fakeMysqldump(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("stub mysqldump requires a POSIX shell")
	}
	path := filepath.Join(t.TempDir(), "mysqldump")
	script := `#!/bin/sh
for db; do :; done
if [ "$db" = "does_not_exist" ]; then
  echo "mysqldump: Got error: 1049: Unknown database '$db'" >&2
  exit 2
fi
echo "-- MySQL dump 10.13 (stub) of $db"
echo "CREATE TABLE t (id INT);"
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
