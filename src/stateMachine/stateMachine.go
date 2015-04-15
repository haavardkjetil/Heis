package stateMachine

import (
"queueManager"
.	"fmt"
."driver"
"time"
"log"
)

func RunStateMachine(numFloors int, 
		destinationChan_pull chan int,
		floorServedChan_push chan int,
		positionUpdate_push chan int, 
		statusChan_push chan queueManager.ElevatorStatus_t, 
		floorSensorChan_pull chan int, 
		motorDirChan_push chan MotorDirection_t, 
		doorLampChan_push chan bool){

	numPositions := numFloors*2-1
	status := queueManager.UNKNOWN
	currentPosition := -1
	doorOpenInterval := time.Second * 3
	doorTimer := time.NewTimer(doorOpenInterval)
	doorTimer.Stop()
	errorDetectionTimer := time.NewTimer(time.Second*4)
	errorDetectionTimer.Stop()
	destinationPos := 0
	

	for{
		if status == queueManager.UNKNOWN{
			status, currentPosition = reinitialize(DIR_DOWN, motorDirChan_push, doorLampChan_push, floorSensorChan_pull, -1, numPositions)
			positionUpdate_push <- currentPosition
			statusChan_push <- status
		}

		select{
		case destinationPos = <- destinationChan_pull:			
			if destinationPos == -1 {
				break
			}

			switch status {
			case queueManager.MOVING_UP:
				// do nothing
			case queueManager.MOVING_DOWN:
				// do nothing
			case queueManager.DOOR_OPEN:
				if destinationPos == currentPosition{
					doorTimer.Reset(doorOpenInterval)
					floorServedChan_push <- currentPosition/2
				}
			case queueManager.UNKNOWN:
				// invalid state.
			case queueManager.IDLE:
				if (currentPosition == destinationPos){
					motorDirChan_push <- DIR_STOP
					status = queueManager.DOOR_OPEN
					doorLampChan_push <- true
					doorTimer.Reset(doorOpenInterval)
					floorServedChan_push <- currentPosition/2
				}
				status = set_direction(destinationPos, currentPosition, numPositions, motorDirChan_push)
				statusChan_push <- status
				if status == queueManager.IDLE || status == queueManager.DOOR_OPEN { 
					errorDetectionTimer.Stop()
				}else if status == queueManager.MOVING_UP || status == queueManager.MOVING_DOWN { 
					errorDetectionTimer.Reset(time.Second*4)
				}

			}
		case <- doorTimer.C:

			switch status {
			case queueManager.MOVING_UP:
				// do nothing
			case queueManager.MOVING_DOWN:
				// do nothing
			case queueManager.DOOR_OPEN:
				doorLampChan_push <- false
				status = set_direction(destinationPos, currentPosition, numPositions, motorDirChan_push)
				statusChan_push <- status

			case queueManager.UNKNOWN:
				// invalid state.
			case queueManager.IDLE:
				// do nothing
			}

		
		case newFloor :=  <- floorSensorChan_pull:

			switch status {
			case queueManager.MOVING_UP:
				if newFloor !=-1 {
					errorDetectionTimer.Reset(time.Second*4)
					if currentPosition > newFloor * 2{
						emergency_shut_down(currentPosition, motorDirChan_push, doorLampChan_push, "Elevator moving down while STATUS == MOVING_UP.")
					}
					currentPosition = newFloor*2


				}else if (currentPosition == numPositions-1){
					status, currentPosition = reinitialize(DIR_DOWN, motorDirChan_push, doorLampChan_push, floorSensorChan_pull, -1, numPositions)
					statusChan_push <- status
				}else{
					currentPosition += 1
				}
				positionUpdate_push <- currentPosition
				waitForRecalculation := time.NewTimer(time.Millisecond*50)
				select{
				case currentPosition = <- floorSensorChan_pull:
					emergency_shut_down(currentPosition, motorDirChan_push, doorLampChan_push, "Elevator moving faster than normal.")
				case destinationPos = <- destinationChan_pull:
				case <- waitForRecalculation.C:
				}
				if (currentPosition == destinationPos){
					motorDirChan_push <- DIR_STOP
					status = queueManager.DOOR_OPEN
					doorLampChan_push <- true
					doorTimer.Reset(doorOpenInterval)
					floorServedChan_push <- currentPosition/2
				}else{
					status = set_direction(destinationPos, currentPosition, numPositions, motorDirChan_push)
				}
				statusChan_push <- status
				if status == queueManager.IDLE || status == queueManager.DOOR_OPEN { errorDetectionTimer.Stop()}

			case queueManager.MOVING_DOWN:
				if newFloor != -1{
					errorDetectionTimer.Reset(time.Second*4)
					if currentPosition < newFloor * 2{
						emergency_shut_down(currentPosition, motorDirChan_push, doorLampChan_push, "Elevator moving up while STATUS == MOVING_DOWN.")
					}
					currentPosition = newFloor * 2
				}else if (currentPosition == 0){
					status, currentPosition = reinitialize(DIR_UP, motorDirChan_push, doorLampChan_push, floorSensorChan_pull, -1, numPositions)
					statusChan_push <- status
				}else{
					currentPosition -= 1
				}
				positionUpdate_push <- currentPosition
				waitForRecalculation := time.NewTimer(time.Millisecond*50)
				select{
				case currentPosition = <- floorSensorChan_pull:
					emergency_shut_down(currentPosition, motorDirChan_push, doorLampChan_push, "Elevator moving faster than normal.")
				case destinationPos = <- destinationChan_pull:
				case <- waitForRecalculation.C:
				}

				if (currentPosition == destinationPos){
					motorDirChan_push <- DIR_STOP
					status = queueManager.DOOR_OPEN
					doorLampChan_push <- true
					doorTimer.Reset(doorOpenInterval)
					floorServedChan_push <- currentPosition/2
				}else{
					status = set_direction(destinationPos, currentPosition, numPositions, motorDirChan_push)
				}
				statusChan_push <- status
				if status == queueManager.IDLE || status == queueManager.DOOR_OPEN { errorDetectionTimer.Stop() }

			case queueManager.DOOR_OPEN:
				emergency_shut_down(currentPosition, motorDirChan_push, doorLampChan_push, "Elevator left floor with door open.")
			case queueManager.IDLE:
				emergency_shut_down(currentPosition, motorDirChan_push, doorLampChan_push, "Elevator left floor while in IDLE.")
			}
		case <- errorDetectionTimer.C:
			emergency_shut_down(currentPosition, motorDirChan_push, doorLampChan_push, "Elevator moving slower than normal")
		}	
	}
}

