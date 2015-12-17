package tun

import (
	"../tonnerre/golang-dns"
	"fmt"
	"net"
	"time"
)

type Client struct {
	ClientVAddr *net.IPAddr
	ServerVAddr *net.IPAddr
	UserId      int

	DNS *DNSUtils
	TUN *Tunnel

	Buffer map[int][]byte

	Running bool
}

func NewClient(topDomain, ldns, laddr, tunName string) (*Client, error) {

	c := new(Client)
	c.Running = false

	/* Will be filled after connected with server */
	c.ClientVAddr = nil
	c.ServerVAddr = nil
	c.UserId = -1

	var err error
	c.DNS, err = NewDNSClient(laddr, ldns, topDomain)
	if err != nil {
		return nil, err
	}
	c.TUN, err = NewTunnel(tunName)
	if err != nil {
		return nil, err
	}

	c.Buffer = make(map[int][]byte)
	return c, nil
}

func (c *Client) DNSSendFreeId() {

	for c.Running {
		time.Sleep(800 * time.Millisecond)

		t := new(TUNCmdPacket)
		t.Cmd = TUN_CMD_EMPTY
		t.UserId = c.UserId
		msgs, err := c.DNS.Inject(t, nil)
		if err != nil {
			Error.Println(err)
			continue
		}
		err = c.sendDNSMessages(msgs)
		if err != nil {
			Error.Println(err)
		}
	}
}

func (c *Client) Connect() error {

	// Create a TUN Packet
	tunPacket := new(TUNCmdPacket)
	tunPacket.Cmd = TUN_CMD_CONNECT

	// Inject the TUN Packet to a DNS Packet
	msgs, err := c.DNS.Inject(tunPacket, nil)
	if err != nil {
		Error.Println(err)
		return err
	}

	//Debug.Println(msgs[0].String())

	// Listening on the port, wating for incoming DNS Packet
	go c.DNSRecv()

	// Send DNS Packet to Local DNS Server
	for i := 0; i < len(msgs); i++ {
		packet, err := msgs[i].Pack()
		if err != nil {
			Error.Println(err)
			return err
		}
		err = c.DNS.Send(packet)
		if err != nil {
			Error.Println(err)
			return err
		}
	}
	return nil
}

func (c *Client) DNSRecv() {

	b := make([]byte, DEF_BUF_SIZE)
	for {
		// rpaddr : the public UDP Addr of remote DNS Server
		n, _, err := c.DNS.Conn.ReadFrom(b)
		if err != nil {
			Error.Println(err)
		}

		dnsPacket := new(dns.Msg)
		err = dnsPacket.Unpack(b[:n])
		if err != nil {
			Error.Println(err)
			continue
		}

		if dns.RcodeToString[dnsPacket.MsgHdr.Rcode] == "SERVFAIL" {
			fmt.Printf("ignore servfail\n")
            if len(dnsPacket.Question[0].Name) < 150 {
			Debug.Printf("Recv DNS Packet:\n%s\n--------------", dnsPacket.String())
            } else {
                fmt.Printf("reason: question's name too long to respond in UDP\n")
            }
			continue
		}

		tunPacket, err := c.DNS.Retrieve(dnsPacket)
		if err != nil {
			Error.Println(err)
			continue
		}

		switch tunPacket.GetCmd() {
		case TUN_CMD_RESPONSE:
			if c.Running == false {
				res, ok := tunPacket.(*TUNResponsePacket)
				if !ok {
					Error.Println("Fail to Convert TUN Packet\n")
					continue
				}
				c.UserId = res.UserId
				c.ServerVAddr = res.Server
				c.ClientVAddr = res.Client
				fmt.Printf("connection established. server vip: %s, client vip: %s\n",
					c.ServerVAddr.String(), c.ClientVAddr.String())

				c.Running = true
				go c.TUNRecv()
				go c.DNSSendFreeId()
			}

		case TUN_CMD_DATA:

			if c.Running == true {

				t, ok := tunPacket.(*TUNIpPacket)
				if !ok {
					Error.Println("Fail to Convert TUN Packet\n")
					continue
				}
                if t.Id == DEF_SENDSTRING_ID {
                    fmt.Printf("recv %s\n", string(t.Payload))
                    continue
                }
				c.TUN.Save(c.Buffer, t)
			}
		case TUN_CMD_ACK:
			if c.Running {
				//fmt.Println("ACK from DNSServer")
			}
		default:
			Debug.Println("Invalid TUN Cmd")
		}
	}
}

func (c *Client) TUNRecv() {

	b := make([]byte, DEF_BUF_SIZE)
	for c.Running == true {

		n, err := c.TUN.Read(b)
		if err != nil {
			Error.Println(err)
			continue
		}

		err = c.DNS.InjectAndSendTo(b[:n], c.UserId, c.DNS.LDns)
		if err != nil {
			Error.Println(err)
			continue
		}
	}
}


/*
// base32-encoded string -> base32-encoded []string
func (c *Client) buildLabels(str string) []string {
	labelsArr := make([]string, 0)
	numLabels := len(str) / DEF_UPSTREAM_LABEL_SIZE
	for i := 0; i < numLabels; i++ {
		labelsArr = append(labelsArr, str[i*DEF_UPSTREAM_LABEL_SIZE:(i+1)*DEF_UPSTREAM_LABEL_SIZE])
	}
	// padding the last partially filled label
	if len(str)%DEF_UPSTREAM_LABEL_SIZE != 0 {
		lastLabel := str[numLabels*DEF_UPSTREAM_LABEL_SIZE:]
		lastLabel += strings.Repeat("_", (DEF_UPSTREAM_LABEL_SIZE - len(lastLabel)))
		labelsArr = append(labelsArr, lastLabel)
	}
	// padding 1-3 empty labels to labelsArr so that len(labelsArr)%4 == 0
	for len(labelsArr)%4 != 0 {
		labelsArr = append(labelsArr, strings.Repeat("_", DEF_UPSTREAM_LABEL_SIZE))
	}
	return labelsArr
}
*/

func (c *Client) sendDNSMessages(msgs []*dns.Msg) error {
	for _, msg := range msgs {
		binary, err := msg.Pack()
		if err != nil {
			return err
		}
		err = c.DNS.Send(binary)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) SendString(str string) {
	if c.Running {
		tunPkt := new(TUNIpPacket)
		tunPkt.Cmd = TUN_CMD_DATA
        tunPkt.UserId = c.UserId
		tunPkt.Id = DEF_SENDSTRING_ID
		tunPkt.Payload = []byte(str)
		msgs, err := c.DNS.Inject(tunPkt, nil)
		if err != nil {
			fmt.Errorf("err")
			return
		}
		err = c.sendDNSMessages(msgs)
		if err != nil {
			Error.Println(err)
		}
	} else {
		fmt.Println("no connection")
	}
}

func (c *Client) Info() {
    fmt.Printf("\n")
	fmt.Printf("client userId: %d, server vip:%s, client vip:%s\n", c.UserId, c.ServerVAddr.String(),
		c.ClientVAddr.String())
	fmt.Printf("running: %t\n", c.Running)
}
