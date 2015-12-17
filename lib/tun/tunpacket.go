package tun

import (
	"../tonnerre/golang-dns"
	"net"
)

const (
	TUN_CMD_CONNECT  byte = 'c'
	TUN_CMD_RESPONSE byte = 'r'
	TUN_CMD_DATA     byte = 'd'
	TUN_CMD_KILL     byte = 'k'
	TUN_CMD_EMPTY    byte = 'e' // empty packet, with user id,
	// just for server to have more dns id
	TUN_CMD_ACK byte = 'a' // no user id, a normal dns request
)

type TUNPacket interface {
	GetCmd() byte

	/* The Physical UDP Address for an incoming packet
	   may change over time, e.g. using different middle
	   DNS Server. By using UserId field to identify the source
	   Of a TUN Packet */
	GetUserId() int
}

type TUNCmdPacket struct {
	Cmd    byte
	UserId int
}

type TUNResponsePacket struct {
	Cmd     byte
	UserId  int
	Server  *net.IPAddr
	Client  *net.IPAddr
	Request *dns.Msg
}

type TUNEmptyPacket struct {
    Cmd     byte
    UserId  int
    Request *dns.Msg
    Payload []byte
}


type TUNAckPacket struct {
	Cmd     byte
	UserId  int
	Request *dns.Msg
}

type TUNIpPacket struct {
	Cmd     byte
	UserId  int
	Id      int
	Offset  int
	More    bool
	Payload []byte
}

func (t *TUNCmdPacket) GetCmd() byte {
	return t.Cmd
}
func (t *TUNResponsePacket) GetCmd() byte {
	return TUN_CMD_RESPONSE
}

func (t *TUNAckPacket) GetCmd() byte {
	return t.Cmd
}

func (t *TUNEmptyPacket) GetCmd() byte {
    return t.Cmd
}

func (t *TUNIpPacket) GetCmd() byte {
    return t.Cmd
}



func (t *TUNCmdPacket) GetUserId() int {
	return t.UserId
}

func (t *TUNResponsePacket) GetUserId() int {
	return t.UserId
}

func (t *TUNAckPacket) GetUserId() int {
	return t.UserId
}

func (t *TUNEmptyPacket) GetUserId() int {
    return t.UserId
}

func (t *TUNIpPacket) GetUserId() int {
	return t.UserId
}

/*
func (t *TUNPacket) Unpack(domain string) (*TUNPacket, error){

    // TODO: TUNCmdPacket, TUNResponsePacket

    labels := strings.Split(name, ".")

	// labels[0]...labels[3]: dlabel(54)
	// labels[4]: identification(5)
	// labels[5]: MF(1)
	// labels[6]: idx(4)
	// labels[7]: cmd(2)
	// labels[8:]: b.jannotti.com(14)
	// total: 54*4+3 + 5+1+1+1+4+1+2+1+14 = 249
	_id, _ := strconv.Atoi(labels[4])
	_mf, _ := strconv.Atoi(labels[5])
	_idx, _ := strconv.Atoi(labels[6])
	_cmd, _ := strconv.Atoi(labels[7])
	outPacket := &TUNIpPacket{
		Cmd: _cmd,
		Id:  _id,
		Idx: _idx,
	}
	if _mf == 1 {
		outPacket.More = true
	}
	raw := labels[0] + labels[1] + labels[2] + labels[3]

	outPacket.EncodedStr = raw
	return outPacket, nil
}
*/
