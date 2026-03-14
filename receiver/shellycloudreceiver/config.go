package shellycloudreceiver

import (
	"fmt"
	"time"

	"go.opentelemetry.io/collector/scraper/scraperhelper"
)

const (
	MinCollectionInterval  = 60 * time.Second
	DefaultRequestDelay    = 500 * time.Millisecond
)

type Config struct {
	scraperhelper.ControllerConfig `mapstructure:",squash"`
	// ServerURL is the region-specific Shelly Cloud endpoint,
	// e.g. https://shelly-68-eu.shelly.cloud
	ServerURL string `mapstructure:"server_url"`
	// AuthKey is the API key from the Shelly Cloud account settings.
	AuthKey string `mapstructure:"auth_key"`
	// RequestDelay is the pause between consecutive device status API calls
	// to avoid hitting Shelly Cloud rate limits. Defaults to 500ms.
	RequestDelay time.Duration `mapstructure:"request_delay"`
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
