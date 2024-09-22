package object

import (
	"github.com/google/gopacket"
)

type NetworkPacketSource struct {
	PacketSource gopacket.PacketSource
}

func (p *NetworkPacketSource) Type() ObjectType { return PACKET_OBJ }
func (p *NetworkPacketSource) Inspect() string  { return "" }
