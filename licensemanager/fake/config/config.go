package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

type Lock struct {
	TemplateID string `json:"template_id"`
	ResourceID string `json:"resource_id"`
}

type Config struct {
	Locks []Lock `json:"locks"`
	Port  int    `json:"port"`

	endpoint string
}

func NewConfig(data string, port string, endpoint string) (*Config, error) {
	if data == "" {
		data = `[]` // default config
	}
	var locks []Lock
	if err := json.Unmarshal([]byte(data), &locks); err != nil {
		return nil, err
	}

	if endpoint == "" {
		endpoint = "api.yc.local"
	}

	if port == "" {
		port = "8080"
	}

	p, err := strconv.Atoi(port)
	if err != nil {
		return nil, err
	}

	config := &Config{
		Locks:    locks,
		endpoint: endpoint,
		Port:     p,
	}

	return config, nil
}

func (c Config) Address() string {
	return fmt.Sprintf("%s:%d", c.endpoint, c.Port)
}

func FromEnv() (*Config, error) {
	port := os.Getenv("PORT")
	bindings := os.Getenv("LOCKS")
	endpoint := os.Getenv("ENDPOINT")

	return NewConfig(bindings, port, endpoint)
}
