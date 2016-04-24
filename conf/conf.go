package conf

import (
	//"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
}

func New() *Config {
	return &Config{}
}

func Load(path string) (*Config, error) {
	b, readErr := ioutil.ReadFile(path)
	if readErr != nil {
		return nil, readErr
	}
	c := New()
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) Dump() ([]byte, error) {
	b, err := yaml.Marshal(c)
	if err != nil {
		return nil, err
	}
	return b, nil
}
