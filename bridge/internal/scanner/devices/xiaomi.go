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

		log.WithFields(logrus.Fields{
			//"temp": tempCelcius,
		}).Debug("received xiaomi sample")

		samples <- sample
	}
}

func parseXiaomiSensorData(sensorData []byte) (ingester.Sample, error) {
	logrus.Infof("data %x", sensorData)

	if len(sensorData) < 5 {
		logrus.Error("sensor data is too short")
		return ingester.Sample{}, nil
	}

	// It looks like the Xiaomi sensors BTLE advertisements are very flexible and are used for a variety of different sensors.
	// https://github.com/AlCalzone/ioBroker.ble/blob/master/src/plugins/lib/xiaomi_protocol.ts#LL108C7-L108C18
	// https://github.com/custom-components/ble_monitor/blob/master/custom_components/ble_monitor/ble_parser/xiaomi.py

	// 4 bit version, 12 bit flags
	controlData := binary.BigEndian.Uint16(sensorData[4:6])
	version := controlData >> 12
	flags := controlData & 0xfff
	newFactory := (flags & 1 << 0) != 0
	connected := (flags & 1 << 1) != 0
	central := (flags & 1 << 2) != 0
	encrypted := (flags & 1 << 3) != 0
	macAddress := (flags & 1 << 4) != 0
	capabilities := (flags & 1 << 5) != 0
	event := (flags & 1 << 6) != 0
	customData := (flags & 1 << 7) != 0
	subtitle := (flags & 1 << 8) != 0
	binding := (flags & 1 << 9) != 0

	logrus.WithFields(logrus.Fields{
		"version":      version,
		"newFactory":   newFactory,
		"connected":    connected,
		"central":      central,
		"encrypted":    encrypted,
		"macAddress":   macAddress,
		"capabilities": capabilities,
		"event":        event,
		"customData":   customData,
		"subtitle":     subtitle,
		"binding":      binding,
	}).Debug("sensor flags")

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

	measurementType := binary.LittleEndian.Uint16(sensorData[12:14])
	logrus.Infof("data %x", measurementType)

	switch measurementType {
	case uint16(0x1004):
		// temperature
		tempCelcius := float32(binary.LittleEndian.Uint16(sensorData[13:15])) / 10
		m.Temperature = &tempCelcius
	case uint16(0x1006):
		// humidity
		humidity := float32(binary.LittleEndian.Uint16(sensorData[13:15])) / 10
		m.Humidity = &humidity
	case uint16(0x1007):
		// light
		light := float32(binary.LittleEndian.Uint16(sensorData[13:15]))
		m.Light = &light
	case uint16(0x1008):
		// moisture
		moisture := float32(sensorData[15:16][0])
		m.Moisture = &moisture
	case uint16(0x1009):
		// conductivity
		conductivity := float32(binary.LittleEndian.Uint16(sensorData[15:17]))
		m.Conductivity = &conductivity
	case uint16(0x100a):
		// battery
		battery := int(binary.LittleEndian.Uint16(sensorData[13:15]))
		m.Battery = &battery
	default:
		logrus.WithFields(logrus.Fields{
			"type": measurementType,
		}).Warn("unknown measurement type")
		return ingester.Sample{}, fmt.Errorf("unknown measurement type: %d", measurementType)
	}

	return m, nil
}

