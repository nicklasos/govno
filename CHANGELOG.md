# Changelog

All notable changes to this project are documented here.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses [Semantic Versioning](https://semver.org/).

## [2.0.0] - 2026-10-14

A full rewrite. The VNO protocol is now production-grade.

### Breaking changes

- **New config format.** The old `cnf` and `aws_*` keys are gone. Storage now lives in a top-level `[s3]` section, with an optional per-database `[database.s3]` override. govno detects a v1 config and points to the migration guide. See [Migrating from v1](README.md#migrating-from-v1).
- **Date placeholders are zero-padded.** `{month}`, `{day}`, `{hours}`, `{minutes}` and `{seconds}` now render as `03` instead of `3`, so keys sort correctly.
- **CLI uses flags.** `govno daily govno.toml` becomes `govno -config govno.toml daily`.
- **Module path** is `github.com/nicklasos/govno/v2`.

### Added

- Retention: `keep = N` on a vno deletes all but the newest N matching backups after each successful run.
- Placeholders `{db}`, `{vno}` and `{date}` (`2006-01-02`).
- Support for S3-compatible storage (MinIO, Cloudflare R2, …) via `endpoint`.
- The AWS default credential chain (env, `~/.aws`, IAM role) is used when no keys are configured.
- Per-database `host` and `mysqldump_args` settings.
- `-db` runs a single database. `-dry-run` previews uploads and deletions.
- Structured logging with `-log-format text|json`. `-version` prints the version.
- Config lookup via `$GOVNO_CONFIG`, then `~/.govno`, then `/etc/govno/govno.toml`.
- Prebuilt binaries for linux/darwin × amd64/arm64, and a Docker image at `ghcr.io/nicklasos/govno`.
- CI (tests, golangci-lint, end-to-end test against MySQL and MinIO).

### Fixed

- The vno `path` template was ignored. Every backup was uploaded to the bucket root under a random temp-file name, so backups never rotated.
- The `cnf` setting was ignored. mysqldump always read `config.cnf` from the current directory, which usually broke under cron.
- A failed mysqldump could still upload an empty or truncated `.gz`. The dump is now streamed to S3, and any mysqldump error aborts the upload.
- AWS credentials were printed to stdout on every run.
- An unknown vno name silently succeeded. It now fails with an error.
- `{week}` returned wrong (even negative) values around New Year.
- If one database failed, the remaining databases were skipped. Now all are attempted, and the exit code is non-zero if any failed.
- Temporary dump files were written to the current directory. No temp files are used anymore.

### Changed

- Migrated from `dep` to Go modules. Requires Go 1.25+.
- Replaced the end-of-life `aws-sdk-go` v1 with `aws-sdk-go-v2`.
- Removed the `go-sh` and external `gzip` dependencies. Compression now uses `compress/gzip`.
- Code reorganized into `cmd/govno` and `internal/*` packages with full test coverage.
