// NEW

package network

import (
"queueManager"

"net"
//"fmt"
"log"
"os"
"strconv"
"bytes"
"encoding/gob"
"sort"
"time"

)

// TODO HOVEDLINJER:
/*
"goroutine, index out of range, participants = 0" feilen er der fortsatt


*/

var t float64
// TODO: endre navn på funksjoner
const bcast = "129.241.187.255"
const udpRcvPort = "2800"
const udpSendPort = "3000"
const SEND_INTERVAL = 10 //milliseconds
const TIMEOUT_INTERVAL = 500 // milliseconds
const SEQUENCE_NUMBER_RANGE = 100

//type queueManager.UpdatePacket_t queueManager.UpdatePacket_t // TODO: nytt navn
//Må være global for at encoding/gob biblioteket skal kunne encode/decode structen
type Packet_t struct {  //TODO: Endre navn til netWorkDatagram
	SequenceNumber int
	Participants []string   
	Orders queueManager.UpdatePacket_t
	ActiveSender string
	NextSender string
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
			packet.Orders.Elevators[newAdress] = newPacket.Orders.Elevators[newAdress]
		}
	}
	if len(packet.Participants) == 0 {
		println("participants = 0 caused by 'add_participants'")
	}
}

func (packet *Packet_t ) remove_participant(adressToRemove string) {
	temp := (*packet).Participants
	sort.Strings(temp)
	indexToRemove := sort.SearchStrings(temp,adressToRemove)
	temp = append(temp[:indexToRemove],temp[indexToRemove+1:]...)
	(*packet).Participants = temp

	delete(packet.Orders.Elevators, adressToRemove)
	if len(packet.Participants) == 0 {
		println("participants = 0 caused by 'remove_participant'")
	}
}

func (packet *Packet_t ) adress_exists_in_list(newAdress string) bool {
	for _, currentAdress := range packet.Participants{
		if currentAdress == newAdress{
			return true
		}
	}
	return false
}

