package config

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/go-yaml/yaml"
)

type Config struct {
	Server struct {
		IP   string `yaml:"IP"`
		Port int    `yaml:"Port"`
	} `yaml:"server"`

	Database struct {
		Username string `yaml:"Username"`
		Password string `yaml:"Password"`
		Network  string `yaml:"Network"`
		Server   string `yaml:"Server"`
		Port     int    `yaml:"Port"`
		Database string `yaml:"Database"`
	} `yaml:"database"`
}

var config Config

func init() {
	configFile := "config.yaml"
	config, err := loadConfigFromYAML(configFile)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	fmt.Printf("Server config: %+v\n", config.Server)
	fmt.Printf("Database config: %+v\n", config.Database)
}

func loadConfigFromYAML(filename string) (Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		// workaround
		data, err = ioutil.ReadFile(filepath.Join("..", filename))
		if err != nil {
			return config, err
		}
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}

func GetConfig() Config {
	return config
}
