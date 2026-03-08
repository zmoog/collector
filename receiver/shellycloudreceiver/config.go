package shellycloudreceiver

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
	// ServerURL is the region-specific Shelly Cloud endpoint,
	// e.g. https://shelly-17-eu.shelly.cloud
	ServerURL string `mapstructure:"server_url"`
	// AuthKey is the API key from the Shelly Cloud account settings.
	AuthKey string `mapstructure:"auth_key"`
}

func (cfg *Config) Validate() error {
	if cfg.CollectionInterval < MinCollectionInterval {
		return fmt.Errorf("collection_interval must be at least %s", MinCollectionInterval)
	}
	if cfg.ServerURL == "" {
		return fmt.Errorf("server_url is required")
	}
	if cfg.AuthKey == "" {
		return fmt.Errorf("auth_key is required")
	}
	return nil
}
