package Network


import (
"net"
"fmt"
"log"
"os"
"strconv"
"bytes"
"encoding/gob"
)

type Packet struct {
	Type string
	Postition int
	Order int
}

const bcast = "129.241.187.157"
const udpPort = "2878"
const localAdress = "78.91.16.139"


func TransmitUDP(transmitChannel chan Packet){
	remoteAddr, err := net.ResolveUDPAddr( "udp", net.JoinHostPort( bcast, udpPort ) )
	if err != nil {
		log.Fatal("Failed to resolve UDP remote address:", err)
	}
	localAddr, err := net.ResolveUDPAddr( "udp", net.JoinHostPort("", udpPort ) )
	if err != nil {
		log.Fatal("Failed to resolve UDP local address:", err)
	}
	sendConnection, err := net.ListenUDP( "udp", localAddr)
	if err != nil {
		log.Fatal("UDP send connection error on " + remoteAddr.String() + ": ", err)
	}

	defer sendConnection.Close()
	fmt.Fprintln( os.Stderr, "Send connection established." )

	var UDPpacketBuffer bytes.Buffer
	UDPpacketEncoder := gob.NewEncoder(&UDPpacketBuffer)
	sendBufferRaw := make( []byte, 1600)
	for {
		var newPacket = <- transmitChannel
		if newPacket.Type == "Terminate" {
			fmt.Fprintln( os.Stderr, "Aborting transmission..." )
			break
		}
		fmt.Fprintln( os.Stderr, "Sending packet: <" + newPacket.Type + strconv.Itoa(newPacket.Postition) + ">" )
		err = UDPpacketEncoder.Encode(newPacket)
		if err != nil {
			log.Fatal("Unable to encode new packet. ", err)
		}
		UDPpacketBuffer.Read(sendBufferRaw)
		
		_, err = sendConnection.WriteToUDP(sendBufferRaw, remoteAddr)
		if err != nil {
			log.Fatal("Failed to send Packet from buffer: ", err)
		}
	}
	
	fmt.Fprintln( os.Stderr, "Connection terminated. Good bye." )
	return
}

