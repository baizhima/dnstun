package tun

import (
	"../ip"
	"../tonnerre/golang-dns"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Conn struct {
	VAddr  *net.IPAddr
	PAddr  *net.UDPAddr //
	UserId int

	InChan chan TUNPacket

	Buffer map[int][]byte
	TUN    *Tunnel
	DNS    *DNSUtils
}

type Server struct {

	/* Physical Address DNS Server Listening on */
	//LAddr   *UDPAddr

	/* Virtual Address in TUN Virtual Interface */
	VAddr         *net.IPAddr
	nextUserVAddr *net.IPAddr // .3 - .255
	nextUserId    int

	Routes_By_VAddr  map[string]*Conn
	Routes_By_UserId map[int]*Conn

	DNS *DNSUtils
	TUN *Tunnel
}

func NewServer(topDomain, laddr, vaddr, tunName string) (*Server, error) {

	s := new(Server)

	var err error
	s.VAddr, err = net.ResolveIPAddr("ip", vaddr)
	if err != nil {
		return nil, err
	}

	s.nextUserId = 0
	err = s.nextUserInfo()
	if err != nil {
		return nil, err
	}

	s.Routes_By_VAddr = make(map[string]*Conn)
	s.Routes_By_UserId = make(map[int]*Conn)

	s.DNS, err = NewDNSServer(laddr, topDomain)
	if err != nil {
		return nil, err
	}

	s.TUN, err = NewTunnel(tunName)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Server) nextUserInfo() error {
	networkPrefix := strings.Join(strings.Split(s.VAddr.String(), ".")[:3], ".")
	suffixVal, err := strconv.Atoi(strings.Split(s.VAddr.String(), ".")[3])
	if err != nil {
		return err
	}
	if s.nextUserId != 0 {
		suffixVal, err = strconv.Atoi(strings.Split(s.nextUserVAddr.String(), ".")[3])
		if err != nil {
			return err
		}
	}
	s.nextUserVAddr, err = net.ResolveIPAddr("ip", networkPrefix+"."+strconv.Itoa(suffixVal+1))
	if err != nil {
		return err
	}
	s.nextUserId += 1
	return nil
}

func (s *Server) NewConn(vaddr *net.IPAddr, user int) *Conn {
	c := new(Conn)
	c.VAddr = vaddr
	c.UserId = user

	c.InChan = make(chan TUNPacket, 200)

	c.TUN = s.TUN
	c.DNS = s.DNS
	c.Buffer = make(map[int][]byte)
	return c
}

func (c *Conn) Recv(tunPacket TUNPacket) error {

	// cast packet to TUNIpPacket:  test if it works
    //Debug.Printf("Conn.Recv gets tunPacket\n")
	t, ok := tunPacket.(*TUNIpPacket)
	if !ok {
		return fmt.Errorf("Unexpected cast fail from TUNPacket to TUNIpPacket\n")
	}
	if t.Id < 0 {
		// special IpPacket: SendString
		return nil
	}
	err := c.TUN.Save(c.Buffer, t)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) NormalReply(msg *dns.Msg, paddr *net.UDPAddr) error {
	return nil
}

func (c *Conn) Reply(msg *dns.Msg, paddr *net.UDPAddr) error {


	select {
	case tunPacket := <-c.InChan:
		// There're pending TUN Packets, Inject it into DNS Reply Packet
		// And Send Back
	    err := c.DNS.Reply(msg, tunPacket, paddr)
        return err

	default:
		// TODO
		// there's no pending TUN Packet to be sent,
		// just reply the request

		// normal reply
        t := &TUNAckPacket{
                Cmd:     TUN_CMD_ACK,
                UserId:  c.UserId,
                Request: msg,
        }
		return c.DNS.Reply(msg, t, paddr)

	}
	return nil
}

func (s *Server) FindConnByVAddr(addr string) (*Conn, error) {

	conn, ok := s.Routes_By_VAddr[addr]
	if !ok {
		return nil, fmt.Errorf("Cannot find Connection for Addr %s\n",
			addr)
	}
	return conn, nil
}

func (s *Server) FindConnByUserId(user int) (*Conn, error) {

	conn, ok := s.Routes_By_UserId[user]
	if !ok {
		return nil, fmt.Errorf("Cannot find Connection for UserId %d\n",
			user)
	}
	return conn, nil
}

func (s *Server) DNSRecv() {
	b := make([]byte, DEF_BUF_SIZE)
	for {
		n, rpaddr, err := s.DNS.Conn.ReadFromUDP(b)
		if err != nil {
			Error.Println(err)
			continue
		}

		dnsPacket := new(dns.Msg)
		err = dnsPacket.Unpack(b[:n])
		if err != nil {
			Error.Println(err)
			continue
		}

		//Debug.Printf("Recv DNS packet:\n%s\n------", dnsPacket.String())
		tunPacket, err := s.DNS.Retrieve(dnsPacket) // TODO
		if err != nil {
			Error.Println(err)
			continue
		}

		switch tunPacket.GetCmd() {
		case TUN_CMD_CONNECT:

			// create new connection for the client
			rvaddr, userId := s.nextUserVAddr, s.nextUserId
			conn := s.NewConn(rvaddr, userId)
			s.Routes_By_VAddr[rvaddr.String()] = conn
			s.Routes_By_UserId[userId] = conn
			err = s.nextUserInfo()
			if err != nil {
				Error.Println(err)
				continue
			}

			t := &TUNResponsePacket{TUN_CMD_RESPONSE,
				userId,
				s.VAddr,
				rvaddr,
				dnsPacket}
			err := s.DNS.Reply(dnsPacket, t, rpaddr)
			if err != nil {
				Error.Println(err)
				continue
			}
			Debug.Printf("Connected with %s\n", conn.PAddr.String())

		case TUN_CMD_EMPTY:
            if conn, ok := s.Routes_By_UserId[tunPacket.GetUserId()]; ok {
                conn.Reply(dnsPacket, rpaddr)
			}
		case TUN_CMD_KILL:

			conn, err := s.FindConnByUserId(tunPacket.GetUserId())
			if err != nil {
				Error.Println(err)
				continue
			}

			delete(s.Routes_By_UserId, conn.UserId)
			delete(s.Routes_By_VAddr, conn.VAddr.String())
			// option: remove user from user pool
			// remove vaddr from vaddr pool
			Debug.Printf("Close Conn with %s\n", conn.VAddr.String())

			// normal reply
			t := &TUNCmdPacket{TUN_CMD_ACK, conn.UserId}
			err = s.DNS.Reply(dnsPacket, t, rpaddr)
			if err != nil {
				Error.Println(err)
				continue
			}

		case TUN_CMD_DATA:
       //Debug.Printf("Recv DNS packet:\n%s\n------", dnsPacket.String())
			conn, err := s.FindConnByUserId(tunPacket.GetUserId())
			if err != nil {
				Error.Println(err)
				continue
			}

			err = conn.Recv(tunPacket)
			if err != nil {
				Error.Println(err)
				continue
			}

			// normal reply this message
			err = s.DNS.Reply(dnsPacket, tunPacket, rpaddr)
			if err != nil {
				Error.Println(err)
				continue
			}

		default:
			// Reply with normal DNS Response
			Error.Println("Invalid TUN Cmd -- Not Implemented")
		}
	}
}

func (s *Server) TUNRecv() {

	b := make([]byte, DEF_BUF_SIZE)
	for {

		n, err := s.TUN.Read(b)
		if err != nil {
			Error.Println(err)
			continue
		}

		ippkt := new(ip.IPPacket)
		err = ippkt.Unmarshal(b[:n])
		if err != nil {
			Error.Println(err)
			continue
		}
		//Debug.Printf("TUNRecv: IP Packet from %s to %s\n",ip.IPAddrInt2Str(ippkt.Header.Src),ip.IPAddrInt2Str(ippkt.Header.Dst))

		rvaddrStr := ip.IPAddrInt2Str(ippkt.Header.Dst)

		conn, err := s.FindConnByVAddr(rvaddrStr)
		if err != nil {
			Debug.Println(err)
			continue
		}

		tunPacket := new(TUNIpPacket)
		tunPacket.Cmd = TUN_CMD_EMPTY // downstream
		tunPacket.Id = int(ippkt.Header.Id)
		tunPacket.Offset = 0
		tunPacket.More = false
		tunPacket.Payload = b[:n]

		conn.InChan <- tunPacket

		/*
		   msgs, err := s.DNS.Inject(tunPacket)
		   if err != nil {
		       Error.Println(err)
		       continue
		   }

		   for i:=0; i<len(msgs); i++{
		       conn.InChan <- msgs[i]
		       continue
		   }*/
	}
}

func (s *Server) SendString(c *Conn, str string) {
    tunPkt := new(TUNIpPacket)
    tunPkt.UserId = c.UserId
    tunPkt.Id = DEF_SENDSTRING_ID
    tunPkt.Cmd = TUN_CMD_EMPTY
    tunPkt.Payload = []byte(str)
    c.InChan <- tunPkt

}

func (s *Server) Info() {
	fmt.Printf("\nserver vip: %s, number of connections: %d\n", s.VAddr.String(), len(s.Routes_By_VAddr))
	for _, v := range s.Routes_By_VAddr {
		v.Info()
	}
}

func (c *Conn) Info() {
	fmt.Printf("\tvip:%s, userId:%d\n", c.VAddr.String(), c.UserId)
}
