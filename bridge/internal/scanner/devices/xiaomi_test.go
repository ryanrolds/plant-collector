package devices

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseXiaomiData(t *testing.T) {
	data := []byte{
		0x71, 0x20, // frame control
		0x98, 0x00, // device type
		0xd9,                               // counter
		0x06, 0x05, 0x04, 0x03, 0x02, 0x01, // reversed mac address
		0x0d, // capabilities
	}

	tests := []struct {
		name                     string
		measurementBytes         []byte
		expectedMeasurementType  string
		expectedMeasurementValue interface{}
	}{
		{
			name: "conductivity",
			measurementBytes: []byte{
				0x09, 0x10, // measurement type
				0x02,       // measurement length
				0x31, 0x00, // measurement value
			},
			expectedMeasurementType:  "conductivity",
			expectedMeasurementValue: float32(49),
		},
		{
			name: "moisture",
			measurementBytes: []byte{
				0x08, 0x10, // measurement type
				0x01, // measurement length
				0x32, // measurement value
			},
			expectedMeasurementType:  "moisture",
			expectedMeasurementValue: float32(50),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testData := append(data, tt.measurementBytes...)
			sample, err := parseXiaomiSensorData(testData)
			assert.Nil(t, err)

			assert.Equal(t, "01:02:03:04:05:06", sample.Plant)
			assert.Equal(t, "bridge", sample.Collector)

			switch tt.expectedMeasurementType {
			case "conductivity":
				assert.Equal(t, tt.expectedMeasurementValue, *sample.Conductivity)
			case "moisture":
				assert.Equal(t, tt.expectedMeasurementValue, *sample.Moisture)
			}
		})
	}
}
