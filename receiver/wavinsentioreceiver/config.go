package wavinsentioreceiver

import (
	"fmt"
	"time"

	"go.opentelemetry.io/collector/scraper/scraperhelper"
)

const (
	MinCollectionInterval = 60 * time.Second
)

type Config struct {
	scraperhelper.ControllerConfig `mapstructure:",squash"`
	Endpoint                       string `mapstructure:"endpoint"`
	Username                       string `mapstructure:"username"`
	Password                       string `mapstructure:"password"`
	WebApiKey                      string `mapstructure:"web_api_key"`
}

func (cfg Config) Validate() error {
	if cfg.CollectionInterval < MinCollectionInterval {
		return fmt.Errorf("interval must be at least %s", MinCollectionInterval)
	}
	if cfg.Endpoint == "" {
		return fmt.Errorf("endpoint is required")
	}
	if cfg.Username == "" {
		return fmt.Errorf("username is required")
	}
	if cfg.Password == "" {
		return fmt.Errorf("password is required")
	}
	if cfg.WebApiKey == "" {
		return fmt.Errorf("web_api_key is required")
	}
	return nil
}
