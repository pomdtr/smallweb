package utils

import "github.com/knadh/koanf/v2"

type Config struct {
	*koanf.Koanf
}

func NewConfig() *Config {
	return &Config{
		Koanf: koanf.New("."),
	}
}

func (c *Config) Reset() {
	c.Koanf = koanf.New(".")
}
