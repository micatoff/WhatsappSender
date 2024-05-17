package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	TgApiToken string `yaml:"tg_api_token"`
}

func New(path string) (*Config, error) {
	config := &Config{}

	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

 	if	err = yaml.Unmarshal(file, config); err != nil {
	 return nil, err
	}
	
	return config, nil
}
