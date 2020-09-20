from bluepy.btle import Scanner, DefaultDelegate

class ScanDelegate(DefaultDelegate):
    def __init__(self):
        DefaultDelegate.__init__(self)

    def handleDiscovery(self, dev, isNewDev, isNewData):
        if isNewDev:
            print("Discovered device", dev.addr)
        elif isNewData:
            print("Received new data from", dev.addr)

def discover_devices():
    # scanner = Scanner().withDelegate(ScanDelegate())
    # devices = scanner.scan(10.0)
    
    devices = Scanner().scan(10.0)
    
    plant_monitors = []
    
    for dev in devices:
        # print "Device %s (%s), RSSI=%d dB" % (dev.addr, dev.addrType, dev.rssi)
        for (adtype, desc, value) in dev.getScanData():
            if adtype == 2 and value == "0000fe95-0000-1000-8000-00805f9b34fb":
                plant_monitors.append(dev.addr)

    return plant_monitors
