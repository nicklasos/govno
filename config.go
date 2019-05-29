package main

import "github.com/BurntSushi/toml"

type Vno struct {
	Name string
	Path string
}

type Database struct {
	Name      string
	Host      string
	Cnf       string
	AwsBucket string `toml:"aws_bucket"`
	AwsId     string `toml:"aws_id"`
	AwsKey    string `toml:"aws_key"`
	AwsRegion string `toml:"aws_region"`
	Vno       []Vno
}

type Config struct {
	Database []Database
}

func ReadConfig(configFile string) (*Config, error) {
	var config Config
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
