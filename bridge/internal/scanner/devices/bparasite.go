package devices

import (
	"encoding/binary"
	"time"

	"github.com/ryanrolds/plant-collector/bridge/internal/ingester"
	"github.com/sirupsen/logrus"
	"tinygo.org/x/bluetooth"
)

func ParseBparasiteData(device bluetooth.ScanResult) (ingester.Sample, bool) {
	macAddr := device.Address.String()
	log := logrus.WithField("mac", macAddr)

	serviceData := device.AdvertisementPayload.GetServiceDatas()
	for _, data := range serviceData {
		// we are only interested in the sensor service data
		if data.UUID.String() != "0000181a-0000-1000-8000-00805f9b34fb" {
			continue
		}

		sensorData := data.Data
		log.Debugf("data %x", sensorData)

		//version := sensorData[0] >> 4
		//counter := sensorData[1] & 0x0f
		batteryVoltage := float32(binary.BigEndian.Uint16(sensorData[2:4])) / 1000             // millivolts
		tempCelcius := float32(binary.BigEndian.Uint16(sensorData[4:6])) / 100                 // centicelcius
		humidity := 100 * (float32(binary.BigEndian.Uint16(sensorData[6:8])) / (1 << 16))      // percent
		soilMoisture := 100 * (float32(binary.BigEndian.Uint16(sensorData[8:10])) / (1 << 16)) // percent
		lux := float32(binary.BigEndian.Uint16(sensorData[16:18]))

		// very basic % calculation
		// TODO figure out the actual battery voltage range and curve
		batteryPercentage := int(batteryVoltage / 3.3 * 100)
		rssi := int(device.RSSI)

		s := ingester.Sample{
			Time:        time.Now(),
			Collector:   "bridge",
			Plant:       device.Address.String(),
			Temperature: &tempCelcius,
			Humidity:    &humidity,
			Moisture:    &soilMoisture,
			Light:       &lux,
			Battery:     &batteryPercentage,
			Rssi:        &rssi,
		}

		log.WithField("sample", s).Debug("received bparasite samples")

		return s, true
	}

	return ingester.Sample{}, false
}
