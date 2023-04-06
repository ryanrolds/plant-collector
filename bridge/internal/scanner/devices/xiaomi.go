package devices

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/ryanrolds/plant-collector/bridge/internal/ingester"
	"github.com/sirupsen/logrus"
	"tinygo.org/x/bluetooth"
)

func HandleXiaomiResult(device bluetooth.ScanResult, samples chan<- ingester.Sample) {
	macAddr := device.Address.String()
	log := logrus.WithField("mac", macAddr)

	serviceData := device.AdvertisementPayload.GetServiceDatas()
	for _, data := range serviceData {
		// we are only interested in the sensor service data
		if data.UUID.String() != "0000fe95-0000-1000-8000-00805f9b34fb" {
			log.WithFields(logrus.Fields{
				"uuid": data.UUID.String(),
				"data": data.Data,
			}).Warn("skipping unknown service data")
			continue
		}

		sample, err := parseXiaomiSensorData(data.Data)
		if err != nil {
			log.Warn("failed to parse sensor data")
			continue
		}

		rssi := int(device.RSSI)
		sample.Rssi = &rssi

		log.WithField("sample", sample).Debug("received xiaomi sample")

		samples <- sample
	}
}

/*
xxxx xxxx xx xxxxxxxxxxxx xx xxxx
0 1  2 3  4  5 6 7 8 9 10 11 1213 14 151617
7120 9800 d9 44e988caea80 0d 0910 02 3100
7120 9800 d4 ea47678d7cc4 0d 0810 01 15
7120 9800 46 104a678d7cc4 0d 0810 01 0f
7120 9800 d9 44e988caea80 0d 0910 02 3100
7120 9800 c7 3d726b8d7cc4 0d 0910 02 3001
7120 9800 d5 ea47678d7cc4 0d 0910 02 2400
7120 9800 47 104a678d7cc4 0d 0910 02 2300
7120 9800 d6 ea47678d7cc4 0d 0410 02 a400
7120 9800 db 44e988caea80 0d 0710 03 1a0000
7120 9800 48 104a678d7cc4 0d 0410 02 a400
7120 9800 48 104a678d7cc4 0d 0410 02 a400
7120 9800 00 104a678d7cc4 0d

*/

func parseXiaomiSensorData(sensorData []byte) (ingester.Sample, error) {
	logrus.Infof("sensor data %x", sensorData)

	if len(sensorData) < 5 {
		logrus.Error("sensor data is too short")
		return ingester.Sample{}, nil
	}

	var addrBytes [6]byte
	copy(addrBytes[:], sensorData[5:11])
	macAddr := bluetooth.MACAddress{
		MAC: bluetooth.MAC(addrBytes),
	}

	m := ingester.Sample{
		Time:      time.Now(),
		Collector: "bridge",
		Plant:     macAddr.String(),
	}

	if len(sensorData) < 15 {
		//logrus.Errorf("sensor data is too short %x", sensorData)
		return ingester.Sample{}, nil
	}

	measurementType := binary.LittleEndian.Uint16(sensorData[12:14])
	logrus.Debugf("data %x", measurementType)

	dataSize := int(sensorData[14:15][0])

	switch measurementType {
	case uint16(0x1004):
		// temperature
		tempCelcius := float32(binary.LittleEndian.Uint16(sensorData[15:15+dataSize])) / 10
		m.Temperature = &tempCelcius
	case uint16(0x1007):
		// light
		light := float32(binary.LittleEndian.Uint16(sensorData[15 : 15+dataSize]))
		m.Light = &light
	case uint16(0x1008):
		// moisture
		moistureRaw := sensorData[15:16][0]
		moisture := float32(int(moistureRaw))
		m.Moisture = &moisture
	case uint16(0x1009):
		// conductivity
		conductivity := float32(binary.LittleEndian.Uint16(sensorData[15 : 15+dataSize]))
		m.Conductivity = &conductivity
	case uint16(0x100a):
		// battery
		battery := int(sensorData[15:16][0])
		m.Battery = &battery
	default:
		logrus.WithFields(logrus.Fields{
			"type": measurementType,
		}).Warn("unknown measurement type")
		return ingester.Sample{}, fmt.Errorf("unknown measurement type: %d", measurementType)
	}

	return m, nil
}