/*
                                                   xxxx xxxx xx xxxxxxxxxxxx xx xxxx
                                                   0 1  2 3  4  5 6 7 8 9 10 11 1213 14 151617
23.01.23 00:29:49 (-0800)  bridge  INFO[0014] data 7120 9800 d9 44e988caea80 0d 0910 02 3100       mac="80:EA:CA:88:E9:44"
23.01.23 00:29:51 (-0800)  bridge  INFO[0017] data 7120 9800 d4 ea47678d7cc4 0d 0810 01 15         mac="C4:7C:8D:67:47:EA"
23.01.23 00:29:54 (-0800)  bridge  INFO[0020] data 7120 9800 46 104a678d7cc4 0d 0810 01 0f         mac="C4:7C:8D:67:4A:10"
23.01.23 00:29:55 (-0800)  bridge  INFO[0021] data 7120 9800 d9 44e988caea80 0d 0910 02 3100       mac="80:EA:CA:88:E9:44"
23.01.23 00:29:58 (-0800)  bridge  INFO[0024] data 7120 9800 c7 3d726b8d7cc4 0d 0910 02 3001       mac="C4:7C:8D:6B:72:3D"
23.01.23 00:29:58 (-0800)  bridge  INFO[0024] data 7120 9800 d5 ea47678d7cc4 0d 0910 02 2400       mac="C4:7C:8D:67:47:EA"
23.01.23 00:30:01 (-0800)  bridge  INFO[0027] data 7120 9800 47 104a678d7cc4 0d 0910 02 2300       mac="C4:7C:8D:67:4A:10"
23.01.23 00:30:06 (-0800)  bridge  INFO[0032] data 7120 9800 d6 ea47678d7cc4 0d 0410 02 a400       mac="C4:7C:8D:67:47:EA"
23.01.23 00:30:07 (-0800)  bridge  INFO[0033] data 7120 9800 db 44e988caea80 0d 0710 03 1a0000     mac="80:EA:CA:88:E9:44"
23.01.23 00:30:11 (-0800)  bridge  INFO[0037] data 7120 9800 48 104a678d7cc4 0d 0410 02 a400       mac="C4:7C:8D:67:4A:10"
23.01.23 00:30:14 (-0800)  bridge  INFO[0040] data 7120 9800 48 104a678d7cc4 0d 0410 02 a400       mac="C4:7C:8D:67:4A:10"
23.01.23 00:30:17 (-0800)  bridge  INFO[0043] data 7120 9800 d7 ea47678d7cc4 0d 0710 03 2a0000     mac="C4:7C:8D:67:47:EA"
23.01.23 00:30:18 (-0800)  bridge  INFO[0044] data 7120 9800 d7 ea47678d7cc4 0d 0710 03 2a0000     mac="C4:7C:8D:67:47:EA"
23.01.23 00:30:20 (-0800)  bridge  INFO[0046] data 7120 9800 c9 3d726b8d7cc4 0d 0710 03 160000     mac="C4:7C:8D:6B:72:3D"
23.01.23 00:30:21 (-0800)  bridge  INFO[0046] data 7120 9800 dd 44e988caea80 0d 0910 02 3400       mac="80:EA:CA:88:E9:44"
23.01.23 00:30:21 (-0800)  bridge  INFO[0047] data 7120 9800 49 104a678d7cc4 0d 0710 03 1e0000     mac="C4:7C:8D:67:4A:10"
23.01.23 00:30:29 (-0800)  bridge  INFO[0055] data 7120 9800 de 44e988caea80 0d 0410 02 a800       mac="80:EA:CA:88:E9:44"
23.01.23 00:30:31 (-0800)  bridge  INFO[0057] data 7120 9800 d8 ea47678d7cc4 0d 0810 01 15         mac="C4:7C:8D:67:47:EA"
23.01.23 00:30:32 (-0800)  bridge  INFO[0058] data 7120 9800 d8 ea47678d7cc4 0d 0810 01 15         mac="C4:7C:8D:67:47:EA"
23.01.23 00:30:40 (-0800)  bridge  INFO[0066] data 7120 9800 d9 ea47678d7cc4 0d 0910 02 2400       mac="C4:7C:8D:67:47:EA"
23.01.23 00:30:40 (-0800)  bridge  INFO[0066] data 7120 9800 cb 3d726b8d7cc4 0d 0910 02 2d01       mac="C4:7C:8D:6B:72:3D"
23.01.23 00:30:45 (-0800)  bridge  INFO[0071] data 7120 9800 4b 104a678d7cc4 0d 0910 02 2300       mac="C4:7C:8D:67:4A:10"
23.01.23 00:30:47 (-0800)  bridge  INFO[0073] data 7120 9800 d9 ea47678d7cc4 0d 0910 02 2400       mac="C4:7C:8D:67:47:EA"
23.01.23 00:30:50 (-0800)  bridge  INFO[0076] data 7120 9800 da ea47678d7cc4 0d 0410 02 a400       mac="C4:7C:8D:67:47:EA"
23.01.23 00:30:51 (-0800)  bridge  INFO[0077] data 7120 9800 4c 104a678d7cc4 0d 0410 02 a300       mac="C4:7C:8D:67:4A:10"
23.01.23 00:30:52 (-0800)  bridge  INFO[0078] data 7120 9800 cc 3d726b8d7cc4 0d 0410 02 aa00       mac="C4:7C:8D:6B:72:3D"
23.01.23 00:30:57 (-0800)  bridge  INFO[0083] data 7120 9800 da ea47678d7cc4 0d 0410 02 a400       mac="C4:7C:8D:67:47:EA"
23.01.23 00:31:01 (-0800)  bridge  INFO[0087] data 7120 9800 db ea47678d7cc4 0d 0710 03 2a0000     mac="C4:7C:8D:67:47:EA"
23.01.23 00:31:07 (-0800)  bridge  INFO[0093] data 7120 9800 4d 104a678d7cc4 0d 0710 03 1e0000     mac="C4:7C:8D:67:4A:10"
23.01.23 00:31:07 (-0800)  bridge  INFO[0093] data 7120 9800 e1 44e988caea80 0d 0910 02 3300       mac="80:EA:CA:88:E9:44"
23.01.23 00:31:10 (-0800)  bridge  INFO[0096] data 7120 9800 4d 104a678d7cc4 0d 0710 03 1e0000     mac="C4:7C:8D:67:4A:10"
23.01.23 00:31:12 (-0800)  bridge  INFO[0098] data 7120 9800 e2 44e988caea80 0d 0410 02 a800       mac="80:EA:CA:88:E9:44"
23.01.23 00:31:12 (-0800)  bridge  INFO[0098] data 7120 9800 4e 104a678d7cc4 0d 0810 01 0f         mac="C4:7C:8D:67:4A:10"
23.01.23 00:31:14 (-0800)  bridge  INFO[0100] data 7120 9800 dd ea47678d7cc4 0d 0910 02 2300       mac="C4:7C:8D:67:47:EA"
23.01.23 00:31:15 (-0800)  bridge  INFO[0101] data 7120 9800 dd ea47678d7cc4 0d 0910 02 2300       mac="C4:7C:8D:67:47:EA"
23.01.23 00:31:24 (-0800)  bridge  INFO[0110] data 7120 9800 de ea47678d7cc4 0d 0410 02 a400       mac="C4:7C:8D:67:47:EA"
23.01.23 00:31:30 (-0800)  bridge  INFO[0116] data 7120 9800 4f 104a678d7cc4 0d 0910 02 2400       mac="C4:7C:8D:67:4A:10"
23.01.23 00:31:35 (-0800)  bridge  INFO[0121] data 7120 9800 df ea47678d7cc4 0d 0710 03 160000     mac="C4:7C:8D:67:47:EA"
23.01.23 00:31:36 (-0800)  bridge  INFO[0122] data 7120 9800 50 104a678d7cc4 0d 0410 02 a400       mac="C4:7C:8D:67:4A:10"
23.01.23 00:31:44 (-0800)  bridge  INFO[0130] data 7120 9800 e0 ea47678d7cc4 0d 0810 01 15         mac="C4:7C:8D:67:47:EA"
23.01.23 00:31:44 (-0800)  bridge  INFO[0130] data 7120 9800 51 104a678d7cc4 0d 0710 03 2f0000     mac="C4:7C:8D:67:4A:10"
23.01.23 00:31:45 (-0800)  bridge  INFO[0130] data 7120 9800 e5 44e988caea80 0d 0910 02 3200       mac="80:EA:CA:88:E9:44"
23.01.23 00:31:46 (-0800)  bridge  INFO[0132] data 7120 9800 51 104a678d7cc4 0d 0710 03 2f0000     mac="C4:7C:8D:67:4A:10"
23.01.23 00:31:59 (-0800)  bridge  INFO[0145] data 7120 9800 e1 ea47678d7cc4 0d 0910 02 2600       mac="C4:7C:8D:67:47:EA"

7120 9800 d9 44e988caea80 0d 0910 02 3100

-----------------------------------------------------

little endian

0:1 = 7120 - frame control?

7120 = 01110001 00100000
LE->BE = 00100000 01110001

00000000 00000010 version
00000000 01110001 flags

NewFactory(0)=1
Connected(1)=0
Central(2)=0
Encrypted(3)=0,

MacAddress(4)=1
Capabilities(5)=1,
Event(6)=1,
CustomData(7)=0,

Subtitle(8)=0,
Binding(9)=0,

1:2 = version (2)
2:4 = 9800 - device id 0x0098 - HHCCJCY01
4:5 = frame counter - plain counter (probably useful in determining if data missing)
5:11 = mac address (if 0:1[0x16] == 1)
11:12 = capabilities (if 0:1[0x32] == 1)

0d = 00001101

Connectable,Encrypt,IO

12:14 - event data type (sensor type)



14:15 - event data length
15:16+ - event data

https://github.com/AlCalzone/ioBroker.ble/blob/master/src/plugins/lib/xiaomi_protocol.ts#L76

// 4 bit version, 12 bit flags
const frameControl = data.readUInt16LE(0);
this._version = frameControl >>> 12;
const flags = frameControl & 0xfff;
this._isNewFactory = (flags & FrameControlFlags.NewFactory) !== 0;
this._isConnected = (flags & FrameControlFlags.Connected) !== 0;
this._isCentral = (flags & FrameControlFlags.Central) !== 0;
this._isEncrypted = (flags & FrameControlFlags.Encrypted) !== 0;
this._hasMacAddress = (flags & FrameControlFlags.MacAddress) !== 0;
this._hasCapabilities = (flags & FrameControlFlags.Capabilities) !== 0;
this._hasEvent = (flags & FrameControlFlags.Event) !== 0;
this._hasCustomData = (flags & FrameControlFlags.CustomData) !== 0;
this._hasSubtitle = (flags & FrameControlFlags.Subtitle) !== 0;
this._isBindingFrame = (flags & FrameControlFlags.Binding) !== 0;

https://github.com/AlCalzone/ioBroker.ble/blob/master/src/plugins/lib/xiaomi_protocol.ts

const enum FrameControlFlags {
	NewFactory = 1 << 0,
	Connected = 1 << 1,
	Central = 1 << 2,
	Encrypted = 1 << 3,
	MacAddress = 1 << 4,
	Capabilities = 1 << 5,
	Event = 1 << 6,
	CustomData = 1 << 7,
	Subtitle = 1 << 8,
	Binding = 1 << 9,
}

export enum CapabilityFlags {
	Connectable = 1 << 0,
	Central = 1 << 1,
	Encrypt = 1 << 2,
	IO = (1 << 3) | (1 << 4),
}

*/
