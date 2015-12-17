
package ip

import (

)
const (

    IP_DEF_PROTOCOL uint8 = 0

    IP_DEF_VS uint8 = 4
    IP_DEF_HLEN uint8 = 20
    IP_DEF_TTL  uint8  = 16

    IP_MAX_SIZE uint16 = 0xffff
    IP_MAX_PAYLOAD uint16 = IP_MAX_SIZE - uint16(IP_DEF_HLEN)

    IP_RF uint16 = 0x8000
    IP_DF uint16 = 0x4000
    IP_MF uint16 = 0x2000
    IP_OFFMASK uint16 = 0x1fff

    MTU uint32 = 1500 //

    FRAG_MAX_PAYLOAD uint32 = MTU - uint32(IP_DEF_HLEN)

    IP_INFINITY_COST uint32 = 16
)


type IPHeader struct{
    Vhl uint8
    Tos uint8
    Len uint16

    Id  uint16
    Off uint16

    Ttl uint8
    Ptl uint8
    Sum uint16

    Src uint32
    Dst uint32
    // ip_option []uint8
}

type IPPacket struct{
    Header IPHeader
    Payload []byte
}

type IPAddr struct {
    addr uint32
}


