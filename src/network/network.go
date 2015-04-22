// NEW

package network

import "queueManager"

import (
"net"
"log"
"os"
"strconv"
"bytes"
"encoding/gob"
"sort"
"time"
)

//TODO:slett alle printf for debugging 

var t float64  // TODO: slett 

const bcast = "129.241.187.255"
const udpRcvPort = "2800"
const udpSendPort = "3000"

const SEND_INTERVAL = 10*time.Millisecond 
const TIMEOUT_INTERVAL = 500*time.Millisecond 
const SEQUENCE_NUMBER_RANGE = 100

type Packet_t struct { 
	SequenceNumber int
	Participants []string   
	Datagram queueManager.UpdatePacket_t 
	ActiveSender string
	NextSender string
}

func new_packet(myID string, nFloors int, initDatagram queueManager.UpdatePacket_t) (Packet_t) {
	packet := Packet_t{}
	packet.SequenceNumber = -1
	packet.Participants = append(packet.Participants, myID)
	packet.Datagram = initDatagram
	packet.NextSender = myID
	packet.ActiveSender = myID
	return packet
}

func (packet *Packet_t) add_participants(newPacket Packet_t ) {	
	for _, newAdress := range newPacket.Participants{
		exists := false
		for _, oldAdress := range packet.Participants{
			if newAdress == oldAdress{
				exists = true
			}
		}
		if !exists {
			packet.Participants = append(packet.Participants, newAdress)
			packet.Datagram.Elevators[newAdress] = newPacket.Datagram.Elevators[newAdress]
		}
	}
	if len(packet.Participants) == 0 {
		println("participants = 0 caused by 'add_participants'")
	}
}

func (packet *Packet_t ) remove_participant(participantToRemove string) { 
	temp := (*packet).Participants
	sort.Strings(temp)
	indexToRemove := sort.SearchStrings(temp,participantToRemove)
	temp = append(temp[:indexToRemove],temp[indexToRemove+1:]...)
	(*packet).Participants = temp

	delete(packet.Datagram.Elevators, participantToRemove)
	if len(packet.Participants) == 0 {
		println("participants = 0 caused by 'remove_participant'")
	}
}

func (packet *Packet_t) calculate_next_sender(lastSender string) string {
	sort.Strings(packet.Participants)
	nextSenderIndex := sort.SearchStrings(packet.Participants, lastSender) + 1
	packet.NextSender =  packet.Participants[nextSenderIndex % len(packet.Participants)] )
	
	// if len(packet.Participants) == 0 {
	// 	println("Length of participants: 0 !!!!!!!!!!!")
	// }

	// if nextSenderIndex >= len(packet.Participants) {
	// 	temp := packet.Participants[0]
	// 	return temp
	// }else {
	// 	temp := packet.Participants[nextSenderIndex]
	// 	return temp
	// }
}

func (packet *Packet_t ) participant_exists(participant string) bool { 
	for _, currentAdress := range packet.Participants{
		if currentAdress == participant {
			return true
		}
	}
	return false
}

func (packet *Packet_t ) update(receivedPacket Packet_t, queueToNetwork_c chan queueManager.UpdatePacket_t, networkToQueue_c chan queueManager.UpdatePacket_t) {
	packet.Participants = receivedPacket.Participants
	packet.SequenceNumber = receivedPacket.SequenceNumber
	if len(receivedPacket.Participants) == 0 {
		receivedPacket.print()
		log.Fatal("Participants list is empty")
	}

	networkToQueue_c <- receivedPacket.Datagram
	packet.Datagram = <- queueToNetwork_c
	if len(packet.Participants) == 0 {
		println("participants = 0 caused by 'update'")
	}
}

func (packet *Packet_t) newer_sequence_number(receivedSequenceNumber int ) bool {
	if ( receivedSequenceNumber - packet.SequenceNumber ) > ( SEQUENCE_NUMBER_RANGE / 2 )  {
		return  false
	}else if ( packet.SequenceNumber - receivedSequenceNumber ) > ( SEQUENCE_NUMBER_RANGE / 2 ) {
		return true
	}else{
		return receivedSequenceNumber >= packet.SequenceNumber 
	}
}

