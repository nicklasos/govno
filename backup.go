package main

import (
	"time"
	"fmt"
)

type Storage interface {
	Upload(bucket, file, dest string) error
}

type Dump interface {
	Create(db, file string) error
	Clear(file string) error
}

func BackupDatabase(database []Database, vnoName string, storage Storage, dump Dump) error {
	for _, db := range database {
		for _, vno := range db.Vno {
			if vno.Name == vnoName {
				path, err := Convert(vno.Path)
				if err != nil {
					return err
				}

				tmpBackupFile := fmt.Sprintf("%d.%s.%s.dump.sql.gz", time.Now().UnixNano(), vnoName, db.Name)

				err = dump.Create(db.Name, tmpBackupFile)
				if err != nil {
					return err
				}
				defer dump.Clear(tmpBackupFile)

				err = storage.Upload(db.Bucket, tmpBackupFile, path)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
