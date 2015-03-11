package stateMachine

import (
"queueManager"
.	"fmt"
."driver"
"time"
"log"
)


func RunStateMachine(numFloors int, 
		floorDetectChan_pull, 
		globalFloorUpdateChan_push chan int, 
		orderDetectChan_pull chan Button_t, 
		orderChan_push chan queueManager.Order_t, 
		globalStatusChan_push chan queueManager.ElevatorStatus_t,
		commandChan_pull chan queueManager.ElevatorCommand_t,
		motorDirChan_push chan MotorDirection_t,
		doorLampChan_push chan bool){
	internalFloorUpdateChan_push := make(chan int)
	sharedStatusChan_ic := make(chan queueManager.ElevatorStatus_t)
	sharedStatusChan_ci := make(chan queueManager.ElevatorStatus_t)
	floorChan_ic := make(chan int)
	go inputFilter(numFloors, floorDetectChan_pull, globalFloorUpdateChan_push, internalFloorUpdateChan_push, orderDetectChan_pull, orderChan_push, sharedStatusChan_ic, sharedStatusChan_ci, globalStatusChan_push)
	go commandFilter(numFloors, commandChan_pull, sharedStatusChan_ci, sharedStatusChan_ic, floorChan_ic, motorDirChan_push, doorLampChan_push)
	for{
		time.Sleep(time.Second * 10)
	}
}

// Tenker at denne rutinen skal motta input og reagere i henhold til tilstanden den er i. Sender deretter filtrert og bearbeidet input og statusinfo til kømanager.
// Deler en statusvariabel med commandFilter.
func inputFilter(numFloors int, 
				floorDetectChan_pull, globalFloorUpdateChan_push, internalFloorUpdateChan_push chan int, 
				buttonDetectChan_pull chan Button_t, 
				orderChan_push chan queueManager.Order_t, 
				sharedStatusChan_push, sharedStatusChan_pull, globalStatusChan_push chan queueManager.ElevatorStatus_t){
	status := queueManager.UNKNOWN
	prevFloor := -1
	for{
		select{
		case newFloor := <- floorDetectChan_pull:
			switch status{
			case queueManager.UNKNOWN:
				prevFloor = newFloor
				internalFloorUpdateChan_push <- newFloor
				globalFloorUpdateChan_push <- newFloor
			case queueManager.IDLE:
				if newFloor != prevFloor{
					status = queueManager.UNKNOWN
					sharedStatusChan_push <- status
					globalStatusChan_push <- status
					internalFloorUpdateChan_push <- newFloor
					globalFloorUpdateChan_push <- newFloor
					prevFloor = newFloor
				}
			case queueManager.MOVING_UP:
				globalFloorUpdateChan_push <- newFloor
				internalFloorUpdateChan_push <- newFloor
				prevFloor = newFloor
			case queueManager.MOVING_DOWN:
				globalFloorUpdateChan_push <- newFloor
				internalFloorUpdateChan_push <- newFloor
				prevFloor = newFloor
			case queueManager.DOOR_OPEN:
				if prevFloor != newFloor{
					status = queueManager.UNKNOWN
					sharedStatusChan_push <- status
					globalStatusChan_push <- status
					internalFloorUpdateChan_push <- newFloor
					globalFloorUpdateChan_push <- newFloor
					prevFloor = newFloor
				}
			}
		case detectedButton := <- buttonDetectChan_pull:
			var buttonCall queueManager.ButtonCall_t
			if detectedButton.Type == BUTTON_CALL_UP{ buttonCall = queueManager.BUTTON_CALL_UP}
			if detectedButton.Type == BUTTON_CALL_DOWN{ buttonCall = queueManager.BUTTON_CALL_DOWN}
			if detectedButton.Type == BUTTON_CALL_INSIDE{ buttonCall = queueManager.BUTTON_CALL_INSIDE}
			newOrder := queueManager.Order_t{
				queueManager.ADD, buttonCall, detectedButton.Floor,
			}
			orderChan_push <- newOrder
		case newStatus := <- sharedStatusChan_push:
			status = newStatus
			globalStatusChan_push <- newStatus
		default:
			time.Sleep(time.Millisecond * 1)
		}
	}
}


