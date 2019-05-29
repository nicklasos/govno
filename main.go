package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("You should provide vno name as first parameter")
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

	fmt.Println(config)

	err = BackupDatabase(
		config.Database,
		vno,
		&AwsStorage{},
		&MySQLDump{},
	)

	if err != nil {
		log.Fatal(err)
	}
}
