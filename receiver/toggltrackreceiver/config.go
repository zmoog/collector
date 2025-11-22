package toggltrackreceiver

import (
	"fmt"
	"time"

	"go.opentelemetry.io/collector/scraper/scraperhelper"
)

const (
	MinCollectionInterval = 1 * time.Minute
	MinLookback           = 1 * time.Hour
)

// mapping is a map of an ID to a name.
type Mapping map[string]string

// mappings is a collection of mappings for workspace,
// project, and task IDs to names.
type Mappings struct {
	Workspaces Mapping `mapstructure:"workspaces"`
	Projects   Mapping `mapstructure:"projects"`
	Tasks      Mapping `mapstructure:"tasks"`
}

type Config struct {
	scraperhelper.ControllerConfig `mapstructure:",squash"`
	Lookback                       string   `mapstructure:"lookback"`
	APIToken                       string   `mapstructure:"api_token"`
	Mappings                       Mappings `mapstructure:"mappings"`
}

func (cfg *Config) Validate() error {
	if cfg.CollectionInterval < MinCollectionInterval {
		return fmt.Errorf("collection_interval must be at least %s", MinCollectionInterval)
	}

	lookback, err := time.ParseDuration(cfg.Lookback)
	if err != nil {
		return fmt.Errorf("invalid lookback duration: %w", err)
	}
	if lookback < MinLookback {
		return fmt.Errorf("lookback must be at least %s", MinLookback)
	}

	if cfg.APIToken == "" {
		return fmt.Errorf("api_token is required")
	}

	return nil
}
