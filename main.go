package main

import(
"driver"
"stateMachine"
"queueManager"
"network"
)

import(
"sync"
"net"
"log"
)

// Navneforslag:
// newtorkToQueueChan
// queueToNetworkChan

func main() {
	var shutdown sync.WaitGroup
	var initialize sync.WaitGroup


	numFloors := driver.N_FLOORS
	networkReceive := make(chan queueManager.UpdatePacket_t)
	networkTransmit := make(chan queueManager.UpdatePacket_t)

	buttonLampChan := make(chan driver.ButtonLampUpdate_t, numFloors*3)
	buttonSensorChan := make(chan driver.Button_t,10)
	floorSensorChan := make(chan int,10)
	motorDirChan := make(chan driver.MotorDirection_t,10)
	doorLampChan := make(chan bool,10)

	destinationChan := make(chan int,10) 
	statusChan := make(chan queueManager.ElevatorStatus_t,10)
	floorServed := make(chan int,10)
	positionChan := make(chan int,10)

	shutdown.Add(4)
	//TODO: inkrementer her
	initialize.Add(3)

	go network.Run( getLocalID(),4,networkTransmit,networkReceive, initialize)

	//TODO: Legg til initialize i queueManager også
	go queueManager.Run( getLocalID(), numFloors, networkReceive, networkTransmit, statusChan, buttonSensorChan, buttonLampChan, floorServed, destinationChan, positionChan)

	go stateMachine.Run(numFloors, destinationChan,floorServed, positionChan, statusChan, floorSensorChan, motorDirChan, doorLampChan, initialize )

	go driver.Run(buttonLampChan, buttonSensorChan, floorSensorChan, motorDirChan, doorLampChan, initialize )
	
	initialize.Wait()
	println("System initialized")

	shutdown.Wait()
	println("System is shutting down")
}


func getLocalID() string {
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
    return "invalidID"
}