package main

import(
"driver"
"stateMachine"
"queueManager"
"network"
)

import(
"net"
"log"
"sync"
"time"
"bytes"
"encoding/gob"
"os"
)

func run_elevator() {
	var shutdown sync.WaitGroup

	numFloors := driver.N_FLOORS

	queueToNetwork_c := make(chan queueManager.Datagram_t)
	networkToQueue_c := make(chan queueManager.Datagram_t)

	buttonLamp_c := make(chan driver.ButtonLampUpdate_t, numFloors*3)
	buttonSensor_c := make(chan driver.Button_t,10)
	floorSensor_c := make(chan int,10)
	motorDir_c := make(chan driver.MotorDirection_t,10)
	doorLamp_c := make(chan bool,10)

	destination_c := make(chan int,10) 
	status_c := make(chan stateMachine.StatusUpdate_t,10)

	shutdown.Add(1)


	go network.Run( getLocalID(),4,networkToQueue_c,queueToNetwork_c)

	go queueManager.Run( getLocalID(), numFloors, queueToNetwork_c, networkToQueue_c, status_c, buttonSensor_c, buttonLamp_c, destination_c)

	go stateMachine.Run(numFloors, destination_c, status_c, floorSensor_c, motorDir_c, doorLamp_c)

	go driver.Run(buttonLamp_c, buttonSensor_c, floorSensor_c, motorDir_c, doorLamp_c)


	shutdown.Wait()
}

func getLocalID() string {
	addrs, err := net.InterfaceAddrs()
    	if err != nil {
        	log.Fatal(err)
         }
        for _, address := range addrs {
        	if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
            	if ipnet.IP.To4() != nil {
                	return ipnet.IP.String()
                }

            }
        }
    return "invalidID"
}


func main() {  
	const LOCAL_HOST = "127.0.0.1"
	const SEND_PORT = "3005"
	const RECEIVE_PORT ="2805"

	listen_timer := time.NewTimer(time.Second)

	var procAttr os.ProcAttr 
   	procAttr.Files = []*os.File{nil, nil, nil}

	send_c := make(chan bool,10)
	receive_c := make(chan bool,10)
	quit := make(chan int,1)

	iAmPrimary := false

	go receive_from_twin_process(receive_c, RECEIVE_PORT, quit)

	println("Backup system running")
	for{
		select{
			case <-receive_c:
				if !iAmPrimary {
					listen_timer.Reset(time.Second)
				}

			case <- listen_timer.C:
					iAmPrimary = true
					listen_timer.Stop()
					go run_elevator()
					time.Sleep(time.Second * 3)
					quit <- 1
					os.StartProcess("run.sh", nil, &procAttr)
					go send_to_twin_process(send_c, RECEIVE_PORT, SEND_PORT, LOCAL_HOST)

			default:
				time.Sleep(time.Millisecond*10)
				if iAmPrimary {
					send_c <- true
				}
		}	
	}
}

func receive_from_twin_process(receive_c chan bool, rcvPort string, quit chan int) {
	localAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort( "",rcvPort))
	if err != nil {
		log.Fatal( "Failed to resolve addr for :" + rcvPort, err );
	}

	recieveConnection, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		log.Fatal("UDP recv connection error on " + localAddr.String(), err)
	}
	
	defer recieveConnection.Close()

	for {
		select{
		case <-quit:
			return
		default:
			receiveBufferRaw := make( []byte, 1600 ) 
			var receiveBuffer bytes.Buffer
			UDPpacketDecoder := gob.NewDecoder(&receiveBuffer)
			recieveConnection.ReadFromUDP( receiveBufferRaw )
			receiveBuffer.Write(receiveBufferRaw)
			var mssg bool 
			UDPpacketDecoder.Decode(&mssg) 
			receive_c <- mssg 
		}
	}
}

func send_to_twin_process(send_c chan bool, rcvPort, sendPort, localHost string){
	remoteAddr, err := net.ResolveUDPAddr( "udp", net.JoinHostPort( localHost, rcvPort ) )
	if err != nil {
		log.Fatal("Failed to resolve UDP remote address:", err)
	}
	localAddr, err := net.ResolveUDPAddr( "udp", net.JoinHostPort("", sendPort ) )
	if err != nil {
		log.Fatal("Failed to resolve UDP local address:", err)
	}
	sendConnection, err := net.ListenUDP( "udp", localAddr)
	if err != nil {
		log.Print("UDP send connection error on " + localAddr.String() + ": ", err)
		return
	}
	defer sendConnection.Close()

	for {
		newPacket := <- send_c
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
