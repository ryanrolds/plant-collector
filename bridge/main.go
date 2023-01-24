package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ryanrolds/plant-collector/bridge/internal/ingester"
	"github.com/ryanrolds/plant-collector/bridge/internal/scanner"
	"github.com/sirupsen/logrus"
)

func init() {
	// Log as JSON instead of the default ASCII formatter.
	//logrus.SetFormatter(&logrus.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	logrus.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	logrus.SetLevel(logrus.InfoLevel)
}

func main() {
	logrus.Info("starting bridge")

	ingesterURL := os.Getenv("INGESTER_URL")
	if ingesterURL == "" {
		logrus.Fatal("INGESTER_URL must be set")
	}

	logrus.WithField("url", ingesterURL).Info("ingester selected")

	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		logrus.Info("SIGINT/SIGTERM received, shutting down")
		cancel()
	}()

	// channel for buffering samples
	samples := make(chan ingester.Sample, 100)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		scanner := scanner.NewBTLEScanner()
		err := scanner.Scan(ctx, samples)
		if err != nil {
			logrus.Error(err)
		}

		logrus.Info("scanner finished")
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		i := ingester.NewIngester(ingesterURL)
		err := i.SendAll(ctx, samples)
		if err != nil {
			logrus.Error(err)
		}

		logrus.Info("ingester finished")
		wg.Done()
	}()

	logrus.Info("waiting for goroutines to finish")
	wg.Wait()

	logrus.Info("shutting down")
}
