package stateMachine

import (
"queueManager"
.	"fmt"
."driver"
"time"
"log"
)
// Tenker at denne rutinen skal motta input og reagere i henhold til tilstanden den er i. Sender deretter filtrert og bearbeidet input og statusinfo til kømanager.
// Deler en statusvariabel med commandFilter.
// func inputFilter(numPositions int, 
// 				floorSensorChan_pull, globalFloorUpdateChan_push, internalFloorUpdateChan_push chan int, 
// 				buttonDetectChan_pull chan Button_t, 
// 				orderChan_push chan queueManager.Order_t, 
// 				statusChan_push, sharedStatusChan_pull, globalStatusChan_push chan queueManager.ElevatorStatus_t){
// 	status = queueManager.UNKNOWN
// 	currentPosition := -1
// 	for{
// 		select{
// 		case newFloor := <- floorSensorChan_pull:
// 			switch status{
// 			case queueManager.UNKNOWN:
// 				currentPosition = newFloor
// 				internalFloorUpdateChan_push <- newFloor
// 				globalFloorUpdateChan_push <- newFloor
// 			case queueManager.IDLE:
// 				if newFloor != currentPosition{
// 					status = queueManager.UNKNOWN
// 					statusChan_push <- status
// 					globalStatusChan_push <- status
// 					internalFloorUpdateChan_push <- newFloor
// 					globalFloorUpdateChan_push <- newFloor
// 					currentPosition = newFloor
// 				}
// 			case queueManager.MOVING_UP:
// 				globalFloorUpdateChan_push <- newFloor
// 				internalFloorUpdateChan_push <- newFloor
// 				currentPosition = newFloor
// 			case queueManager.MOVING_DOWN:
// 				globalFloorUpdateChan_push <- newFloor
// 				internalFloorUpdateChan_push <- newFloor
// 				currentPosition = newFloor
// 			case queueManager.DOOR_OPEN:
// 				if currentPosition != newFloor{
// 					status = queueManager.UNKNOWN
// 					statusChan_push <- status
// 					globalStatusChan_push <- status
// 					internalFloorUpdateChan_push <- newFloor
// 					globalFloorUpdateChan_push <- newFloor
// 					currentPosition = newFloor
// 				}
// 			}
// 		case detectedButton := <- buttonDetectChan_pull:
// 			var buttonCall queueManager.ButtonCall_t
// 			if detectedButton.Type == BUTTON_CALL_UP{ buttonCall = queueManager.BUTTON_CALL_UP}
// 			if detectedButton.Type == BUTTON_CALL_DOWN{ buttonCall = queueManager.BUTTON_CALL_DOWN}
// 			if detectedButton.Type == BUTTON_CALL_INSIDE{ buttonCall = queueManager.BUTTON_CALL_INSIDE}
// 			newOrder := queueManager.Order_t{
// 				queueManager.ADD, buttonCall, detectedButton.Floor,
// 			}
// 			orderChan_push <- newOrder
// 		case newstatus = <- statusChan_push:
// 			status = newStatus
// 			globalStatusChan_push <- newStatus
// 		default:
// 			time.Sleep(time.Millisecond * 1)
// 		}
// 	}
// }


// Denne rutinen er tenkt å konsentrere seg om kommandoer fra kømodulen. Kommandoene blir vurdert i forhold til nåværende tilstand, slik at kun gyldige kommandoer blir utført
// og ny oppdatert tilstand blir beregnet. Deler en statusvariabel med inputFilter
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
					errorDetectionTimer.Stop()
				}
				status = set_direction(destinationPos, currentPosition, numPositions, motorDirChan_push)
				statusChan_push <- status
				errorDetectionTimer.Reset(time.Second*4)
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
			errorDetectionTimer.Reset(time.Second*4)
			switch status {
			case queueManager.MOVING_UP:
				if newFloor !=-1 {
					if currentPosition > newFloor * 2{
						emergencyShutDown(currentPosition, motorDirChan_push, doorLampChan_push, "Elevator moving down while STATUS == MOVING_UP.")
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
					emergencyShutDown(currentPosition, motorDirChan_push, doorLampChan_push, "Elevator moving faster than normal.")
				case destinationPos = <- destinationChan_pull:
				case <- waitForRecalculation.C:
				}
				if (currentPosition == destinationPos){
					motorDirChan_push <- DIR_STOP
					status = queueManager.DOOR_OPEN
					doorLampChan_push <- true
					doorTimer.Reset(doorOpenInterval)
					floorServedChan_push <- currentPosition/2
					errorDetectionTimer.Stop()
				}else{
					status = set_direction(destinationPos, currentPosition, numPositions, motorDirChan_push)
				}
				statusChan_push <- status

			case queueManager.MOVING_DOWN:
				if newFloor != -1{
					if currentPosition < newFloor * 2{
						emergencyShutDown(currentPosition, motorDirChan_push, doorLampChan_push, "Elevator moving up while STATUS == MOVING_DOWN.")
					}
					currentPosition = newFloor * 2
				}else if (currentPosition == 0){
					Println("reinitialize!")
					status, currentPosition = reinitialize(DIR_UP, motorDirChan_push, doorLampChan_push, floorSensorChan_pull, -1, numPositions)
					statusChan_push <- status
				}else{
					currentPosition -= 1
				}
				positionUpdate_push <- currentPosition
				waitForRecalculation := time.NewTimer(time.Millisecond*50)
				select{
				case currentPosition = <- floorSensorChan_pull:
					emergencyShutDown(currentPosition, motorDirChan_push, doorLampChan_push, "Elevator moving faster than normal.")
				case destinationPos = <- destinationChan_pull:
				case <- waitForRecalculation.C:
				}
				Println("curr:",currentPosition,"dest:",destinationPos)
				if (currentPosition == destinationPos){
					Println("Her skal jeg stoppe!")
					motorDirChan_push <- DIR_STOP
					status = queueManager.DOOR_OPEN
					doorLampChan_push <- true
					doorTimer.Reset(doorOpenInterval)
					floorServedChan_push <- currentPosition/2
					errorDetectionTimer.Stop()
				}else{
					Println("Skal ikke stoppe her.")
					status = set_direction(destinationPos, currentPosition, numPositions, motorDirChan_push)
				}
				statusChan_push <- status

			case queueManager.DOOR_OPEN:
				emergencyShutDown(currentPosition, motorDirChan_push, doorLampChan_push, "Elevator left floor with door open.")
			case queueManager.IDLE:
				emergencyShutDown(currentPosition, motorDirChan_push, doorLampChan_push, "Elevator left floor while in IDLE.")
			}
		case <- errorDetectionTimer.C:
			emergencyShutDown(currentPosition, motorDirChan_push, doorLampChan_push, "Elevator moving slower than normal")
			// Kan restarte timer her hvis newFloor == -1.
		// default:
		// 	time.Sleep(time.Millisecond * 1)
		}	
	}
}

func emergencyShutDown(currentPosition int, motorDirChan_push chan MotorDirection_t, doorLampChan_push chan bool, errorMsg string){
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
			emergencyShutDown(currentPos, motorDirChan_push, doorLampChan_push, "Unable to initialize. Something is wrong.")
		}
	}
	motorDirChan_push <- DIR_STOP
	Println("Elevator is now at floor ", currentPos, ".")
	return queueManager.IDLE, currentPos
}

func set_direction(destinationPos, currentPos, numPositions int, motorDirChan_push chan MotorDirection_t) (queueManager.ElevatorStatus_t) {
	if(destinationPos == -1){
		Println("destinationPos -1, go idle")
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
	return queueManager.IDLE
}