func emergency_shut_down(currentPosition int, motorDirChan_push chan MotorDirection_t, doorLampChan_push chan bool, errorMsg string){
	motorDirChan_push <- DIR_STOP
	doorLampChan_push <- (currentPosition % 2 == 0)
	time.Sleep(time.Millisecond*100)
	log.Fatal(errorMsg)
}


func reinitialize(newDir MotorDirection_t, motorDirChan_push chan MotorDirection_t, doorLampChan_push chan bool, floorSensorChan_pull chan int, currentPos, numPositions int) (queueManager.ElevatorStatus_t, int) {
	// Initialize!
	Println("Status == queueManager.UNKNOWN! \nReinitializing local elevator...")
	if currentPos % 2 != 0 || currentPos < 0 || currentPos >= numPositions{
		doorLampChan_push <- false
		motorDirChan_push <- newDir
		errorDetectionTimer := time.NewTimer(time.Second*4)
		select{
		case newFloor := <- floorSensorChan_pull:
			currentPos = newFloor*2
		case <- errorDetectionTimer.C:
			emergency_shut_down(currentPos, motorDirChan_push, doorLampChan_push, "Unable to initialize. Something is wrong.")
		}
	}
	motorDirChan_push <- DIR_STOP
	Println("Elevator initialized")
	return queueManager.IDLE, currentPos
}

func set_direction(destinationPos, currentPos, numPositions int, motorDirChan_push chan MotorDirection_t) (queueManager.ElevatorStatus_t) {
	if(destinationPos == -1){
		motorDirChan_push <- DIR_STOP
		return queueManager.IDLE
	}
	if destinationPos < 0 || destinationPos > numPositions{
		log.Fatal("Invalid position in set_direction()","destinationPos:",destinationPos,"currentPos:",currentPos)
	}else if destinationPos < currentPos{	
		motorDirChan_push <- DIR_DOWN
		return queueManager.MOVING_DOWN
	}else if destinationPos > currentPos{
		motorDirChan_push <- DIR_UP
		return queueManager.MOVING_UP
	}
	motorDirChan_push <- DIR_STOP
	return queueManager.DOOR_OPEN
}