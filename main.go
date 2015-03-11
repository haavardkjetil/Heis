package main

import (
"queueManager"
"driver"
"stateMachine"
"time"
"network"
)

func main(){
	localIP := network.Global_get_ip()
	numFloors := 4
	networkReceive := make(chan queueManager.UpdatePacket_t)
	networkTransmit := make(chan queueManager.UpdatePacket_t)

	go network.Run(4,networkTransmit,networkReceive)

	orderChan := make(chan queueManager.Order_t)
	positionChan := make(chan int)
	globalStatusChan := make(chan queueManager.ElevatorStatus_t)
	commandChan := make(chan queueManager.ElevatorCommand_t)
	quitChan := make(chan int)

	go queueManager.RunQueueManager(localIP, numFloors, networkReceive, networkTransmit, orderChan, globalStatusChan, commandChan, positionChan, quitChan)

	//driver
	buttonLampChan := make(chan driver.ButtonLampUpdate_t,10)
	buttonSensorChan := make(chan driver.Button_t,10)
	floorSensorChan := make(chan int,10)
	floorIndicatorChan := make(chan int,10)
	motorDirChan := make(chan driver.MotorDirection_t,10)
	doorLampChan := make(chan bool,10)

	go stateMachine.RunStateMachine(numFloors, floorSensorChan, positionChan, buttonSensorChan, orderChan, globalStatusChan, commandChan, motorDirChan, doorLampChan)

	go driver.Run(buttonLampChan, buttonSensorChan, floorSensorChan, floorIndicatorChan, motorDirChan, doorLampChan)
	for{
		time.Sleep(time.Second * 10)
	}
}	