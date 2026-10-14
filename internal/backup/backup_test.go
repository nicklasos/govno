package backup

import (
	"bytes"
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nicklasos/govno/v2/internal/config"
	"github.com/nicklasos/govno/v2/internal/storage"
)

type fakeDumper struct {
	fail map[string]error
}

func (d fakeDumper) Dump(_ context.Context, db config.Database, w io.Writer) error {
	if _, err := io.WriteString(w, "dump of "+db.Name); err != nil {
		return err
	}
	return d.fail[db.Name]
}

type fakeStore struct {
	mu      sync.Mutex
	objects map[string]storage.Object
	data    map[string]string
	deleted []string
	putErr  error
}

func newFakeStore() *fakeStore {
	return &fakeStore{objects: map[string]storage.Object{}, data: map[string]string{}}
}

func (s *fakeStore) Put(_ context.Context, key string, r io.Reader) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err // a real upload aborts here, storing nothing
	}
	if s.putErr != nil {
		return s.putErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[key] = storage.Object{Key: key, LastModified: time.Now()}
	s.data[key] = string(b)
	return nil
}

func (s *fakeStore) List(_ context.Context, prefix string) ([]storage.Object, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []storage.Object
	for k, o := range s.objects {
		if strings.HasPrefix(k, prefix) {
			out = append(out, o)
		}
	}
	return out, nil
}

func (s *fakeStore) Delete(_ context.Context, keys []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, k := range keys {
		delete(s.objects, k)
		s.deleted = append(s.deleted, k)
	}
	return nil
}

func (s *fakeStore) add(key string, age time.Duration) {
	s.objects[key] = storage.Object{Key: key, LastModified: time.Now().Add(-age)}
}

var fixedNow = time.Date(2026, time.March, 5, 22, 0, 0, 0, time.UTC) // Thursday

func testConfig() *config.Config {
	return &config.Config{
		S3: config.S3{Bucket: "b", Prefix: "mysql/"},
		Database: []config.Database{
			{
				Name: "shop",
				Vno: []config.Vno{
					{Name: "daily", Path: "{db}/daily/{weekday}.sql.gz"},
					{Name: "nightly", Path: "{db}/nightly/{date}.sql.gz", Keep: 3},
				},
			},
			{
				Name: "blog",
				Vno:  []config.Vno{{Name: "daily", Path: "{db}/{hostname}/{day}.sql.gz"}},
			},
		},
	}
}

func newRunner(store *fakeStore, d Dumper) *Runner {
	return &Runner{
		Dumper:     d,
		NewStorage: func(context.Context, config.S3) (Storage, error) { return store, nil },
		Hostname:   "web-1",
		Now:        func() time.Time { return fixedNow },
	}
}

func TestRun(t *testing.T) {
	store := newFakeStore()
	if err := newRunner(store, fakeDumper{}).Run(context.Background(), testConfig(), "daily", ""); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"mysql/shop/daily/4.sql.gz":  "dump of shop",
		"mysql/blog/web-1/05.sql.gz": "dump of blog",
	}
	if len(store.data) != len(want) {
		t.Fatalf("stored %v, want %v", store.data, want)
	}
	for k, v := range want {
		if store.data[k] != v {
			t.Errorf("object %q = %q, want %q", k, store.data[k], v)
		}
	}
}

func TestRunDBFilter(t *testing.T) {
	store := newFakeStore()
	if err := newRunner(store, fakeDumper{}).Run(context.Background(), testConfig(), "daily", "blog"); err != nil {
		t.Fatal(err)
	}
	if len(store.data) != 1 || store.data["mysql/blog/web-1/05.sql.gz"] == "" {
		t.Errorf("stored %v", store.data)
	}
}

func TestRunUnknownVno(t *testing.T) {
	err := newRunner(newFakeStore(), fakeDumper{}).Run(context.Background(), testConfig(), "dialy", "")
	if err == nil || !strings.Contains(err.Error(), `"dialy"`) {
		t.Errorf("expected unknown vno error, got %v", err)
	}
}

