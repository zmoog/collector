package zcsazzurroreceiver

import (
	"fmt"
	"time"

	"go.opentelemetry.io/collector/scraper/scraperhelper"
)

const (
	MinCollectionInterval = 30 * time.Second
)

type Config struct {
	scraperhelper.ControllerConfig `mapstructure:",squash"`
	ClientID                       string `mapstructure:"client_id"`
	AuthKey                        string `mapstructure:"auth_key"`
	ThingKey                       string `mapstructure:"thing_key"`
}

func (cfg *Config) Validate() error {
	if cfg.CollectionInterval < MinCollectionInterval {
		// ZCS updates data every 5 minutes, so it makes no sense
		// to have a smaller interval.
		//
		// However, having a smaller interval comes handy
		// when testing.
		//
		// The receiver checks the last update time
		// and only processes metrics if the last update
		// is newer than the last processed time.
		//
		// So, having a smaller interval than the update
		// interval is not a problem.
		return fmt.Errorf("collection_interval must be at least %s", MinCollectionInterval)
	}

	if cfg.AuthKey == "" {
		return fmt.Errorf("auth_key is required")
	}
	if cfg.ClientID == "" {
		return fmt.Errorf("client_id is required")
	}
	if cfg.ThingKey == "" {
		return fmt.Errorf("thing_key is required")
	}
	return nil
}
