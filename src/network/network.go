package network

import (
"net"
//"fmt"
"log"
//"os"
//"strconv"
"bytes"
"encoding/gob"
"sort"
//"time"
)
// TODO: endre navn på funksjoner
const bcast = "129.241.187.157"
const udpPort = "2878"
const SEND_WINDOW = 100 //milliseconds

type Packet_t struct {  //TODO: Endre navn?
	Participants []string   // String of IP adresses
	ActiveOrders [][3]bool 
	RemoveOrders [][3]bool
}

func newPacket(nFloors int) (Packet_t) {
	packet := Packet_t{}
	packet.Participants = append(packet.Participants, getMyIP())
	packet.ActiveOrders = make([][3]bool,nFloors)
	packet.RemoveOrders = make([][3]bool,nFloors)
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

func calculate_next_sender(Participants []string, lastSender string) string {
	sort.Strings(Participants)
	nextSenderIndex := sort.SearchStrings(Participants, lastSender) + 1
	if nextSenderIndex == len(Participants){
		return Participants[0]
	}else {
		return Participants[nextSenderIndex]
	}
}

func message_organizer(nFloors int) {  //TODO: Dust navn; bør endres
	packet := newPacket(nFloors)
	//timer := time.NewTimer(time.Millisecond * SEND_WINDOW)
	lastSender := packet.Participants[0]
	nextSender := packet.Participants[0]
	packet.Participants = append(packet.Participants, "999.999.999.999")
	packet.Participants = append(packet.Participants, "111.111.111.111")
	
	println("First Sender:")
	println(lastSender)
	
	for {
		nextSender = calculate_next_sender(packet.Participants,lastSender)
		lastSender = nextSender

		// if nextSender == getMyIP() {

		// 	go receive_message() // Lag kanaler som sendes ut
		// 	go send_message()

		// 	for{
		// 		select{
		// 			case <-reciveChan:
		// 				// Update IP liste
		// 			case <- timer.C:
		// 				timer.Reset()
		// 				// Send done til threadene
		// 				break
		// 		}
		// 	}

		// }else {
		// 	// Start thread med receive func
		// 	for{
		// 		select{
		// 			case <-reciveChan:
		// 				// Update packet
		// 				break
		// 			case <- timer.C:
		// 				timer.Reset()
		// 				// fjern gjeldende IP-adresse fra listen
		// 				break
		// 		}
		// 	}
		// }
		// //send informasjonen som er mottatt til QueueManager
	} 
}

/*
PSEUDO-kode for message organisator

Avgjør hvem sin tur det er til å sende:

	Hvis det er dens tur til å sende
		Broadcast ordrebok sammen med ip Liste gjentatte ganger i hele sendevindu
	
		Samtidig lytt etter nye "connect to gjengen" forespørsler
			Hvis det er en ny ip adresse:
				legg til i Ip liste
				Oppdater ordreliste

	hvis ikke:
		lytt etter andre meldinger
		oppdater ordreliste
		
		
		Hvis lyttevindu utløper uten respons:
			fjern gjeldende IP-adresse fra listen

	
	send informasjonen som er mottatt til QueueManager
*/


func Run(nFloors int) {
	//message_organizer(nFloors)
	
	transmitChannel := make(chan Packet_t,5)
	quit := make(chan int)
	//status <- true
	go send_message(transmitChannel,quit)
	for {
		p := newPacket(nFloors)
		p.Participants = append(p.Participants,"222.222.222.222")
		p.Participants = append(p.Participants,"333.333.333.333")
		transmitChannel <-p
		quit <- 0
	}


	/*
	PSEUDO-kode for Run funksjon

	Ha en for select loop

	Lytt etter informasjon fra extern kanal (ordre lagt til og ordre fjernet)
		- send til message organizer
	Lytt etter informasjon fra intern kanal (motatt melding fra nettverk)
		- send til queueManager
	*/
}

func receive_message(transmitChannel chan Packet_t, quit chan int) {
	//Initializing
	localAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort( "",udpPort))
	if err != nil {
		log.Fatal( "Failed to resolve addr for :" + udpPort, err );
	}

	recieveConnection, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		log.Fatal("UDP recv connection error on " + localAddr.String(), err)
	}
	
	defer recieveConnection.Close()

	receiveBufferRaw := make( []byte, 1600 ) // standard MTU size -- no packet should be bigger
	var receiveBuffer bytes.Buffer
	UDPpacketDecoder := gob.NewDecoder(&receiveBuffer)
	//Initialization done


		for {
			select{
				case <-quit:
					break
				default:
					println("Looping")
					receiveBuffer.Reset()
					_, from, err := recieveConnection.ReadFromUDP( receiveBufferRaw )
					if from.String() == recieveConnection.LocalAddr().String() {
						continue
					}
					if err != nil {
						log.Fatal("Error receiving UDP packet: " + err.Error(),err )
					}

					receiveBuffer.Write(receiveBufferRaw)
					var mssg Packet_t
					err = UDPpacketDecoder.Decode(&mssg)
					if err != nil {
						log.Fatal("Could not decode message: ", err)
					}

					transmitChannel <- mssg 
					receiveBuffer.Reset()
			}
		}
}

func send_message(transmitChannel chan Packet_t, quit chan int){
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

	var sendBuffer bytes.Buffer
	UDPpacketEncoder := gob.NewEncoder(&sendBuffer)
	sendBufferRaw := make( []byte, 1600)
	for {
		select{
			case <-quit:
				break
			default:		
				sendBuffer.Reset() 
				newPacket := <- transmitChannel

				err = UDPpacketEncoder.Encode(newPacket)
				if err != nil {
					log.Fatal("Unable to encode new packet. ", err)
				}
				sendBuffer.Read(sendBufferRaw)
				sendBuffer.Reset()

				_, err = sendConnection.WriteToUDP(sendBufferRaw, remoteAddr)
				if err != nil {
					log.Fatal("Failed to send packet from buffer: ", err)
				}
		}
	}
	return
}
