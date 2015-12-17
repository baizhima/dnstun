package tun

import (
	"../songgao/water"
	"fmt"
)

type Tunnel struct {
	name string
	conn *water.Interface
}

func NewTunnel(name string) (*Tunnel, error) {

	t := new(Tunnel)
	t.name = name

	var err error
	t.conn, err = water.NewTUN(name)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (t *Tunnel) Write(p []byte) error {
	n, err := t.conn.Write(p)
	if err != nil {
		return err
	}
	if n != len(p) {
		return fmt.Errorf("Short write %d, should be %d", n, len(p))
	}
	return nil
}

func (t *Tunnel) Save(buffer map[int][]byte, tun *TUNIpPacket) error {

	if tun.Offset == 0 && tun.More == false {
		ippkt := tun.Payload
		t.conn.Write(ippkt) // send to upper layer
		return nil
	}
	pkt, ok := buffer[tun.Id]
	if ok {
		if tun.Offset == len(pkt) {
			pkt := append(pkt, tun.Payload...)
			if tun.More == false {
				t.Write(pkt)
				delete(buffer, tun.Id)
			} else {
				buffer[tun.Id] = pkt
			}
		}
	} else {
		buffer[tun.Id] = tun.Payload
	}
	return nil
}

func (t *Tunnel) Read(p []byte) (int, error) {
	n, err := t.conn.Read(p)
	return n, err
}