// Denne rutinen er tenkt å konsentrere seg om kommandoer fra kømodulen. Kommandoene blir vurdert i forhold til nåværende tilstand, slik at kun gyldige kommandoer blir utført
// og ny oppdatert tilstand blir beregnet. Deler en statusvariabel med inputFilter
func commandFilter(numFloors int, 
					commandChan_pull chan queueManager.ElevatorCommand_t, 
					sharedStatusChan_push, sharedStatusChan_pull chan queueManager.ElevatorStatus_t, 
					floorChan_pull chan int, 
					motorDirChan_push chan MotorDirection_t, 
					doorLampChan_push chan bool){
	status := queueManager.UNKNOWN
	prevFloor := -1
	doorOpenInterval := time.Second * 3
	doorTimer := time.NewTimer(doorOpenInterval)
	errorDetectionTimer := time.NewTimer(time.Second*4)
	errorDetectionTimer.Stop()
	for{
		if status == queueManager.UNKNOWN{
			// Initialize!
			Println("Status == queueManager.UNKNOWN! \nReinitializing local elevator...")
			if prevFloor < numFloors && prevFloor >= 0{
				motorDirChan_push <- DIR_STOP
				status = queueManager.IDLE
				sharedStatusChan_push <- status
				//wait
			}else{
				doorLampChan_push <- false
				motorDirChan_push <- DIR_DOWN
				errorDetectionTimer.Reset(time.Second*4)
				for status == queueManager.UNKNOWN {
					select{
					case newFloor := <- floorChan_pull:
						if newFloor < numFloors && newFloor >= 0 {
							motorDirChan_push <- DIR_STOP
							prevFloor = newFloor
							status = queueManager.IDLE
							sharedStatusChan_push <- status
						}
					case <- errorDetectionTimer.C:
						motorDirChan_push <- DIR_STOP
						log.Fatal("Elevator is broken.")
					default:
						time.Sleep(time.Millisecond * 1)
					}
				}
			}
			Println("Elevator is now in a valid state at floor ", prevFloor, ".")
		}

		select{
		case newCommand := <- commandChan_pull:
			switch newCommand{
			case queueManager.GO_UP:
				if prevFloor == numFloors-1{
					status = queueManager.UNKNOWN
					motorDirChan_push <- DIR_STOP
					sharedStatusChan_push <- status
				}else{
					select{
					case newStatus := <- sharedStatusChan_pull:
						status = newStatus
						if status == queueManager.UNKNOWN{ break }
					default:
					}
					motorDirChan_push <- DIR_UP
					status = queueManager.MOVING_UP
					sharedStatusChan_push <- status
				}
			case queueManager.GO_DOWN:
				if prevFloor == 0{
					status = queueManager.UNKNOWN
					sharedStatusChan_push <- status
					motorDirChan_push <- DIR_UP
				}else{
					select{
					case newStatus := <- sharedStatusChan_pull:
						status = newStatus
						if status == queueManager.UNKNOWN{ break }
					default:
					}
					motorDirChan_push <- DIR_DOWN
					status = queueManager.MOVING_DOWN
					sharedStatusChan_push <- status
				}
			case queueManager.OPEN_DOORS:
				select{
				case newFloor := <- floorChan_pull:
					prevFloor = newFloor
				default:
				}
				if prevFloor == -1{
					status = queueManager.UNKNOWN
					sharedStatusChan_push <- status
				}else{
					motorDirChan_push <- DIR_STOP
					status = queueManager.DOOR_OPEN
					sharedStatusChan_push <- status
					doorLampChan_push <- true
					doorTimer.Reset(doorOpenInterval)
				}
			case queueManager.GO_IDLE:
				if prevFloor == -1{
					status = queueManager.UNKNOWN
					sharedStatusChan_push <- status
					//wait...
				}else{
					motorDirChan_push <- DIR_STOP
					status = queueManager.IDLE
					sharedStatusChan_push <- status
				}
			}
		case <- doorTimer.C:
			doorLampChan_push <- false
			if prevFloor == -1{
				status = queueManager.UNKNOWN
			}else{
				status = queueManager.IDLE
			}
			sharedStatusChan_push <- status
		case newStatus := <- sharedStatusChan_pull:
			status = newStatus
		case newFloor :=  <- floorChan_pull:
			if (prevFloor == 0 && newFloor == -1 && status == queueManager.MOVING_DOWN) || (prevFloor == numFloors-1 && newFloor == -1 && status == queueManager.MOVING_UP){
				motorDirChan_push <- DIR_STOP
				status = queueManager.UNKNOWN
				sharedStatusChan_push <- status
			}
			prevFloor = newFloor
			// Kan restarte timer her hvis newFloor == -1.
		default:
			time.Sleep(time.Millisecond * 1)
		}	
	}
}




