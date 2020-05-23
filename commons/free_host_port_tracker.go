package commons

import "github.com/palantir/stacktrace"

type FreeHostPortTracker struct {
	PortRangeStart int
	PortRangeEnd int
	takenPorts map[int]bool
}

func NewFreeHostPortTracker(portRangeStart int, portRangeEnd int) *FreeHostPortTracker {
	portMap := make(map[int]bool)
	return &FreeHostPortTracker{
		PortRangeStart: portRangeStart,
		PortRangeEnd: portRangeEnd,
		takenPorts: portMap,
	}
}

func (hostPortTracker FreeHostPortTracker) GetFreePort() (port int, err error) {
	for port, taken := range hostPortTracker.takenPorts {
		if !taken {
			hostPortTracker.takenPorts[port] = true
			return port, nil
		}
	}
	return -1, stacktrace.NewError("There are no more free ports available given the host port range.")
}

func (hostPortTracker FreeHostPortTracker) ReleasePort(port int) (err error) {
	if _, ok := hostPortTracker.takenPorts[port]; ok {
		delete(hostPortTracker.takenPorts, port)
	}
	return nil
}

