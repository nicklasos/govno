// Package config loads and validates the govno TOML configuration.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/nicklasos/govno/v2/internal/pathtpl"
)

// S3 describes where backups are stored. Empty credentials mean the AWS
// default credential chain (env, shared config, IAM role) is used.
type S3 struct {
	Bucket          string `toml:"bucket"`
	Region          string `toml:"region"`
	Prefix          string `toml:"prefix"`
	Endpoint        string `toml:"endpoint"`
	AccessKeyID     string `toml:"access_key_id"`
	SecretAccessKey string `toml:"secret_access_key"`
}

// String keeps credentials out of logs.
func (s S3) String() string {
	return fmt.Sprintf("s3://%s/%s (region=%s endpoint=%s)", s.Bucket, s.Prefix, s.Region, s.Endpoint)
}

// Vno is one backup schedule of a database, e.g. "daily".
type Vno struct {
	Name string `toml:"name"`
	Path string `toml:"path"`
	Keep int    `toml:"keep"`
}

// Database is a MySQL database to back up.
type Database struct {
	Name          string   `toml:"name"`
	DefaultsFile  string   `toml:"defaults_file"`
	Host          string   `toml:"host"`
	MysqldumpArgs []string `toml:"mysqldump_args"`
	S3            *S3      `toml:"s3"`
	Vno           []Vno    `toml:"vno"`
}

// Config is the root of the configuration file.
type Config struct {
	S3       S3         `toml:"s3"`
	Database []Database `toml:"database"`
}

// legacyKeys are v1 config keys, used to give a helpful migration error.
var legacyKeys = map[string]bool{
	"cnf": true, "aws_bucket": true, "aws_id": true, "aws_key": true, "aws_region": true,
}

// Load reads, normalizes and validates a config file. Relative defaults_file
// paths are resolved against the config file's directory.
func Load(path string) (*Config, error) {
	var cfg Config
	md, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	if undecoded := md.Undecoded(); len(undecoded) > 0 {
		keys := make([]string, 0, len(undecoded))
		legacy := false
		for _, k := range undecoded {
			keys = append(keys, k.String())
			if legacyKeys[k[len(k)-1]] {
				legacy = true
			}
		}
		msg := fmt.Sprintf("config %s: unknown keys: %s", path, strings.Join(keys, ", "))
		if legacy {
			msg += " (this looks like a govno v1 config, see \"Migrating from v1\" in the README)"
		}
		return nil, errors.New(msg)
	}

	dir := filepath.Dir(path)
	for i := range cfg.Database {
		db := &cfg.Database[i]
		if db.DefaultsFile != "" && !filepath.IsAbs(db.DefaultsFile) {
			db.DefaultsFile = filepath.Join(dir, db.DefaultsFile)
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config %s: %w", path, err)
	}
	return &cfg, nil
}

// StorageFor returns the effective S3 settings of db: the global [s3]
// section with non-empty per-database fields applied on top.
func (c *Config) StorageFor(db Database) S3 {
	s := c.S3
	if o := db.S3; o != nil {
		if o.Bucket != "" {
			s.Bucket = o.Bucket
		}
		if o.Region != "" {
			s.Region = o.Region
		}
		if o.Prefix != "" {
			s.Prefix = o.Prefix
		}
		if o.Endpoint != "" {
			s.Endpoint = o.Endpoint
		}
		if o.AccessKeyID != "" {
			s.AccessKeyID = o.AccessKeyID
			s.SecretAccessKey = o.SecretAccessKey
		}
	}
	return s
}

// Validate reports every problem in the config at once.
func (c *Config) Validate() error {
	var errs []error
	if len(c.Database) == 0 {
		errs = append(errs, errors.New("no [[database]] sections"))
	}
	seenDB := map[string]bool{}
	for i, db := range c.Database {
		where := fmt.Sprintf("database[%d]", i)
		if db.Name == "" {
			errs = append(errs, fmt.Errorf("%s: name is required", where))
		} else {
			where = fmt.Sprintf("database %q", db.Name)
			if seenDB[db.Name] {
				errs = append(errs, fmt.Errorf("%s: duplicate name", where))
			}
			seenDB[db.Name] = true
		}
		if db.DefaultsFile == "" {
			errs = append(errs, fmt.Errorf("%s: defaults_file is required", where))
		}
		s := c.StorageFor(db)
		if s.Bucket == "" {
			errs = append(errs, fmt.Errorf("%s: s3 bucket is required", where))
		}
		if (s.AccessKeyID == "") != (s.SecretAccessKey == "") {
			errs = append(errs, fmt.Errorf("%s: set both access_key_id and secret_access_key, or neither", where))
		}
		if len(db.Vno) == 0 {
			errs = append(errs, fmt.Errorf("%s: no [[database.vno]] sections", where))
		}
		seenVno := map[string]bool{}
		for _, v := range db.Vno {
			if v.Name == "" {
				errs = append(errs, fmt.Errorf("%s: vno name is required", where))
				continue
			}
			if seenVno[v.Name] {
				errs = append(errs, fmt.Errorf("%s: duplicate vno %q", where, v.Name))
			}
			seenVno[v.Name] = true
			if err := pathtpl.Validate(v.Path); err != nil {
				errs = append(errs, fmt.Errorf("%s vno %q: %w", where, v.Name, err))
			}
			if v.Keep < 0 {
				errs = append(errs, fmt.Errorf("%s vno %q: keep must be >= 0", where, v.Name))
			}
		}
	}
	return errors.Join(errs...)
}

// DefaultPath returns the config file to use when -config is not given:
// $GOVNO_CONFIG, then ~/.govno, then /etc/govno/govno.toml.
func DefaultPath() string {
	if p := os.Getenv("GOVNO_CONFIG"); p != "" {
		return p
	}
	if home, err := os.UserHomeDir(); err == nil {
		p := filepath.Join(home, ".govno")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "/etc/govno/govno.toml"
}
