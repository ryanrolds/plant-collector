package scanner

import (
	"context"

	"github.com/ryanrolds/plant-collector/bridge/internal/ingester"
	"github.com/ryanrolds/plant-collector/bridge/internal/scanner/devices"
	"github.com/sirupsen/logrus"
	"tinygo.org/x/bluetooth"
)

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
