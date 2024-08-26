package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

type WorkMode string

const (
	ALWAYS WorkMode = "always"
	NEVER  WorkMode = "never"
	DRYRUN WorkMode = "dryrun"
)

type SkuBinding struct {
	SkuID     string `json:"sku_id"`
	ProductID string `json:"product_id"`
}

type Config struct {
	SkuBindings []SkuBinding `json:"sku_bindings"`
	Mode        WorkMode     `json:"mode"`
	Port        int          `json:"port"`

	endpoint string `json:"endpoint"`
}

func NewConfig(data string, port string, endpoint string, mode WorkMode) (*Config, error) {
	if data == "" {
		data = `[]` // default config
	}
	var bindings []SkuBinding
	if err := json.Unmarshal([]byte(data), &bindings); err != nil {
		return nil, err
	}

	err := validateMode(mode)
	if err != nil {
		return nil, err
	}
	if mode == "" {
		mode = ALWAYS
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
		SkuBindings: bindings,
		Mode:        mode,
		endpoint:    endpoint,
		Port:        p,
	}

	return config, nil
}

func (c Config) Address() string {
	return fmt.Sprintf("%s:%d", c.endpoint, c.Port)
}

func FromEnv() (*Config, error) {
	port := os.Getenv("PORT")
	bindings := os.Getenv("BINDINGS")
	endpoint := os.Getenv("ENDPOINT")
	mode := WorkMode(os.Getenv("MODE"))

	return NewConfig(bindings, port, endpoint, mode)
}

func validateMode(mode WorkMode) error {
	switch mode {
	case ALWAYS, DRYRUN, NEVER, "":
		return nil
	default:
		return fmt.Errorf("invalid mode")
	}
}
