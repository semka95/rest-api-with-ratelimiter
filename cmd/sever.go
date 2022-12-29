package cmd

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rate-limiter/api"
	"rate-limiter/limiter"

	"go.uber.org/zap"
)

// RestServer represents rest server
type RestServer struct {
	logger *zap.Logger
	config *Config
}

// NewServer creates rest server
func NewServer(logger *zap.Logger, config *Config) RestServer {
	return RestServer{
		logger: logger,
		config: config,
	}
}

// RunServer runs rest server
func (s *RestServer) RunServer() {
	// init limiter store
	rateLimiter, err := limiter.NewRequestLimiter(s.config.RequestsPerInterval, s.config.RequestsInterval, s.config.RequestCooldown)
	if err != nil {
		s.logger.Error("can't create limiter store", zap.Error(err))
		return
	}

	// init router
	a := api.API{}
	router := a.NewRouter(net.CIDRMask(s.config.Mask, 32), rateLimiter, s.logger)

	// init server
	srv := &http.Server{
		Addr:           s.config.HTTPServerAddress,
		Handler:        router,
		ReadTimeout:    time.Duration(s.config.ReadTimeout) * time.Second,
		IdleTimeout:    time.Duration(s.config.IdleTimeout) * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	// run server
	s.logger.Info("starting http server", zap.String("address", s.config.HTTPServerAddress))
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("can't start server", zap.Error(err), zap.String("server address", s.config.HTTPServerAddress))
		}
	}()

	// graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	s.logger.Info("received interrupt signal, stopping server")
	timeout, cancel := context.WithTimeout(context.Background(), time.Duration(s.config.ShutdownTimeout)*time.Second)
	defer cancel()
	rateLimiter.Close(timeout)
	if err := srv.Shutdown(timeout); err != nil {
		s.logger.Error("can't shutdown http server", zap.Error(err))
	}
}
