package main

import (
	"testing"
)

func TestReadConfigError(t *testing.T) {
	_, err := ReadConfig(".not-existed-config")
	if err == nil {
		t.Error("It should be an error on read non existed file")
	}
}

func TestReadConfig(t *testing.T) {
	config, err := ReadConfig("govno.toml")

	// fmt.Printf("%v+\n", config.Database)

	if err != nil {
		t.Error(err)
	}

	if len(config.Database) == 0 {
		t.Error("Config length is zero")
	}

	if config.Database[0].Name != "database_name" {
		t.Error("Parse database name is not working")
	}
}
