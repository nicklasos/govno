package main

type Storage interface {
	Upload(bucket, file, dest string) error
}

type Dump interface {
	Create(db, file string) error
}

func BackupDatabase(database []Database, vnoName string, storage Storage, dump Dump) error {
	for _, db := range database {
		for _, vno := range db.Vno {
			if vno.Name == vnoName {
				path, err := Convert(vno.Path)
				if err != nil {
					return err
				}

				tmpBackupFile := "dump.sql.gz"

				err = dump.Create(db.Name, tmpBackupFile)
				if err != nil {
					return err
				}

				err = storage.Upload(db.Bucket, tmpBackupFile, path)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