func (packet *Packet_t ) merge_packets(receivedPacket Packet_t, pullQueueChan chan queueManager.UpdatePacket_t, pushQueueChan chan queueManager.UpdatePacket_t) {
	packet.Participants = receivedPacket.Participants
	packet.SequenceNumber = receivedPacket.SequenceNumber
	// println("This is what i received: ")
	// queueManager.PrintOrderQueues(packet.Orders.Elevators)
	//t0 := time.Now()
	pushQueueChan <- receivedPacket.Orders
	packet.Orders = <- pullQueueChan
	//t = time.Since(t0).Seconds()
	if len(packet.Participants) == 0 {
		println("participants = 0 caused by 'merge_packets'")
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

func (packet *Packet_t) print() {
	println("next sender:",packet.NextSender)
	println("Participants:")
	for i, adress := range packet.Participants{
		print("\tAdress #",i," : ",adress,"\n")
	}
	print("Sequence #",packet.SequenceNumber, "\n")
}

func newPacket(nFloors int, initOrders queueManager.UpdatePacket_t) (Packet_t) {
	packet := Packet_t{}
	packet.SequenceNumber = -1
	packet.Participants = append(packet.Participants, getMyIP())
	packet.Orders = initOrders
	packet.NextSender = getMyIP()
	packet.ActiveSender = getMyIP()
	return packet
}

func getMyIP() string {
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
    return "invalidIP"
}

func Global_get_ip() string {  //TODO: slett denne
	return getMyIP()
}

//Hvorfor ikke bare ta inn en hel packet?
func calculate_next_sender(participants []string, lastSender string) string {
	sort.Strings(participants)
	nextSenderIndex := sort.SearchStrings(participants, lastSender) + 1
	// Kan bruke modulo her: return participants[nextSenderIndex % len(participants)] )
	
	if len(participants) == 0 {
		println("Length of participants: 0 !!!!!!!!!!!")
	}

	if nextSenderIndex >= len(participants) {
		temp := participants[0]
		return temp
	}else {
		temp := participants[nextSenderIndex]
		return temp
	}
}

func Run(nFloors int, pullQueueChan chan queueManager.UpdatePacket_t, pushQueueChan chan queueManager.UpdatePacket_t) {  //TODO: Dust navn; bør endres
	initOrders := <-pullQueueChan //TODO: nytt navn
	myPacket := newPacket(nFloors,initOrders)
	myIP := getMyIP()
	timeoutTimer := time.NewTimer(time.Millisecond * TIMEOUT_INTERVAL)
	transmitTimer := time.NewTimer(time.Millisecond * SEND_INTERVAL)
	iAmAloneTimer := time.NewTimer(time.Millisecond * TIMEOUT_INTERVAL * 5)
	iAmActiveSender := true

	sendChan := make(chan Packet_t,1)
	receiveChan := make(chan Packet_t,1)
	quit := make(chan int, 2)

	go receive_message(receiveChan, quit)
	go send_message(sendChan, quit)

	 fails := 0


	 //var sum1 float64 = 0
	 println("\nNetwork Module Initialized\n")


	for {
		myPacket.NextSender = calculate_next_sender(myPacket.Participants, myIP)  //TODO: bør endres slik at den kun tar inn ett argument
		
		logger(myPacket,fails)
		
		// if int(counter) % 50 == 0 { 
		// 	println("Average RTT: ",sum/counter)
		// 	println("Last SRT: ",sum1)
		// }

		if iAmActiveSender {
			//println("I am active")	
			timeoutTimer.Reset(time.Millisecond * TIMEOUT_INTERVAL)
			transmitTimer.Reset(time.Millisecond* SEND_INTERVAL)
			myPacket.SequenceNumber = (myPacket.SequenceNumber + 1) % SEQUENCE_NUMBER_RANGE //TODO: hva skal skjer når den nå SEQENCE_NUMBER_RANGE
			shouldBreak := false
			for !shouldBreak {
				select{
					case receivedPacket := <-receiveChan:
						//println("received packet from: ",receivedPacket.ActiveSender, " sent to: ", receivedPacket.NextSender)
						if myPacket.adress_exists_in_list(receivedPacket.ActiveSender){
							if myPacket.newer_sequence_number(receivedPacket.SequenceNumber) && receivedPacket.adress_exists_in_list(myIP){
								iAmActiveSender = false
								shouldBreak = true
							}
						}else{
							myPacket.add_participants(receivedPacket)
							shouldBreak = true
						}


					case <- timeoutTimer.C:
						if myPacket.NextSender != myIP {
							myPacket.remove_participant(myPacket.NextSender)
							fails++
							println("timeout")
							shouldBreak = true
						}
					case <- transmitTimer.C:
						transmitTimer.Reset(time.Millisecond * SEND_INTERVAL)
						sendChan <- myPacket
						if len(myPacket.Participants) == 1 {
							pushQueueChan <- myPacket.Orders
							myPacket.Orders = <- pullQueueChan
						}
						//println("sending packet to: ", myPacket.NextSender)
				}

			}
			
		}else {
			//println("I am pasive")	
			iAmAloneTimer.Reset(time.Millisecond * TIMEOUT_INTERVAL * 5)
			shouldBreak := false
			for !shouldBreak {
				select{
					case receivedPacket := <-receiveChan:
						//println("received packet from: ",receivedPacket.ActiveSender, " sent to: ", receivedPacket.NextSender)
						//println("received seqence #: ",receivedPacket.SequenceNumber,"local seqence #: ", myPacket.SequenceNumber)
						if receivedPacket.NextSender == myIP && myPacket.newer_sequence_number(receivedPacket.SequenceNumber) { 
							myPacket.merge_packets(receivedPacket, pullQueueChan, pushQueueChan)
							iAmActiveSender = true
							shouldBreak = true
						}
					case <- iAmAloneTimer.C: 
						var emptyList []string   //TODO: litt dust løsning
						myPacket.Participants = append(emptyList, getMyIP()) //Sletter hele listen, mulig bug her
						iAmActiveSender = true
						fails++
						println("I am alone")
						shouldBreak = true
				}
				
			}
		}
	}
	quit <- 1
	quit <- 1 
}

func receive_message(transmitChannel chan Packet_t, quit chan int) {
	//Initializing
	localAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort( "",udpRcvPort))
	if err != nil {
		log.Fatal( "Failed to resolve addr for :" + udpRcvPort, err );
	}

	recieveConnection, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		log.Fatal("UDP recv connection error on " + localAddr.String(), err)
	}
	
	defer recieveConnection.Close()
	//Initialization done


		for {
			select{
				case <-quit:
					return
				default:
					// Decoderen må opprettes ny for hver pakke, fordi UDP benyttes og man
					// kan ikke være sikker på at den første pakken ble sendt (som inneholder type)
					receiveBufferRaw := make( []byte, 1600 ) // standard MTU size -- no packet should be bigger
					var receiveBuffer bytes.Buffer
					UDPpacketDecoder := gob.NewDecoder(&receiveBuffer)

					_, from, err := recieveConnection.ReadFromUDP( receiveBufferRaw )
					if from.String() == getMyIP() + ":" + udpSendPort {
						continue
					}
					if err != nil {
						log.Fatal("Error receiving UDP packet: " + err.Error(),err )
					}
					
					receiveBuffer.Write(receiveBufferRaw)
					mssg := Packet_t{} // Mulig feil her og neste linje
					err = UDPpacketDecoder.Decode(&mssg) 
					if err != nil {
						log.Print("Could not decode message: ", err)
						continue
					}
					transmitChannel <- mssg 
			}
		}
}

func send_message(transmitChannel chan Packet_t, quit chan int){
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
		select{
			case <-quit:
				return
			case newPacket := <- transmitChannel:
				
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