//TODO: Slett
func (packet *Packet_t) print() {
	println("next sender:",packet.NextSender)
	println("Participants:")
	for i, adress := range packet.Participants{
		print("\tAdress #",i," : ",adress,"\n")
	}
	print("Sequence #",packet.SequenceNumber, "\n")
}

func Run(myID string, nFloors int, queueToNetwork_c chan queueManager.UpdatePacket_t, networkToQueue_c chan queueManager.UpdatePacket_t) {  //TODO: Dust navn; bør endres
	initDatagram := <-queueToNetwork_c 
	myPacket := new_packet(myID, nFloors,initDatagram) 
	transmitWindow_timer := time.NewTimer( TIMEOUT_INTERVAL ) 
	retransmit_timer := time.NewTimer( SEND_INTERVAL ) 
	iAmAlone_timer := time.NewTimer( TIMEOUT_INTERVAL * 5) 
	iAmActiveSender := true

	send_c := make(chan Packet_t,1) 		
	receive_c := make(chan Packet_t,1)	
	
	go receive_message( receive_c )	
	go send_message( send_c )		

	fails := 0	// TODO: slett


	for {
		myPacket.calculate_next_sender( myID)  //TODO: bør endres slik at den kun tar inn ett argument
		
		logger(myPacket,fails) // TODO: SLett

		if iAmActiveSender {	
			
			transmitWindow_timer.Reset(TIMEOUT_INTERVAL)
			retransmit_timer.Reset(SEND_INTERVAL)
			myPacket.SequenceNumber = (myPacket.SequenceNumber + 1) % SEQUENCE_NUMBER_RANGE //TODO: hva skal skjer når den nå SEQENCE_NUMBER_RANGE
			shouldBreak := false
			
			for !shouldBreak {
				select{
					case receivedPacket := <-receive_c:
						if myPacket.participant_exists(receivedPacket.ActiveSender){
							if myPacket.newer_sequence_number(receivedPacket.SequenceNumber) && receivedPacket.participant_exists(myID){
								iAmActiveSender = false
								shouldBreak = true
							}
						}else{
							myPacket.add_participants(receivedPacket)
							shouldBreak = true
						}


					case <- transmitWindow_timer.C:
						if myPacket.NextSender != myID {
							myPacket.remove_participant(myPacket.NextSender)
							fails++ //TODO: slett
							shouldBreak = true
						}
					case <- retransmit_timer.C:
						retransmit_timer.Reset( SEND_INTERVAL )
						send_c <- myPacket
						if len(myPacket.Participants) == 1 {
							networkToQueue_c <- myPacket.Datagram
							myPacket.Datagram = <- queueToNetwork_c
						}
				}

			}
			
		}else {
	
			iAmAlone_timer.Reset( TIMEOUT_INTERVAL * 5)
			shouldBreak := false

			for !shouldBreak {
				select{
					case receivedPacket := <-receive_c:
						if receivedPacket.NextSender == myID && myPacket.newer_sequence_number(receivedPacket.SequenceNumber) { 
							myPacket.update(receivedPacket, queueToNetwork_c, networkToQueue_c)
							iAmActiveSender = true
							shouldBreak = true
						}
					case <- iAmAlone_timer.C: 
						var emptyList []string  
						myPacket.Participants = append(emptyList, myID ) 
						iAmActiveSender = true
						fails++  //TODO: slett dene
						shouldBreak = true
				}
				
			}
		}
	}
}

