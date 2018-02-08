package main

import (
	"os"
	"log"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("You should provide vno mane as first parameter")
	}

	vno := os.Args[1]
	file := os.Getenv("HOME") + "/.govno"

	if len(os.Args) > 2 {
		file = os.Args[2]
	}

	config, err := ReadConfig(file)
	if err != nil {
		log.Fatal(err)
	}

	err = BackupDatabase(config.Database, vno, &AwsStorage{}, &MySQLDump{})
	if err != nil {
		log.Fatal(err)
	}
}
