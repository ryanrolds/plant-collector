package devices

import (
	"time"

	"github.com/ryanrolds/plant-collector/bridge/internal/ingester"
	"github.com/sirupsen/logrus"
	"tinygo.org/x/bluetooth"
)

var lastSeenFreshness = 30 * time.Minute
var maxLastPollAge = 1 * time.Minute // 30 minutes

type XiaomiDevice struct {
	MacAddress string
	LastPoll   time.Time
	LastSeen   time.Time
	Battery    int
}

type XiaomiBatteryPoller struct {
	devices map[string]XiaomiDevice
}

func NewXiaomiBatteryPoller() *XiaomiBatteryPoller {
	return &XiaomiBatteryPoller{
		devices: make(map[string]XiaomiDevice),
	}
}

func (p *XiaomiBatteryPoller) AddDevice(mac string) {
	// if we already have the device, update the last seen time
	if _, ok := p.devices[mac]; ok {
		p.devices[mac] = XiaomiDevice{
			MacAddress: mac,
			LastPoll:   p.devices[mac].LastPoll,
			LastSeen:   time.Now(),
		}

		logrus.Infof("added device: %s", mac)
		return
	}

	// otherwise, add the device
	p.devices[mac] = XiaomiDevice{
		MacAddress: mac,
		LastPoll:   time.Time{},
		LastSeen:   time.Now(),
	}
}

func (p *XiaomiBatteryPoller) Poll(adapter *bluetooth.Adapter, samples chan<- ingester.Sample) {
	for mac, sensor := range p.devices {
		// if we haven't seen the device in 30 minutes, remove it
		if time.Since(sensor.LastSeen) > lastSeenFreshness {
			delete(p.devices, mac)
			logrus.Infof("removed device: %s", mac)
			continue
		}

		// if we haven't polled the device in an hour, poll it
		if time.Since(sensor.LastPoll) > maxLastPollAge {
			macAddress, err := bluetooth.ParseMAC(mac)
			if err != nil {
				logrus.WithField("mac", mac).WithError(err).Error("failed to parse mac address")
				continue
			}

			device, err := adapter.Connect(bluetooth.MACAddress{
				MAC: macAddress,
			}, bluetooth.ConnectionParams{})
			if err != nil {
				logrus.WithError(err).Error("failed to connect to device")
				continue
			}

			battery, err := p.readDeviceBattery(device)
			if err != nil {
				logrus.WithError(err).Error("failed to read device battery")
			}

			p.devices[mac] = sensor

			samples <- ingester.Sample{
				Time:      time.Now(),
				Collector: "bridge",
				Plant:     mac,
				Battery:   &battery,
			}
		}
	}
}

func (p *XiaomiBatteryPoller) readDeviceBattery(device *bluetooth.Device) (int, error) {
	logrus.WithField("device", device).Info("polling device")

	//readService := bluetooth.New16BitUUID(0x0033)
	services, err := device.DiscoverServices(nil)
	if err != nil {
		logrus.Info("failed to discover services")
		return 0, err
	}

	for _, service := range services {
		logrus.WithField("service", service).Info("found service")
	}

	return 0, nil
}
