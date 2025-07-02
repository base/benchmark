package portmanager

import (
	"fmt"
	"net"
	"time"
)

type PortPurpose uint64

const (
	ELPortPurpose PortPurpose = iota
	AuthELPortPurpose
	ELMetricsPortPurpose
	BuilderMetricsPortPurpose
)

type PortManager interface {
	AcquirePort(nodeType string, purpose PortPurpose) uint64
	ReleasePort(portNumber uint64)
}

type portManager struct {
	// ports is a map of node type to a map of port purpose to port number.
	ports map[uint64]struct{}
}

func NewPortManager() PortManager {
	return &portManager{
		ports: make(map[uint64]struct{}),
	}
}

func (p *portManager) AcquirePort(nodeType string, purpose PortPurpose) uint64 {
	// find the next available port number
	for port := uint64(10000); port < 65535; port++ {
		if _, exists := p.ports[port]; !exists {
			// check if port is already in use
			if conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 100*time.Millisecond); err == nil {
				_ = conn.Close()
				continue // port is in use, try the next one
			}

			p.ports[port] = struct{}{}
			return port
		}
	}

	panic(fmt.Sprintf("no available ports for node type %s and purpose %d", nodeType, purpose))
}

func (p *portManager) ReleasePort(portNumber uint64) {
	if _, exists := p.ports[portNumber]; !exists {
		return
	}

	delete(p.ports, portNumber)
}
