package scanner

import (
	"context"
	"time"

	"github.com/ryanrolds/plant-collector/bridge/internal/ingester"
	"github.com/ryanrolds/plant-collector/bridge/internal/scanner/devices"
	"github.com/sirupsen/logrus"
	"tinygo.org/x/bluetooth"
)

var batteryPollTickerInterval = 5 * time.Minute

type BTLEScanner struct{}

func NewBTLEScanner() *BTLEScanner {
	return &BTLEScanner{}
}

func (s *BTLEScanner) Scan(ctx context.Context, samples chan<- ingester.Sample) error {
	adapter := bluetooth.DefaultAdapter
	err := adapter.Enable()
	if err != nil {
		return err
	}

	xiaomiBatteryPoller := devices.NewXiaomiBatteryPoller()
	// Poll battery levels on a ticker
	go func() {
		logrus.Info("starting battery polling")

		ticker := time.NewTicker(batteryPollTickerInterval)

		for {
			select {
			case <-ctx.Done():
				logrus.Info("stopping battery polling")
				ticker.Stop()
				return
			case <-ticker.C:
				logrus.Debug("polling battery levels")
				xiaomiBatteryPoller.Poll(adapter, samples)
			}
		}
	}()

	// Stop scanning on context cancel
	go func() {
		<-ctx.Done()
		logrus.Info("stopping scan")
		err := adapter.StopScan()
		if err != nil {
			logrus.Error(err)
		}
	}()

	logrus.Info("starting scan")
	adapter.Scan(func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
		log := logrus.WithFields(logrus.Fields{
			"mac":  device.Address.String(),
			"name": device.LocalName(),
		})

		// Flower Care
		if device.LocalName() == "Flower care" {
			devices.HandleXiaomiResult(device, samples)
			xiaomiBatteryPoller.AddDevice(device.Address.String())
			return
		}

		// b-parasite
		if device.LocalName() == "prst" {
			m, ok := devices.ParseBparasiteData(device)
			if ok {
				samples <- m
			}

			return
		}

		log.Debug("not a plant sensor")
	})

	return nil
}
