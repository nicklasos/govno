//go:build integration

package backup_test

import (
	"compress/gzip"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/nicklasos/govno/v2/internal/backup"
	"github.com/nicklasos/govno/v2/internal/config"
	"github.com/nicklasos/govno/v2/internal/dump"
	"github.com/nicklasos/govno/v2/internal/storage"
)

// TestEndToEnd dumps a real MySQL database into a real S3-compatible bucket.
//
//	GOVNO_IT_MYSQL_CNF   mysql client options file (user, password, host, port)
//	GOVNO_IT_MYSQL_DB    database to dump (must exist)
//	GOVNO_IT_S3_ENDPOINT e.g. http://127.0.0.1:9000
//	GOVNO_IT_S3_BUCKET   bucket (must exist)
//	AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY / AWS_REGION
func TestEndToEnd(t *testing.T) {
	cnf := os.Getenv("GOVNO_IT_MYSQL_CNF")
	endpoint := os.Getenv("GOVNO_IT_S3_ENDPOINT")
	if cnf == "" || endpoint == "" {
		t.Skip("GOVNO_IT_* environment not set")
	}
	ctx := context.Background()

	s3cfg := config.S3{
		Bucket:   os.Getenv("GOVNO_IT_S3_BUCKET"),
		Region:   os.Getenv("AWS_REGION"),
		Endpoint: endpoint,
		Prefix:   "it-" + time.Now().Format("150405") + "/",
	}
	cfg := &config.Config{
		S3: s3cfg,
		Database: []config.Database{{
			Name:         os.Getenv("GOVNO_IT_MYSQL_DB"),
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
		Dumper:     dump.MySQL{},
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
			return aws.Credentials{
				AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
				SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
			}, nil
		}),
	})
}
