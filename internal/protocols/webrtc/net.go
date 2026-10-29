package webrtc

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/bluenviron/gortsplib/v5/pkg/readbuffer"
	"github.com/pion/transport/v5"
	"github.com/pion/transport/v5/stdnet"
	"github.com/wlynxg/anet"
)

// interfacesTTL is how long Interfaces() reuses the last list.
// Listing interfaces is expensive on Windows, where the addresses of each interface
// need a separate GetAdaptersAddresses call, and a session lists them several times.
// The TTL is short enough to still follow network changes.
const interfacesTTL = 5 * time.Second

// Net is stdnet.Net with the following changes:
//   - Interfaces() is overridden to query the OS directly, caching the result for a few seconds only.
//   - ListenUDP() is overridden to apply the configured read buffer size to the returned UDPConn.
type Net struct {
	UDPReadBufferSize int

	stdnet.Net

	mutex       sync.Mutex
	ifs         []*transport.Interface
	ifsExpireAt time.Time
}

// Interfaces returns the current list of network interfaces.
// The OS is queried again once the previous result is older than interfacesTTL.
func (n *Net) Interfaces() ([]*transport.Interface, error) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	now := time.Now()
	if n.ifs == nil || !now.Before(n.ifsExpireAt) {
		ifs, err := queryInterfaces()
		if err != nil {
			return nil, err
		}
		n.ifs = ifs
		n.ifsExpireAt = now.Add(interfacesTTL)
	}

	return append([]*transport.Interface(nil), n.ifs...), nil
}

func queryInterfaces() ([]*transport.Interface, error) {
	oifs, err := anet.Interfaces()
	if err != nil {
		return nil, err
	}

	ifs := make([]*transport.Interface, 0, len(oifs))
	for i := range oifs {
		ifc := transport.NewInterface(oifs[i])

		var addrs []net.Addr
		addrs, err = anet.InterfaceAddrsByInterface(&oifs[i])
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			ifc.AddAddress(addr)
		}

		ifs = append(ifs, ifc)
	}

	return ifs, nil
}

// InterfaceByIndex returns the interface specified by index.
func (n *Net) InterfaceByIndex(index int) (*transport.Interface, error) {
	ifaces, err := n.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, ifc := range ifaces {
		if ifc.Index == index {
			return ifc, nil
		}
	}

	return nil, fmt.Errorf("%w: index=%d", transport.ErrInterfaceNotFound, index)
}

// InterfaceByName returns the interface specified by name.
func (n *Net) InterfaceByName(name string) (*transport.Interface, error) {
	ifaces, err := n.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, ifc := range ifaces {
		if ifc.Name == name {
			return ifc, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", transport.ErrInterfaceNotFound, name)
}

// ListenUDP acts like ListenPacket for UDP networks and applies the configured
// read buffer size.
func (n *Net) ListenUDP(network string, laddr *net.UDPAddr) (transport.UDPConn, error) {
	conn, err := net.ListenUDP(network, laddr)
	if err != nil {
		return nil, err
	}

	if n.UDPReadBufferSize != 0 {
		if err = readbuffer.SetReadBuffer(conn, n.UDPReadBufferSize); err != nil {
			return nil, err
		}
	}

	return conn, nil
}
