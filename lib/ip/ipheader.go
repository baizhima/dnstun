package ip

import (
//    "net"
    "bytes"
    "encoding/binary"
    "fmt"
)

func (p *IPHeader) String() string {
    return fmt.Sprintf(
        "|vhl:%x|tos:%v|len:%v|id:%v|off:%x|ttl:%v|ptl:%v|chksum:%x\n|src:%v|dst:%v|",
        p.Vhl,
        p.Tos,
        p.Len,
        p.Id,
        p.Off,
        p.Ttl,
        p.Ptl,
        p.Sum,
        NewIPAddrFromInt(p.Src).String(),
        NewIPAddrFromInt(p.Dst).String())
}

var staticIphdrId uint16 = 0

func IPNewId() uint16 {
    staticIphdrId += 1
    return staticIphdrId
}

func (p *IPHeader) init(
                ptl uint8,
                dst uint32,
                src uint32,
                plen uint16,
                off  uint16){

    p.Vhl = (IP_DEF_VS<<4)|(IP_DEF_HLEN/4)
    p.Tos = 0    // not important
    p.Len = uint16(IP_DEF_HLEN) + plen
    p.Id  = IPNewId()  // set late

    p.Off = off

    p.Ttl = IP_DEF_TTL
    p.Ptl = ptl
    p.Sum = 0    // set later

    p.Src = src
    p.Dst = dst

    p.SetCheckSum()
}

func (p *IPHeader) Hlen() uint8 {
    return (p.Vhl&0x0f)*4
}

func (p *IPHeader) calCheckSum() uint16{

    var sum uint32 = 0

    sum += (uint32(p.Vhl)<<8) | uint32(p.Tos)
    sum += uint32(p.Len)
    sum += uint32(p.Id)
    sum += uint32(p.Off)
    sum += (uint32(p.Ttl)<<8) | uint32(p.Ptl)


    var src uint32 = p.Src
    var dst uint32 = p.Dst
    //var src uint32 = IP2Int(&p.ip_src)
    //var dst uint32 = IP2Int(&p.ip_dst)

    sum += ((src>>16)&0xffff) + (src&0xffff)
    sum += ((dst>>16)&0xffff) + (dst&0xffff)

    sum = ((sum>>16)&0xffff) + (sum&0xffff)

    return uint16(sum)
}

func (p *IPHeader) VerifyCheckSum() bool{
    sum := p.calCheckSum()
    sum += p.Sum
    return (sum == uint16(0xffff))
}

func (p* IPHeader) SetCheckSum(){

    sum := p.calCheckSum()
    p.Sum = uint16(^sum)
}


func (h *IPHeader)Marshal() ([]byte, error) {

    buf := new(bytes.Buffer)

    var err error

    err = binary.Write(buf, binary.BigEndian, h.Vhl)
    if err != nil {
        return nil, err
    }
    err = binary.Write(buf, binary.BigEndian, h.Tos)
    if err != nil {
        return nil, err
    }
    err = binary.Write(buf, binary.BigEndian, h.Len)
    if err != nil {
        return nil, err
    }
    err = binary.Write(buf, binary.BigEndian, h.Id)
    if err != nil {
        return nil, err
    }
    err = binary.Write(buf, binary.BigEndian, h.Off)
    if err != nil {
        return nil, err
    }
    err = binary.Write(buf, binary.BigEndian, h.Ttl)
    if err != nil {
        return nil, err
    }
    err = binary.Write(buf, binary.BigEndian, h.Ptl)
    if err != nil {
        return nil, err
    }

    err = binary.Write(buf, binary.BigEndian, h.Sum)
    if err != nil {
        return nil, err
    }

    //err = binary.Write(buf, binary.BigEndian, IP2Int(&h.ip_src))

    err = binary.Write(buf, binary.BigEndian, h.Src)
    if err != nil {
        return nil, err
    }

    err = binary.Write(buf, binary.BigEndian, h.Dst)
    if err != nil {
        return nil, err
    }

    return buf.Bytes(), nil
}

func (h *IPHeader) Unmarshal( data []byte) error{

    var err error
    buf := bytes.NewBuffer(data)

    err = binary.Read(buf, binary.BigEndian, &h.Vhl)
    if err != nil {
        return err
    }
    err = binary.Read(buf, binary.BigEndian, &h.Tos)
    if err != nil {
        return err
    }
    err = binary.Read(buf, binary.BigEndian, &h.Len)
    if err != nil {
        return err
    }
    err = binary.Read(buf, binary.BigEndian, &h.Id)
    if err != nil {
        return err
    }
    err = binary.Read(buf, binary.BigEndian, &h.Off)
    if err != nil {
        return err
    }
    err = binary.Read(buf, binary.BigEndian, &h.Ttl)
    if err != nil {
        return err
    }
    err = binary.Read(buf, binary.BigEndian, &h.Ptl)
    if err != nil {
        return err
    }
    err = binary.Read(buf, binary.BigEndian, &h.Sum)
    if err != nil {
        return err
    }

    var src uint32
    var dst uint32
    err = binary.Read(buf, binary.BigEndian, &src)
    if err != nil {
        return err
    }
    err = binary.Read(buf, binary.BigEndian, &dst)
    if err != nil {
        return err
    }

    h.Src = src
    h.Dst = dst
    //h.ip_src = Int2IP(src)
    //h.ip_dst = Int2IP(dst)

    return nil
}
