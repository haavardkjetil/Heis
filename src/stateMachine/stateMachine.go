package stateMachine

import (
"driver"
)

import(
"time"
"log"
)

type ElevatorStatus_t int
const(
	MOVING_UP 	ElevatorStatus_t = iota
	MOVING_DOWN
	IDLE
	UNKNOWN
	DOOR_OPEN
)

type StatusUpdate_t struct{
	Position int
	Status ElevatorStatus_t
}

func Run(numFloors int, 
		destination_c chan int,
		status_c chan StatusUpdate_t, 
		floorSensor_c chan int, 
		motorDir_c chan driver.MotorDirection_t, 
		doorLamp_c chan bool){

	numPositions := numFloors*2-1
	status := UNKNOWN
	currentPosition := -1
	doorOpenInterval := time.Second * 3
	betweenFloorTime := time.Second * 4
	doorTimer := time.NewTimer(doorOpenInterval)
	doorTimer.Stop()
	errorDetectionTimer := time.NewTimer(betweenFloorTime)
	errorDetectionTimer.Stop()
	destinationPos := 0
	
	status, currentPosition = reinitialize(driver.DIR_DOWN, motorDir_c, doorLamp_c, floorSensor_c, -1, numPositions)
	status_c <- StatusUpdate_t{currentPosition, status}

	for{
		select{
		case destinationPos = <- destination_c:			
			if destinationPos == -1 {
				break
			}
			switch status {
			case DOOR_OPEN:
				if destinationPos == currentPosition{
					doorTimer.Reset(doorOpenInterval)
					status_c <- StatusUpdate_t{currentPosition, status}
				}
			case IDLE:
				if (currentPosition == destinationPos){
					motorDir_c <- driver.DIR_STOP
					status = DOOR_OPEN
					doorLamp_c <- true
					doorTimer.Reset(doorOpenInterval)
					status_c <- StatusUpdate_t{currentPosition, status}
				}else{
					status = set_direction(destinationPos, currentPosition, status, numPositions, motorDir_c)
					status_c <- StatusUpdate_t{currentPosition, status}
				}
				if status == IDLE || status == DOOR_OPEN { 
					errorDetectionTimer.Stop()
				}else if status == MOVING_UP || status == MOVING_DOWN { 
					errorDetectionTimer.Reset(betweenFloorTime)
				}

			}
		case <- doorTimer.C:
			switch status {
			case DOOR_OPEN:
				doorLamp_c <- false
				status = set_direction(destinationPos, currentPosition, status, numPositions, motorDir_c)
				status_c <- StatusUpdate_t{currentPosition, status}
			}

		case newFloor :=  <- floorSensor_c:
			switch status {
			case MOVING_UP:
				if newFloor !=-1 {
					errorDetectionTimer.Reset(betweenFloorTime)
					if currentPosition > newFloor * 2{
						emergency_stop(currentPosition, motorDir_c, doorLamp_c)
					}
					currentPosition = newFloor*2
				}else if (currentPosition == numPositions-1){
					status, currentPosition = reinitialize(driver.DIR_DOWN, motorDir_c, doorLamp_c, floorSensor_c, -1, numPositions)
				}else{
					currentPosition += 1
				}
				status_c <- StatusUpdate_t{currentPosition, status}
				if (currentPosition == destinationPos){
					motorDir_c <- driver.DIR_STOP
					status = DOOR_OPEN
					doorLamp_c <- true
					doorTimer.Reset(doorOpenInterval)
				}else{
					status = set_direction(destinationPos, currentPosition, status, numPositions, motorDir_c)
				}
				if status == IDLE || status == DOOR_OPEN { errorDetectionTimer.Stop()}
				status_c <- StatusUpdate_t{currentPosition, status}

			case MOVING_DOWN:
				if newFloor != -1{
					errorDetectionTimer.Reset(betweenFloorTime)
					if currentPosition < newFloor * 2{
						emergency_stop(currentPosition, motorDir_c, doorLamp_c)					}
					currentPosition = newFloor * 2
				}else if (currentPosition == 0){
					status, currentPosition = reinitialize(driver.DIR_UP, motorDir_c, doorLamp_c, floorSensor_c, -1, numPositions)
				}else{
					currentPosition -= 1
				}
				status_c <- StatusUpdate_t{currentPosition, status}
				if (currentPosition == destinationPos){
					motorDir_c <- driver.DIR_STOP
					status = DOOR_OPEN
					doorLamp_c <- true
					doorTimer.Reset(doorOpenInterval)
				}else{
					status = set_direction(destinationPos, currentPosition, status, numPositions, motorDir_c)
				}
				if status == IDLE || status == DOOR_OPEN { errorDetectionTimer.Stop() }
				status_c <- StatusUpdate_t{currentPosition, status}

			case DOOR_OPEN:
					emergency_stop(-1, motorDir_c, doorLamp_c)
			case IDLE:
					emergency_stop(-1, motorDir_c, doorLamp_c)
			}
		case <- errorDetectionTimer.C:
			emergency_stop(currentPosition, motorDir_c, doorLamp_c)
		}
	}
}

func emergency_stop(currentPosition int, motorDir_c chan driver.MotorDirection_t, doorLamp_c chan bool){
	motorDir_c <- driver.DIR_STOP
	doorLamp_c <- (currentPosition % 2 == 0)
	time.Sleep(time.Millisecond*100)
	log.Fatal("Emergency stop.")
}


func reinitialize(newDir driver.MotorDirection_t, motorDir_c chan driver.MotorDirection_t, doorLamp_c chan bool, floorSensor_c chan int, currentPos, numPositions int) (ElevatorStatus_t, int) {
	if currentPos % 2 != 0 || currentPos < 0 || currentPos >= numPositions{
		doorLamp_c <- false
		motorDir_c <- newDir
		errorDetectionTimer := time.NewTimer(time.Second*4)
		select{
		case newFloor := <- floorSensor_c:
			currentPos = newFloor*2
		case <- errorDetectionTimer.C:
			emergency_stop(currentPos, motorDir_c, doorLamp_c)
			return UNKNOWN, 0
		}
	}
	motorDir_c <- driver.DIR_STOP
	return IDLE, currentPos
}

func set_direction(destinationPos, currentPos int, currentStatus ElevatorStatus_t, numPositions int, motorDir_c chan driver.MotorDirection_t) (ElevatorStatus_t) {
	if currentPos%2 != 0{
		return currentStatus
	}else if(destinationPos == currentPos){
		motorDir_c <- driver.DIR_STOP
		return DOOR_OPEN
	}else if(destinationPos == -1){
		motorDir_c <- driver.DIR_STOP
		return IDLE
	}else if destinationPos < currentPos{	
		motorDir_c <- driver.DIR_DOWN
		return MOVING_DOWN
	}else{
		motorDir_c <- driver.DIR_UP
		return MOVING_UP
	}
}