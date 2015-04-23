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
)

func main() {
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
       		// check the address type and if it is not a loopback the display it
        	if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
            	if ipnet.IP.To4() != nil {
                	return ipnet.IP.String()
                }

            }
        }
   	log.Fatal("could not resolve ID")
    return "invalidID"
}


