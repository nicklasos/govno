package main

import (
	"os"

	"github.com/codeskyblue/go-sh"
)

type MySQLDump struct{}

func (d *MySQLDump) Create(db, file string) error {
	err := sh.Command(
		"mysqldump",
		"--defaults-file=config.cnf",
		"--single-transaction",
		db,
	).Command("gzip", "-9").WriteStdout(file)

	if err != nil {
		return err
	}

	return nil
}

func (d *MySQLDump) Clear(file string) error {
	return os.Remove(file)
}
