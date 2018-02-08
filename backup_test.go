package main

import (
	"testing"
)

type MockStorage struct {
	Uploaded bool
}

func (s *MockStorage) Upload(bucket, file, dest string) error {
	s.Uploaded = true

	return nil
}

type MockBackup struct {
	Created bool
	Cleared bool
}

func (b *MockBackup) Create(db, file string) error {
	if db == "database_name" {
		b.Created = true
	}

	return nil
}

func (b *MockBackup) Clear(file string) error {
	b.Cleared = true

	return nil
}

func TestBackupDatabase(t *testing.T) {
	config, _ := ReadConfig("govno.toml")

	mockStorage := &MockStorage{false}
	mockBackup := &MockBackup{false, false}

	BackupDatabase(config.Database, "daily", mockStorage, mockBackup)

	if !mockStorage.Uploaded {
		t.Error("Error in Upload")
	}

	if !mockBackup.Created {
		t.Error("Error in Create")
	}

	if !mockBackup.Cleared {
		t.Error("Error in Clear")
	}
}
