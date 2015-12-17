package ip

import (
    "fmt"
    "net"
)

func NewIPAddrFromInt(addr uint32) (*IPAddr){
    ip := new(IPAddr)
    ip.addr = addr
    return ip
}

func NewIPAddrFromNet(addr *net.IPAddr) (*IPAddr, error){
    return NewIPAddr(addr.String())
}

func NewIPAddr(addrStr string) (*IPAddr, error){

    ip := new(IPAddr)

    addr, err := IPAddrStr2Int(addrStr)
    if err != nil {
        return nil, err
    }
    ip.addr = addr
    return ip, nil
}

func (ip *IPAddr) ToInt() uint32{
    return ip.addr
}

func (ip *IPAddr) String() string{
    return IPAddrInt2Str(ip.addr)
}

func (ip *IPAddr) FromNetIPAddr(netAddr *net.IPAddr) error{

    addrStr := netAddr.String()
    newIP, err := NewIPAddr(addrStr)
    if err != nil {
        return err
    }
    ip.addr = newIP.addr

    return nil
}

func (ip *IPAddr) ToNetIPAddr() (*net.IPAddr, error){

    return net.ResolveIPAddr("ip", ip.String())
}

func IPAddrInt2Str(addr uint32) string{

    return fmt.Sprintf("%d.%d.%d.%d",
                (addr>>24)&0xff,
                (addr>>16)&0xff,
                (addr>>8 )&0xff,
                (addr) & 0xff)
}

func IPAddrStr2Int(addrStr string) (uint32, error){

    var addr uint32 = 0
    var a,b,c,d uint32 = 0, 0, 0, 0
    _, err := fmt.Sscanf(addrStr, "%d.%d.%d.%d", &a, &b, &c, &d)
    if err != nil {
        return 0, fmt.Errorf("Invalid Address: %s\n", addrStr)
    }

    addr = (a<<24) | (b<<16) | (c<<8) | d
    return addr, nil
}

/*

func IP2Int(p *net.IPAddr) uint32 {

    ip := p.IP

    var num uint32
    num = (uint32(ip[12])<<24) | (uint32(ip[13])<<16) |
          (uint32(ip[14])<<8)  |  uint32(ip[15])

    return num
}
func Int2IP(ip uint32) net.IPAddr{

    ipaddr := net.IPv4(
               byte((ip>>24)&0xff), byte((ip>>16)&0xff),
               byte((ip>>8)&0xff),  byte(ip&0xff))
    return net.IPAddr{ipaddr,""}
}
*/


