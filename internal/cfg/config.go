package cfg

import (
	"errors"
	"log"

	"github.com/spf13/viper"

	"maileryze/internal/types"
)

type AppConfig struct {
	Providers []types.EmailProvider `mapstructure:"providers"`
}

func Load() *AppConfig {
	var c AppConfig
	if err := viper.Unmarshal(&c); err != nil {
		log.Fatalf("unable to decode config into struct, %v", err)
	}
	return &c
}

func ProviderByAlias(alias string) (*types.EmailProvider, error) {
	c := Load()
	for _, p := range c.Providers {
		if p.Alias == alias {
			return &p, nil
		}
	}
	return nil, errors.New("provider not found")
}
