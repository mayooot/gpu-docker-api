package config

import (
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Port      string `toml:"port"`
	EtcdAddr  string `toml:"etcd_addr"`
	StartPort int    `toml:"start_port"`
	EndPort   int    `toml:"end_port"`
}

func NewConfigWithFile(name string) (*Config, error) {
	cfg, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return NewConfig(string(cfg))
}

func NewConfig(data string) (*Config, error) {
	var c Config
	_, err := toml.Decode(data, &c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
