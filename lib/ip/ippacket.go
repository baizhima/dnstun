package ip

import (
    "bytes"
    "encoding/binary"
    "fmt"
)

func (p *IPPacket)Marshal() ([]byte, error){

    buf := new(bytes.Buffer)

    hdr, err := p.Header.Marshal()
    if err != nil{
        return nil, err
    }

    _, err = buf.Write(hdr)
    if err != nil{
        return nil, err
    }

    err = binary.Write(buf, binary.BigEndian, p.Payload)
    if err != nil{
        return nil, err
    }

    return buf.Bytes(), nil
}

func (p *IPPacket)Unmarshal(data []byte) error{

    err := p.Header.Unmarshal(data)
    if err != nil{
        return err
    }

    hlen := p.Header.Hlen()

    // copy
    p.Payload = []byte(data[hlen:])
    return nil
}

func (p *IPPacket) String() string{
    return fmt.Sprintf("\n\nHEADER:%v\nPAYLOAD:%v\n\n",
                    p.Header.String(), p.Payload)
}
