package main

import (
	"log"
	"rate-limiter/cmd"

	"go.uber.org/zap"
)

func main() {
	config, err := cmd.NewConfig()
	if err != nil {
		log.Fatalf("can't decode config: %s \n", err.Error())
		return
	}

	zapConfig := zap.NewDevelopmentConfig()
	if config.Env == "prod" {
		zapConfig = zap.NewProductionConfig()
	}
	logger := zap.Must(zapConfig.Build())
	defer logger.Sync()
	zap.ReplaceGlobals(logger)

	srv := cmd.NewServer(logger, config)
	srv.RunServer()
}
