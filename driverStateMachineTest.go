package main

import(
"driver"
"stateMachine"
"queueManager"
"time"
"network"
)

func main() {

	localIP := network.Global_get_ip()
	numFloors := 4
	networkReceive := make(chan queueManager.UpdatePacket_t)
	networkTransmit := make(chan queueManager.UpdatePacket_t)

	buttonLampChan := make(chan driver.ButtonLampUpdate_t,10)
	buttonSensorChan := make(chan driver.Button_t,10)
	floorSensorChan := make(chan int,10)
	motorDirChan := make(chan driver.MotorDirection_t,10)
	doorLampChan := make(chan bool,10)

	destinationChan := make(chan int,10) 
	statusChan := make(chan queueManager.ElevatorStatus_t,10)
	floorServed := make(chan int,10)
	positionChan := make(chan int,10)


	go network.Run(4,networkTransmit,networkReceive)

	go queueManager.RunQueueManager(localIP, numFloors, networkReceive, networkTransmit, statusChan, buttonSensorChan, floorServed, destinationChan, positionChan)

	go stateMachine.RunStateMachine(numFloors, destinationChan,floorServed, positionChan, statusChan, floorSensorChan, motorDirChan, doorLampChan)

	go driver.Run(buttonLampChan, buttonSensorChan, floorSensorChan, motorDirChan, doorLampChan)
	
	for {
		time.Sleep(time.Millisecond)
	}
}
