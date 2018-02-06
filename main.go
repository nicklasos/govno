package main

import (
	"os"
	"fmt"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("You should provide vno name as first parameter")

		return
	}

	vno := os.Args[1]
	file := os.Getenv("HOME") + "/.govno"

	if len(os.Args) > 2 {
		file = os.Args[2]
	}

	config, err := ReadConfig(file)
	if err != nil {
		fmt.Println(err)

		return
	}

	Backup(config, vno)
}
