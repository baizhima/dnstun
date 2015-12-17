package tun

import (
	"../ip"
	"../tonnerre/golang-dns"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"net"
	"strconv"
	"strings"
)

const (
	DNS_Client int = 0
	DNS_Server int = 1
)

type DNSUtils struct {
	Kind      int
	Conn      *net.UDPConn
	TopDomain string
	LAddr     *net.UDPAddr
	LDns      *net.UDPAddr
}

func NewDNSClient(laddrstr, ldnsstr, topDomain string) (*DNSUtils, error) {

	d := new(DNSUtils)
	d.Kind = DNS_Client
	d.TopDomain = topDomain

	var err error
	d.LDns, err = net.ResolveUDPAddr("udp", ldnsstr)
	if err != nil {
		return nil, err
	}

	d.LAddr, err = net.ResolveUDPAddr("udp", laddrstr)
	if err != nil {
		return nil, err
	}

	/* Listen on UDP address laddr */
	d.Conn, err = net.ListenUDP("udp", d.LAddr)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func NewDNSServer(laddrstr, topDomain string) (*DNSUtils, error) {

	d := new(DNSUtils)
	d.Kind = DNS_Server
	d.TopDomain = topDomain

	var err error
	d.LAddr, err = net.ResolveUDPAddr("udp", laddrstr)
	if err != nil {
		return nil, err
	}
	d.LDns = d.LAddr

	/* Listen on UDP address laddr */
	d.Conn, err = net.ListenUDP("udp", d.LAddr)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (d *DNSUtils) NewDNSPacket(t TUNPacket) (*dns.Msg, error) {

	switch t.GetCmd() {
	case TUN_CMD_CONNECT:
		labels := []string{string(TUN_CMD_CONNECT), d.TopDomain}
		domain := strings.Join(labels, ".")

		msg := new(dns.Msg)
		msg.SetQuestion(domain, dns.TypeTXT)
		msg.RecursionDesired = true
		return msg, nil

	default:
		return nil, fmt.Errorf("NewDNSPacket: Invalid TUN CMD\n")
	}
}

func (d *DNSUtils) Send(p []byte) error {
	if d.Kind != DNS_Client {
		return fmt.Errorf("Send: Only used by Client\n")
	}
	_, err := d.Conn.WriteToUDP(p, d.LDns)
	return err
}

func (d *DNSUtils) SendTo(addr *net.UDPAddr, p []byte) error {

	_, err := d.Conn.WriteToUDP(p, addr)
	return err
}

func (d *DNSUtils) Reply(msg *dns.Msg, tun TUNPacket, paddr *net.UDPAddr) error {
	var msgs []*dns.Msg
	var err error
	switch tun.GetCmd() {
	case TUN_CMD_RESPONSE, TUN_CMD_ACK:
		msgs, err = d.Inject(tun, nil)
		if err != nil {
			Error.Println(err)
            fmt.Printf("cmd :%s\n", string(tun.GetCmd()))
            fmt.Println(msg.String())
			return err
	    }
    // upstream 
	case TUN_CMD_DATA:
        // not appropriate to use inject(tun) here
        // just ack
        // TODO currently msgs will be dropped by intermediate DNS Server due to UDP limit 500bytes
        t := new(TUNAckPacket)
        t.Cmd = TUN_CMD_ACK
        t.UserId = tun.GetUserId()
        t.Request = msg
		msgs, err = d.Inject(t, msg)
        //fmt.Printf("msg to reply\n")
        //fmt.Println(msgs[0].String())
		if err != nil {
			return err
		}
    // downstreaming
    case TUN_CMD_EMPTY:
        msgs, err = d.Inject(tun, msg)
        if err != nil {
            return err
        }

	default:
		return fmt.Errorf("DNS Reply: Invalid TUN Cmd %s\n", string(tun.GetCmd()))
	}
	for _, msg := range msgs {

		binary, err := msg.Pack()
		if err != nil {
			return err
		}
		err = d.SendTo(paddr, binary)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *DNSUtils) Inject(tun TUNPacket, request *dns.Msg) ([]*dns.Msg, error) {

	msgs := make([]*dns.Msg, 0)

	switch tun.GetCmd() {
	case  TUN_CMD_EMPTY:
        if request != nil {
            // downstream 
            t, ok := tun.(*TUNIpPacket)
            if !ok {
                return nil, fmt.Errorf("cannot cast to TUNIpPacket")
            }
            return d.InjectIPPacket(t.UserId, t.Id, t.Payload, request)
		} else {
            // upstream
            t, ok := tun.(*TUNCmdPacket)
            if !ok {
                return nil, fmt.Errorf("cannot cast to TUNCmdPacket")
            }
            msg := new(dns.Msg)
            labels := []string{strconv.Itoa(t.UserId), string(TUN_CMD_EMPTY), d.TopDomain}
            domain := strings.Join(labels, ".")
            msg.SetQuestion(domain, dns.TypeTXT)
            msg.RecursionDesired = true
            msgs = append(msgs, msg)
            return msgs, nil
        }
    case TUN_CMD_DATA:
        t, ok := tun.(*TUNIpPacket)
        if !ok {
            return nil, fmt.Errorf("Invaild Conversion\n")
        }
        return d.InjectIPPacket(t.UserId, t.Id, t.Payload, request)
	case TUN_CMD_CONNECT:
		msg, err := d.NewDNSPacket(tun)
		if err != nil {
			Error.Println(err)
			return nil, err
		}
		msgs = append(msgs, msg)
		return msgs, nil


	case TUN_CMD_KILL:
		Error.Println("Inject for CMD_KILL not implemented")
		return nil, nil

	case TUN_CMD_RESPONSE, TUN_CMD_ACK:
		var replyStr string
		var ans dns.RR
		var err error
		reply := new(dns.Msg)
		reply.Answer = make([]dns.RR, 1)
		if tun.GetCmd() == TUN_CMD_RESPONSE {
			tunPkt, ok := tun.(*TUNResponsePacket)
			if !ok {
				return nil, fmt.Errorf("error casting to TUNResponsePacket\n")
			}
			domain := tunPkt.Request.Question[0].Name
			ans, err = dns.NewRR(domain + " 0 IN TXT xx")
			ans.(*dns.TXT).Txt = make([]string, 3)
			if err != nil {
				return nil, err
			}
			reply.SetReply(tunPkt.Request)
			serverIpStr := strings.Replace(tunPkt.Server.String(), ".", "_", -1)
			clientIpStr := strings.Replace(tunPkt.Client.String(), ".", "_", -1)
			replyDomains := []string{string(TUN_CMD_RESPONSE), strconv.Itoa(tunPkt.UserId), serverIpStr, clientIpStr}
			replyStr = strings.Join(replyDomains, ".")
		} else if tun.GetCmd() == TUN_CMD_ACK {
			tunPkt, ok := tun.(*TUNAckPacket)
			if !ok {
				return nil, fmt.Errorf("error casting to TUNAckPacket\n")
			}
			domain := tunPkt.Request.Question[0].Name
			ans, err = dns.NewRR(domain + " 0 IN TXT xx")
			ans.(*dns.TXT).Txt = make([]string, 3)
			if err != nil {
				return nil, err
			}
			reply.SetReply(tunPkt.Request)
			replyStr = string(TUN_CMD_ACK)
		} else {
            fmt.Printf("TUN_CMD %s\n not handled\n", string(tun.GetCmd()))
        }
		ans.(*dns.TXT).Txt[0] = replyStr
		reply.Answer[0] = ans
		msgs = append(msgs, reply)
		return msgs, nil
	default:
		return nil, fmt.Errorf("Invalid TUN CMD %s", tun.GetCmd())
	}

	return nil, fmt.Errorf("Not Implement\n")
}

/* Given a DNS Packet, Retrieve TUNPacket from it */
func (d *DNSUtils) Retrieve(in *dns.Msg) (TUNPacket, error) {

	if len(in.Answer) > 0 {
		// dns packet sent from DNSServer
		ans, ok := in.Answer[0].(*dns.TXT)
		if !ok {
			return nil, fmt.Errorf("unexpected reply RR record, not TXT\n")
		}
		cmdDomains := strings.Split(ans.Txt[0], ".")
		cmd := byte(cmdDomains[0][0])
		var err error
		switch cmd {
		case TUN_CMD_RESPONSE:
			t := new(TUNResponsePacket)
			t.Cmd = TUN_CMD_RESPONSE
			t.UserId, err = strconv.Atoi(cmdDomains[1])
			if err != nil {
				return nil, err
			}
			serverIpStr := strings.Replace(cmdDomains[2], "_", ".", -1)
			clientIpStr := strings.Replace(cmdDomains[3], "_", ".", -1)
			t.Server, err = net.ResolveIPAddr("ip", serverIpStr)
			if err != nil {
				return nil, err
			}
			t.Client, err = net.ResolveIPAddr("ip", clientIpStr)
			if err != nil {
				return nil, err
			}
			return t, nil
		case TUN_CMD_ACK:
			t := new(TUNCmdPacket)
			t.Cmd = TUN_CMD_ACK
			return t, nil
        case TUN_CMD_DATA:
        // downstream data transmission
            userId, err := strconv.Atoi(cmdDomains[1])
            if err != nil {
                return nil, err
            }
            ipId, err := strconv.Atoi(cmdDomains[2])
            if err != nil {
                return nil, err
            }
            mf, err := strconv.Atoi(cmdDomains[3])
            if err != nil {
                return nil, err
            }
            offset, err := strconv.Atoi(cmdDomains[4])
            if err != nil {
                return nil, err
            }
            t := new(TUNIpPacket)
            t.Cmd = TUN_CMD_DATA
            t.UserId = userId
            t.Id = ipId
            t.More = (mf == 1)
            t.Offset = offset
            encodedStr := strings.Replace(strings.Join(ans.Txt[1:3], ""), "_", "", -1)
            raw, err := base64.StdEncoding.DecodeString(encodedStr)
            if err != nil {
                return nil, err
            }
            t.Payload = raw
            return t, nil
		default:
			return nil, fmt.Errorf("TUN_CMD %s from DNSServer not implemented \n", string(cmd))
		}

		return nil, fmt.Errorf("DNSUtils.Retrieve should not be here")

	} else {
		// dns packet sent from DNSClient
		r := in.Question[0]
		domains := strings.Split(r.Name[:len(r.Name)-1], ".") // trim the last '.'from "b.jannotti.com."[-1]
		n := len(domains)
		if n < 4 {
			return nil, fmt.Errorf("unexpecetd domain name format %s\n", r.Name)
		}
		cmd := byte(domains[n-4][0])
		if cmd != TUN_CMD_CONNECT && n < 5 {
			return nil, fmt.Errorf("unexpecetd domain name format %s\n", r.Name)
		}
		switch cmd {
		case TUN_CMD_CONNECT:
			t := new(TUNCmdPacket)
			t.Cmd = cmd
			t.UserId = -1 // has not been allocated by DNSServer
            return t, nil
		case TUN_CMD_DATA:
			t := new(TUNIpPacket)
			t.Cmd = cmd
			ipId, err := strconv.Atoi(domains[n-8])
            userId, err2 := strconv.Atoi(domains[n-5])
            t.More = (domains[n-6] == "1")
            offset, err3 := strconv.Atoi(domains[n-7])
			if err != nil || err2 != nil || err3 != nil{
				return nil, fmt.Errorf("error casting ipId or userId or offset")
			}
            t.Offset = offset
			t.Id = ipId
            t.UserId = userId

            encodedStr := strings.Replace(strings.Join(domains[:4], ""), "_", "", -1)
            raw, err := base32.StdEncoding.DecodeString(encodedStr)
            if err != nil {
                return nil, fmt.Errorf("error decode SendString's string")
            }
			if ipId == DEF_SENDSTRING_ID {
				fmt.Printf("recv: %s\n", string(raw))
			} else {
                t.Payload = raw
            }
			return t, nil
		default:
			var err error
			t := new(TUNCmdPacket)
			t.Cmd = cmd
			t.UserId, err = strconv.Atoi(domains[n-5])
			if err != nil {
				return nil, fmt.Errorf("cannot parse UserId from %s\n", domains[n-5])
			}
			return t, nil
		}
		fmt.Println("retrieve should not be here")
		return nil, nil
	}

}

func (d *DNSUtils) injectToLabels(b []byte, base int) ([]string, error) {
	var LABEL_SIZE int
	var encodedStr string
	var labelsPerDns int
	if base == DEF_UPSTREAM_ENCODING_BASE {
		LABEL_SIZE = DEF_UPSTREAM_LABEL_SIZE
		encodedStr = base32.StdEncoding.EncodeToString(b)
		labelsPerDns = DEF_UPSTREAM_LABELS_PER_DNS
	} else if base == DEF_DOWNSTREAM_ENCODING_BASE {
		LABEL_SIZE = DEF_DOWNSTREAM_LABEL_SIZE
		encodedStr = base64.StdEncoding.EncodeToString(b)
		labelsPerDns = DEF_DOWNSTREAM_LABELS_PER_DNS
	} else {
		return nil, fmt.Errorf("unsupported encoding base")
	}

	numLabels := len(encodedStr) / LABEL_SIZE
	labelsArr := make([]string, 0)

	for i := 0; i < numLabels; i++ {
		labelsArr = append(labelsArr, encodedStr[i*LABEL_SIZE:(i+1)*LABEL_SIZE])
	}
	// last label
	if len(encodedStr)%LABEL_SIZE != 0 {
		lastLabel := encodedStr[numLabels*LABEL_SIZE:]
	//	lastLabel += strings.Repeat("_", (LABEL_SIZE - len(lastLabel)))
		labelsArr = append(labelsArr, lastLabel)
	}

	// padding placeholder labels to labelsArr so that len(labelsArr) % labelsPerDns == 0
	for {
		if len(labelsArr)%labelsPerDns == 0 {
			break
		}
        labelsArr = append(labelsArr, "_")
		//labelsArr = append(labelsArr, strings.Repeat("_", LABEL_SIZE))
	}

	return labelsArr, nil
}

func (d *DNSUtils) InjectIPPacket(userId int, ipId int, b []byte, request *dns.Msg) ([]*dns.Msg, error) {
	msgs := make([]*dns.Msg, 0)
    var base, labelsPerDns, labelSize int
    var cmd byte
    if d.Kind == DNS_Client {
        // upstream
        base = DEF_UPSTREAM_ENCODING_BASE
        labelsPerDns = DEF_UPSTREAM_LABELS_PER_DNS
        labelSize = DEF_UPSTREAM_LABEL_SIZE
        cmd = TUN_CMD_DATA
    } else {
        // downstream
        base = DEF_DOWNSTREAM_ENCODING_BASE
        labelsPerDns = DEF_DOWNSTREAM_LABELS_PER_DNS
        labelSize = DEF_DOWNSTREAM_LABEL_SIZE
        cmd = TUN_CMD_EMPTY
    }
    ipIdStr := strconv.Itoa(ipId)
    userIdStr := strconv.Itoa(userId)
    labels, err := d.injectToLabels(b, base)
    if err != nil {
        return nil, err
    }
    for i := 0; i < len(labels)/labelsPerDns; i++ {
        currLabels := labels[i*labelsPerDns : (i+1)*labelsPerDns]
        encodedStr := strings.Join(currLabels, ".")
        var mf string = "1"
        if i == len(labels)/labelsPerDns-1 {
            mf = "0"
        }
        offsetStr := strconv.Itoa(i*labelSize)
//        var domainLabels string
        currMsg := new(dns.Msg)
        if d.Kind == DNS_Client {
            domainLabels := []string{encodedStr, ipIdStr, mf, offsetStr, userIdStr, string(cmd), d.TopDomain}
            domain := strings.Join(domainLabels, ".")
            if len(domain) > 253 {
                return nil, fmt.Errorf("Domain name %d > 253\n", len(domain))
            }
            currMsg.SetQuestion(domain, dns.TypeTXT)
            currMsg.RecursionDesired = true
        } else {
            firstTxt := strings.Join([]string{string(TUN_CMD_DATA), userIdStr, ipIdStr, mf, offsetStr},".")
            secondTxt := currLabels[0]
            thirdTxt := currLabels[1]
            currMsg.SetReply(request)
            currMsg.Answer = make([]dns.RR, 1)
            ans, err := dns.NewRR(currMsg.Question[0].Name + " 0 IN TXT xx")
            if err != nil {
                return nil, err
            }
            ans.(*dns.TXT).Txt = make([]string, 3)
            ans.(*dns.TXT).Txt[0] = firstTxt
            ans.(*dns.TXT).Txt[1] = secondTxt
            ans.(*dns.TXT).Txt[2] = thirdTxt
            currMsg.Answer[0] = ans
            //Debug.Println(currMsg.String())
        }
        msgs = append(msgs, currMsg)
    }

	return msgs, nil
}

/* inject ip packet */
func (d *DNSUtils) InjectAndSendTo(b []byte, userId int, addr *net.UDPAddr) error {

	ippkt := new(ip.IPPacket)
	err := ippkt.Unmarshal(b)
	if err != nil {
		return err
	}

	id := ippkt.Header.Id

	t := new(TUNIpPacket)
	t.Cmd = TUN_CMD_DATA
	t.Id = int(id)
    t.UserId = userId
	t.More = false
	t.Offset = 0
	t.Payload = b

	msgs, err := d.Inject(t, nil)
    //Debug.Printf("msg to send \n%s\n-----\n", msgs)
	if err != nil {
		return err
	}

	for i := 0; i < len(msgs); i++ {

		pkt, err := msgs[i].Pack()
		if err != nil {
			return err
		}

		err = d.SendTo(addr, pkt)
		if err != nil {
			return err
		}
	}
	return nil
}
