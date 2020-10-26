import os
import scanner
from bluepy.btle import Peripheral
import time
import sys
import struct
import requests
import datetime
import getmac

MAGIC_ENABLE_SENSORS = b"\xA0\x1F"
BLE_CHAR_CMD = 0x0033
BLE_CHAR_SENSORS = 0x0035
BLE_CHAR_BATTERY = 0x0038
INGESTER_URL = os.getenv("INGESTER_URL", "https://plants.pedanticorderliness.com/metrics")
COLLECTOR_MAC = getmac.get_mac_address()
FIVE_MINUTES = 60 * 5

while True:
    monitors = scanner.discover_devices()

    for monitor in monitors:
        print("Found monitor: %s" % monitor)

        tags = ["plant_mac:%s" % monitor]

        try:
            periph = Peripheral()
            print("connecting...")
            periph.connect(monitor)
            print("connected")
            periph.writeCharacteristic(BLE_CHAR_CMD, MAGIC_ENABLE_SENSORS, withResponse=True)
            print("enabled sensor reading")

            sample = periph.readCharacteristic(BLE_CHAR_SENSORS)
            print("Data: %s" % ":".join("{:02x}".format(c) for c in sample))

            temp = struct.unpack("<H", sample[0:2])[0] / 10
            light = struct.unpack("<L", sample[3:7])[0]
            moist = struct.unpack("<B", sample[7:8])[0]
            cond = struct.unpack("<H", sample[8:10])[0]

            sample = periph.readCharacteristic(BLE_CHAR_BATTERY)
            battery = struct.unpack("<B", sample[0:1])[0]

            # Report metrics to ingester
            metrics = {
                    'time': datetime.datetime.utcnow().isoformat() + "Z",
                    'collector': COLLECTOR_MAC,
                    'plant': monitor,
                    'temp': temp,
                    'light': light,
                    'moist': moist,
                    'cond': cond,
                    'battery': battery
            }
            res = requests.post(INGESTER_URL, json=metrics, timeout=30) 
            if res.status_code != 204:
                    print("Problem sending metrics to ingester", res.status_code)

            print("Temp: %fc, Light: %d lux, Moisture: %d%%, Cond: %d uS/cm, Batt: %d%%" % 
                    (temp, light, moist, cond, battery))

            print("disconnecting...")
            periph.disconnect()
            print("disconnected")
        except:
            print("Problem communicating with %s" % (monitor))
            print(sys.exc_info())

            if periph is not None:
                print("disconnecting...")
                periph.disconnect()
                print("disconnected")

    time.sleep(FIVE_MINUTES)