func TestRunDumpFailureStoresNothingAndContinues(t *testing.T) {
	store := newFakeStore()
	dumpErr := errors.New("access denied")
	err := newRunner(store, fakeDumper{fail: map[string]error{"shop": dumpErr}}).
		Run(context.Background(), testConfig(), "daily", "")

	if !errors.Is(err, dumpErr) {
		t.Fatalf("expected dump error, got %v", err)
	}
	if _, ok := store.data["mysql/shop/daily/4.sql.gz"]; ok {
		t.Error("failed dump was uploaded")
	}
	if _, ok := store.data["mysql/blog/web-1/05.sql.gz"]; !ok {
		t.Error("other database was not backed up after a failure")
	}
}

func TestRunPutFailure(t *testing.T) {
	store := newFakeStore()
	store.putErr = errors.New("bucket gone")
	err := newRunner(store, fakeDumper{}).Run(context.Background(), testConfig(), "daily", "")
	if !errors.Is(err, store.putErr) {
		t.Errorf("expected put error, got %v", err)
	}
}

func TestPrune(t *testing.T) {
	store := newFakeStore()
	store.add("mysql/shop/nightly/2026-03-04.sql.gz", 1*24*time.Hour)
	store.add("mysql/shop/nightly/2026-03-03.sql.gz", 2*24*time.Hour)
	store.add("mysql/shop/nightly/2026-03-02.sql.gz", 3*24*time.Hour)
	store.add("mysql/shop/nightly/2026-03-01.sql.gz", 4*24*time.Hour)
	store.add("mysql/shop/nightly/notes.txt", 100*24*time.Hour)       // not matching template
	store.add("mysql/shop/daily/1.sql.gz", 100*24*time.Hour)          // other vno
	store.add("mysql/blog/nightly/2020-01-01.sql.gz", 1000*time.Hour) // other db

	if err := newRunner(store, fakeDumper{}).Run(context.Background(), testConfig(), "nightly", ""); err != nil {
		t.Fatal(err)
	}

	slices.Sort(store.deleted)
	want := []string{"mysql/shop/nightly/2026-03-01.sql.gz", "mysql/shop/nightly/2026-03-02.sql.gz"}
	if !slices.Equal(store.deleted, want) {
		t.Errorf("deleted %v, want %v", store.deleted, want)
	}
	if _, ok := store.objects["mysql/shop/nightly/2026-03-05.sql.gz"]; !ok {
		t.Error("current backup missing")
	}
}

func TestPruneNeverDeletesCurrent(t *testing.T) {
	store := newFakeStore()
	// An existing object with the same key and a newer timestamp (clock skew).
	store.add("mysql/shop/nightly/2026-03-05.sql.gz", -time.Hour)
	cfg := testConfig()
	cfg.Database[0].Vno[1].Keep = 1

	if err := newRunner(store, fakeDumper{}).Run(context.Background(), cfg, "nightly", ""); err != nil {
		t.Fatal(err)
	}
	if len(store.deleted) != 0 {
		t.Errorf("deleted %v", store.deleted)
	}
}

func TestDryRun(t *testing.T) {
	store := newFakeStore()
	store.add("mysql/shop/nightly/2026-03-04.sql.gz", 1*24*time.Hour)
	store.add("mysql/shop/nightly/2026-03-03.sql.gz", 2*24*time.Hour)
	store.add("mysql/shop/nightly/2026-03-02.sql.gz", 3*24*time.Hour)

	var logs bytes.Buffer
	r := newRunner(store, fakeDumper{fail: map[string]error{"shop": errors.New("must not dump")}})
	r.DryRun = true
	r.Logger = slogText(&logs)

	if err := r.Run(context.Background(), testConfig(), "nightly", ""); err != nil {
		t.Fatal(err)
	}
	if len(store.data) != 0 || len(store.deleted) != 0 {
		t.Errorf("dry run modified storage: data=%v deleted=%v", store.data, store.deleted)
	}
	if !strings.Contains(logs.String(), "2026-03-02.sql.gz") {
		t.Errorf("dry run did not report pruning:\n%s", logs.String())
	}
}
