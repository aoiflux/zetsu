package pcapng

import (
	"fmt"
	"zetsu/object"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

// ParsePcap reads and processes packets from a pcap file
func ParsePcap(args ...object.Object) object.Object {
	if len(args) != 1 {
		return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
	}

	if args[0].Type() != object.STRING_OBJ {
		return &object.Error{Message: fmt.Sprintf("first argument to `ParsePcap` must be STRING, got=%s", args[0].Type())}
	}

	handle, err := pcap.OpenOffline(args[0].Inspect())
	if err != nil {
		return &object.Error{Message: fmt.Sprintf("failed to open pcap file: %v", err)}
	}
	defer handle.Close()

	// Create a packet source to iterate over packets
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	return &object.NetworkPacketSource{PacketSource: *packetSource}
	// for packet := range packetSource.Packets() {
	// 	processPacket(packet)
	// }
}

// processPacket handles each individual packet
// func processPacket(packet gopacket.Packet) {
// 	fmt.Println("Packet captured: ", packet)
// 	// You can analyze layers, payload, etc.
// }
