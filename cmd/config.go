package cmd

import (
	"context"
	"time"

	"github.com/sethvargo/go-envconfig"
)

// Config stores app configuration
type Config struct {
	HTTPServerAddress   string        `env:"HTTP_SERVER_ADDRESS,default=0.0.0.0:8080"`
	ReadTimeout         int           `env:"READ_TIMEOUT,default=5"`
	IdleTimeout         int           `env:"IDLE_TIMEOUT,default=30"`
	ShutdownTimeout     int           `env:"SHUTDOWN_TIMEOUT,default=10"`
	Mask                int           `env:"MASK,default=24"`
	RequestsPerInterval uint64        `env:"REQUESTS_PER_INTERVAL,default=10"`
	RequestsInterval    time.Duration `env:"REQUESTS_INTERVAL,default=10s"`
	RequestCooldown     time.Duration `env:"REQUEST_COOLDOWN,default=10s"`
	Env                 string        `env:"ENV,default=dev"`
}

// NewConfig reads config from env and creates config struct
func NewConfig() (*Config, error) {
	ctx := context.Background()
	var c Config
	if err := envconfig.Process(ctx, &c); err != nil {
		return nil, err
	}

	return &c, nil
}
