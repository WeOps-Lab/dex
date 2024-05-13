package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

func main() {
	reg := prometheus.NewRegistry()
	reg.MustRegister(newDockerCollector())

	router := http.NewServeMux()
	router.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		Registry: reg,
	}))

	listenAddress := flag.String("web.listen-address", getEnv("EXPORTER_WEB_LISTEN_ADDRESS", ":8089"), "Address to listen on for web interface and telemetry.")
	isDebug := flag.Bool("debug", false, "Output verbose debug information")

	flag.Parse()

	log.SetFormatter(&log.JSONFormatter{})
	if *isDebug {
		log.SetLevel(log.DebugLevel)
		log.Debugln("Enabling debug output")
	} else {
		log.SetLevel(log.InfoLevel)
	}

	server := &http.Server{
		Addr:         *listenAddress,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	done := make(chan bool)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		<-quit
		log.Info("Server is shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		close(done)
	}()

	log.Infof("Server is ready to handle requests at %v", *listenAddress)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Could not listen on %d: %v\n", listenAddress, err)
	}

	<-done
	log.Info("Server stopped")
}

func getEnv(key string, defaultVal string) string {
	if envVal, ok := os.LookupEnv(key); ok {
		return envVal
	}
	return defaultVal
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