func receive_message(transmitChannel chan Packet_t) {
	localAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort( "",udpRcvPort))
	if err != nil {
		log.Fatal( "Failed to resolve addr for :" + udpRcvPort, err );
	}

	recieveConnection, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		log.Fatal("UDP recv connection error on " + localAddr.String(), err)
	}
	
	defer recieveConnection.Close()

	for {
		receiveBufferRaw := make( []byte, 1600 ) 
		var receiveBuffer bytes.Buffer
		UDPpacketDecoder := gob.NewDecoder(&receiveBuffer)

		_, from, err := recieveConnection.ReadFromUDP( receiveBufferRaw )
		if from.String() == get_my_IP() + ":" + udpSendPort {
			continue
		}
		if err != nil {
			log.Fatal("Error receiving UDP packet: " + err.Error(),err )
		}
		
		receiveBuffer.Write(receiveBufferRaw)
		receivedPacket := Packet_t{} 
		err = UDPpacketDecoder.Decode(&receivedPacket) 
		if err != nil {
			log.Print("Could not decode message: ", err)
			continue
		}
		transmitChannel <- receivedPacket 
	}
}

func send_message( transmitChannel chan Packet_t ){
	remoteAddr, err := net.ResolveUDPAddr( "udp", net.JoinHostPort( bcast, udpRcvPort ) )
	if err != nil {
		log.Fatal("Failed to resolve UDP remote address:", err)
	}
	localAddr, err := net.ResolveUDPAddr( "udp", net.JoinHostPort("", udpSendPort ) )
	if err != nil {
		log.Fatal("Failed to resolve UDP local address:", err)
	}
	sendConnection, err := net.ListenUDP( "udp", localAddr)
	if err != nil {
		log.Fatal("UDP send connection error on " + localAddr.String() + ": ", err)
	}
	defer sendConnection.Close()

	for {
		newPacket := <- transmitChannel:
	
		var sendBuffer bytes.Buffer
		UDPpacketEncoder := gob.NewEncoder(&sendBuffer)
		sendBufferRaw := make( []byte, 1600)

		err = UDPpacketEncoder.Encode(newPacket)
		if err != nil {
			log.Fatal("Unable to encode new packet. ", err)
		}

		sendBuffer.Read(sendBufferRaw)
		_, err = sendConnection.WriteToUDP(sendBufferRaw, remoteAddr)
		if err != nil {
			log.Fatal("Failed to send packet from buffer: ", err)
		}
	}
}

func get_my_IP() string {
	addrs, err := net.InterfaceAddrs()
    	if err != nil {
        	log.Fatal(err)
         }
        for _, address := range addrs {
       		// check the address type and if it is not a loopback the display it
        	if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
            	if ipnet.IP.To4() != nil {
                	return ipnet.IP.String()
                }

            }
        }
    log.Fatal("Could not resolve IP-adress")
    return "invalidIP" //Just to satisfy golang requirement of consistent return statements
}

func logger(packet Packet_t,numberOfFails int) {

    f, err := os.Create("data")
    if err != nil {
        panic(err)
    }
    defer f.Close()

    E3 := "Sequence number: " + strconv.Itoa(packet.SequenceNumber) +"\n" 
    
    _, err = f.WriteString(E3)
    if err != nil {
        panic(err)
    }

	E5 := "NextSender: " + packet.NextSender +"\n" 
    
    _, err = f.WriteString(E5)
    if err != nil {
        panic(err)
    }

    E1 := "ActiveSender: " + packet.ActiveSender +"\nParticipants: \n" 
    
    _, err = f.WriteString(E1)
    if err != nil {
        panic(err)
    }

    
    for i,adress := range packet.Participants{
    	E2 := "   Participant #" + strconv.Itoa(i) + ": " + adress + "\n"
	    E2 = E2
	    _, err = f.WriteString(E2)
	    if err != nil {
	        panic(err)
	    }
	}

	
	

	E4 := "\n Number of failures: " + strconv.Itoa(numberOfFails) +"\n" 
    
    _, err = f.WriteString(E4)
    if err != nil {
        panic(err)
    }
	
    E6 := "RTT: " + strconv.FormatFloat(t,'e',4,64) +"\n" 
    
    _, err = f.WriteString(E6)
    if err != nil {
        panic(err)
    }

    f.Sync()
}
