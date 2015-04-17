package queueManager

import (
.	"math"
"errors"
.	"fmt"
"sort"
"driver"
"log"
)



type ButtonCall_t int
const(
	BUTTON_CALL_UP 		ButtonCall_t = iota
	BUTTON_CALL_DOWN
	BUTTON_CALL_INSIDE
)

type  OrderCommand_t int
const(
	DELETE	OrderCommand_t = -1
	ADD 	OrderCommand_t = 1
)

type ElevatorStatus_t int
const(
	MOVING_UP 	ElevatorStatus_t = iota
	MOVING_DOWN
	IDLE
	UNKNOWN
	DOOR_OPEN
)

type ElevatorCommand_t int
const(
	GO_UP 	ElevatorCommand_t = iota
	GO_DOWN
	OPEN_DOORS
	GO_IDLE
)

type Elevator_t struct{
	Status ElevatorStatus_t
	Position int
	NumPositions int
	NumFloors int
	Orders[][]bool
}

//Datagram_t ??
type UpdatePacket_t struct{
	Elevators map[string]Elevator_t
	GlobalOrders[][]bool
}

type Order_t struct{
	Operation OrderCommand_t
	Button driver.Button_t  			// Æsj..:!
}


func Run(localIP string, 
					numFloors int, 
					networkReceive, networkTransmit chan UpdatePacket_t, 
					statusChan chan ElevatorStatus_t, 
	      			buttonSensorChan_pull chan driver.Button_t,
	      			buttonLampChan_push chan driver.ButtonLampUpdate_t,
	      			deleteOrder_pull chan int,
					destinationChan_push chan int, 
					positionChan chan int){

	numPositions := numFloors*2-1
	currentPosition := <- positionChan
	currentStatus := UNKNOWN

	var globalOrders = make( [][]bool, numFloors)
	for floor := 0; floor < numFloors; floor++ {
		globalOrders[floor] = make([]bool, 2)
	}
	
	globalElevators := make( map[string]Elevator_t )
	globalElevators[localIP] = make_elevator(numFloors, currentPosition, currentStatus)
	networkTransmit <- UpdatePacket_t{ globalElevators, globalOrders}
	Println("Elevator queue system successfully initialized.")

	numElevators := len(globalElevators)
	var newOrders []Order_t 
	for{
		select{
		case newStatus := <- statusChan:
			if newStatus != DOOR_OPEN{
				currentStatus = newStatus
			}

		case floorServed := <- deleteOrder_pull:
			var newOrder Order_t
			newOrder.Operation = DELETE
			newOrder.Button.Floor = floorServed
			newOrders = append(newOrders, newOrder)

		case currentPosition = <- positionChan:

		case newButtonCall := <- buttonSensorChan_pull:
			var newOrder Order_t
			newOrder.Operation = ADD
			newOrder.Button = newButtonCall
			newOrders = append(newOrders, newOrder)

		case networkUpdate := <- networkReceive:
			shouldRedistribute := false
			shouldPrint := false
			if len(networkUpdate.Elevators) != numElevators{
				shouldRedistribute = true
				merge_bool_matrix(globalOrders, networkUpdate.GlobalOrders)
				numElevators = len(networkUpdate.Elevators)
			}else{
				copy_bool_matrix(globalOrders, networkUpdate.GlobalOrders)
			}

			localElevator := make_elevator(numFloors, currentPosition, currentStatus)
			if currentPosition != networkUpdate.Elevators[localIP].Position{
				shouldPrint = true
				Println("currentPosition = ", currentPosition)
			}
			copy_bool_matrix(localElevator.Orders, networkUpdate.Elevators[localIP].Orders)

			for _, newOrder := range newOrders{
				if newOrder.Operation == ADD{
					shouldRedistribute = true
					add_order(&localElevator, newOrder.Button, globalOrders)
				}else{
					delete_order(&localElevator, newOrder.Button.Floor, globalOrders)
				}
			}
			update_lights(numFloors, globalOrders, localElevator.Orders, buttonLampChan_push)
			newOrders = nil
			networkUpdate.Elevators[localIP] = localElevator
			if shouldRedistribute{
				redistribute_orders(networkUpdate.Elevators, globalOrders)
			}

			copy_bool_matrix(networkUpdate.GlobalOrders, globalOrders)
			networkTransmit <- networkUpdate

			nextDestination := get_next_destination(networkUpdate.Elevators[localIP], numPositions)
			destinationChan_push <- nextDestination
			if shouldPrint || true {
				print_queues(networkUpdate.Elevators)

			}
		}
	}
}

func make_elevator(numFloors, position int, status ElevatorStatus_t) Elevator_t{
	e := Elevator_t{status, position, numFloors*2-1, numFloors, make([][]bool, numFloors)}
	for floor := 0; floor < numFloors; floor++ {
		e.Orders[floor] = make([]bool, 3)
	}
	return e
}

func update_lights(numFloors int, globalOrders, localOrders [][]bool, buttonLampChan_push chan driver.ButtonLampUpdate_t){
	for floor := 0; floor < numFloors; floor++{
		button := driver.Button_t{
			driver.BUTTON_CALL_UP, floor,
		}
		if floor != numFloors-1{
			buttonLampChan_push <- driver.ButtonLampUpdate_t{
				button, globalOrders[floor][BUTTON_CALL_UP],
			} 
		}
		if floor != 0{		
			button.Type = driver.BUTTON_CALL_DOWN
			buttonLampChan_push <- driver.ButtonLampUpdate_t{
				button, globalOrders[floor][BUTTON_CALL_DOWN],
			}
		}
		button.Type = driver.BUTTON_CALL_INSIDE
		buttonLampChan_push <- driver.ButtonLampUpdate_t{
			button, localOrders[floor][BUTTON_CALL_INSIDE],
		} 
	}
}

func find_optimal_elevator(elevators map[string]Elevator_t, buttonCall ButtonCall_t, orderedFloor int) string {  
	bestTime := Inf(1)
	sortedIPs := make([]string, 0, len(elevators))
	for elevatorIP := range elevators{
		sortedIPs = append(sortedIPs, elevatorIP)
	}
	sort.Strings(sortedIPs)
	bestElevator := sortedIPs[0]
	for _, elevatorIP := range sortedIPs {
		elevator := elevators[elevatorIP]
		elevStatus := elevator.Status
		if elevStatus == UNKNOWN {continue}
		tempOrders := make( [][]bool, elevator.NumFloors )
		copy_bool_matrix(tempOrders, elevator.Orders) 
		tempOrders[orderedFloor][buttonCall] = true
		newTravelTime, _ := calculate_cost(elevator.Position, elevator.Status, tempOrders)
		if newTravelTime < bestTime {
			bestTime = newTravelTime
			bestElevator = elevatorIP
		}else if newTravelTime == bestTime {
			if elevatorIP < bestElevator{
				bestElevator = elevatorIP
				bestTime = newTravelTime
			}
		}
	}
	return bestElevator
}

func calculate_cost(initialPosition int, initialStatus ElevatorStatus_t, orders [][]bool) (float64, error) {
	if initialPosition < 0 || initialPosition >= len(orders)*2-1{
		return Inf(1), errors.New("Call to calculate_cost(): Initial position outside range.")
	}
	if initialStatus == UNKNOWN{
		return Inf(1), nil
	}
	numFloors := len(orders)
	numPositions := numFloors*2-1
	tempOrders := make( [][]bool, numFloors )
	copy_bool_matrix(tempOrders, orders) 
	var driveTime float64 = 0
	var waitTime float64 = 0
	var totalTime float64 = 0
	var floorToFloorTime float64 = 2.2
	var doorOpenTime float64 = 3
	if (initialStatus == MOVING_UP || initialStatus == IDLE) {
		var distanceTravelledUp float64 = 0
		for position := initialPosition; position < numPositions; position++{
			if position % 2 == 0{
				floor := position/2
				if tempOrders[floor][BUTTON_CALL_UP] || tempOrders[floor][BUTTON_CALL_INSIDE] || ( floor == numFloors-1 && tempOrders[floor][BUTTON_CALL_DOWN]){
					driveTime += distanceTravelledUp*floorToFloorTime
					waitTime += doorOpenTime
					tempOrders[floor][BUTTON_CALL_UP] = false
					tempOrders[floor][BUTTON_CALL_DOWN] = false
					tempOrders[floor][BUTTON_CALL_INSIDE] = false
					distanceTravelledUp = 0
				}else if position == (initialPosition + 1){
					driveTime += distanceTravelledUp*floorToFloorTime
					distanceTravelledUp = 0
				}
				distanceTravelledUp += 0.1
			}else{
				distanceTravelledUp += 0.9
			}
		}
		distanceTravelledUp -= 0.1
		var distanceTravelledDown float64 = 0
		for position := numPositions-1; position >= 0; position--{
			if position % 2 == 0{
				floor := position/2
				if tempOrders[floor][BUTTON_CALL_DOWN] || tempOrders[floor][BUTTON_CALL_INSIDE] {
					driveTime += Abs(distanceTravelledDown - distanceTravelledUp)*floorToFloorTime
					waitTime += doorOpenTime
					tempOrders[floor][BUTTON_CALL_UP] = false
					tempOrders[floor][BUTTON_CALL_DOWN] = false
					tempOrders[floor][BUTTON_CALL_INSIDE] = false
					distanceTravelledDown = 0
					distanceTravelledUp = 0
				}
				distanceTravelledDown += 0.1
			}else{
				distanceTravelledDown += 0.9
			}
		}
		distanceTravelledDown -= 0.1
		for position := 0; position <= initialPosition; position++{
			if position % 2 == 0{
				floor := position/2
				if tempOrders[floor][BUTTON_CALL_UP] || tempOrders[floor][BUTTON_CALL_INSIDE] {
					driveTime += Abs(distanceTravelledUp - distanceTravelledDown)*floorToFloorTime
					waitTime += doorOpenTime
					tempOrders[floor][BUTTON_CALL_UP] = false
					tempOrders[floor][BUTTON_CALL_DOWN] = false
					tempOrders[floor][BUTTON_CALL_INSIDE] = false
					distanceTravelledDown = 0
					distanceTravelledUp = 0
				}
				distanceTravelledUp += 0.1
			}else{
				distanceTravelledUp += 0.9
			}
		}
		totalTime = driveTime + waitTime	
	}
	driveTime = 0
	waitTime = 0
	copy_bool_matrix(tempOrders, orders)
	if (initialStatus == MOVING_DOWN || initialStatus == IDLE) {
		var distanceTravelledDown float64 = 0
		for position := initialPosition; position >= 0; position--{
			if position % 2 == 0{
				floor := position/2
				if tempOrders[floor][BUTTON_CALL_DOWN] || tempOrders[floor][BUTTON_CALL_INSIDE] || ( (floor == 0) && tempOrders[floor][BUTTON_CALL_UP]){
					driveTime += distanceTravelledDown*floorToFloorTime
					waitTime += doorOpenTime
					tempOrders[floor][BUTTON_CALL_UP] = false
					tempOrders[floor][BUTTON_CALL_DOWN] = false
					tempOrders[floor][BUTTON_CALL_INSIDE] = false
					distanceTravelledDown = 0
				}else if position == (initialPosition - 1){
					driveTime += distanceTravelledDown*floorToFloorTime
					distanceTravelledDown = 0
				}
				distanceTravelledDown += 0.1
			}else{
				distanceTravelledDown += 0.9
			}
		}
		distanceTravelledDown -= 0.1
		var distanceTravelledUp float64 = 0
		for position := 0; position < numPositions; position++{
			if position % 2 == 0{
				floor := position/2
				if tempOrders[floor][BUTTON_CALL_UP] || tempOrders[floor][BUTTON_CALL_INSIDE] {
					driveTime += Abs(distanceTravelledUp - distanceTravelledDown)*floorToFloorTime
					waitTime += doorOpenTime
					tempOrders[floor][BUTTON_CALL_UP] = false
					tempOrders[floor][BUTTON_CALL_DOWN] = false
					tempOrders[floor][BUTTON_CALL_INSIDE] = false
					distanceTravelledDown = 0
					distanceTravelledUp = 0
				}
				distanceTravelledUp += 0.1
			}else{
				distanceTravelledUp += 0.9
			}
		}
		distanceTravelledUp -= 0.1
		for position := numPositions-1; position >= initialPosition; position--{
			if position % 2 == 0{
				floor := position/2
				if tempOrders[floor][BUTTON_CALL_DOWN] || tempOrders[floor][BUTTON_CALL_INSIDE] {
					driveTime += Abs(distanceTravelledDown - distanceTravelledUp)*floorToFloorTime
					waitTime += doorOpenTime
					tempOrders[floor][BUTTON_CALL_UP] = false
					tempOrders[floor][BUTTON_CALL_DOWN] = false
					tempOrders[floor][BUTTON_CALL_INSIDE] = false
					distanceTravelledDown = 0
					distanceTravelledUp = 0
				}
				distanceTravelledDown += 0.1
			}else{
				distanceTravelledDown += 0.9
			}
		}
		if initialStatus == IDLE {
			// dersom de er like returneres den siste (tiden nedover).
			totalTime = Min( totalTime, driveTime + waitTime )
		}else{
			totalTime = driveTime + waitTime
		}
	}
	return totalTime, nil
}

func redistribute_orders(elevators map[string]Elevator_t, sharedOrders [][]bool){
	for floor := range sharedOrders{
		for _, elevator := range elevators{
			elevator.Orders[floor][BUTTON_CALL_UP] = false
			elevator.Orders[floor][BUTTON_CALL_DOWN] = false
		}
	}
	for floor := range sharedOrders{
		if sharedOrders[floor][BUTTON_CALL_UP] {
			bestElevator := find_optimal_elevator(elevators, BUTTON_CALL_UP, floor)
			Println("Best elevator for ", floor, ": ", bestElevator)
			elevators[bestElevator].Orders[floor][BUTTON_CALL_UP] = true
		}
		if sharedOrders[floor][BUTTON_CALL_DOWN] {
			bestElevator := find_optimal_elevator(elevators, BUTTON_CALL_DOWN, floor)
			Println("Best elevator for ", floor, ": ", bestElevator)
			elevators[bestElevator].Orders[floor][BUTTON_CALL_DOWN] = true
		}
	}
}

func get_next_destination(elevator Elevator_t, numPositions int) int {
	initialPosition := elevator.Position
	initialStatus := elevator.Status
	orders := elevator.Orders
	destinationUp := -1
	destinationDown := -1
	upTime := Inf(1)
	downTime := Inf(1)
	shouldDoSomething := false
	if (initialStatus == MOVING_UP || initialStatus == IDLE) {
		for position := initialPosition; position < numPositions; position++{
			if position % 2 == 0{
				floor := position/2
				if orders[floor][BUTTON_CALL_UP] || orders[floor][BUTTON_CALL_INSIDE] {
					destinationUp = position
					upTime, _ = calculate_cost(initialPosition, MOVING_UP, elevator.Orders)
					shouldDoSomething = true
					break
				}
			}
		}
		if (destinationUp == -1) {
			for position := numPositions-1; position >= initialPosition; position--{
				if position % 2 == 0{
					floor := position/2
					if orders[floor][BUTTON_CALL_DOWN]{
						destinationUp = position
						upTime, _ = calculate_cost(initialPosition, MOVING_UP, elevator.Orders)
						shouldDoSomething = true
						break
					}
				}
			}
		}
	}
	if (initialStatus == MOVING_DOWN || initialStatus == IDLE) {
		for position := initialPosition; position >= 0; position--{
			if position % 2 == 0{
				floor := position/2
				if orders[floor][BUTTON_CALL_DOWN] || orders[floor][BUTTON_CALL_INSIDE]{
					if initialStatus == IDLE {
						downTime, _ = calculate_cost(initialPosition, MOVING_DOWN, elevator.Orders)
						if downTime <= upTime{
							destinationDown = position
						}
					}else{
						destinationDown = position
					}
					shouldDoSomething = true
					break
				}
			}
		}
		if destinationDown == -1 {
			for position := 0; position <= initialPosition; position++{
				if position % 2 == 0{
					floor := position/2
					if orders[floor][BUTTON_CALL_UP]{
						if initialStatus == IDLE {
							downTime, _ = calculate_cost(initialPosition, MOVING_DOWN, elevator.Orders)
							if downTime <= upTime{
								destinationDown = position
							}
						}else{
							destinationDown = position
						}
						shouldDoSomething = true
						break
					}
				}
			}
		}
	}
	if (shouldDoSomething && upTime < downTime) {return destinationUp}
	return destinationDown
}

func merge_bool_matrix(dst, src [][]bool) bool {
	if dst == nil || src == nil || len(dst) != len(src){
		log.Fatal("len(dst) != len(src)")
	}
	for i := range src{
		if len(dst[i]) != len(src[i]){
			log.Fatal("len(dst[i]) != len(src[i])")
		}
		for j := range src[i]{
			if src[i][j] || dst[i][j]{ 
				dst[i][j] = true
			}
		}
	}
	return true
}

func copy_bool_matrix(dst, src [][]bool) bool {
	if len(dst) < len(src){
		return false
	}
	for i := range src{
		tempList := make([]bool, len(src[i]))
		copy(tempList, src[i])
		dst[i] = make([]bool, len(src[i]))
		copy(dst[i], tempList)
		// dst[i] = make([]bool, len(src[i]))
		// copy(dst[i], src[i])
	}
	return true
}

func delete_order(source *Elevator_t, floor int, globalOrders [][]bool) error{
	if floor < 0 || floor >= source.NumFloors || floor >= len(globalOrders){
		return errors.New("Call to add_order(): floor does not exist.")
	}
	globalOrders[floor][BUTTON_CALL_DOWN] = false
	globalOrders[floor][BUTTON_CALL_UP] = false
	source.Orders[floor][BUTTON_CALL_INSIDE] = false
	source.Orders[floor][BUTTON_CALL_UP] = false
	source.Orders[floor][BUTTON_CALL_DOWN] = false
	return nil
}

func add_order(source *Elevator_t, activeButton driver.Button_t, globalOrders [][]bool) error{
	floor := activeButton.Floor
	if floor < 0 || floor >= source.NumFloors || floor >= len(globalOrders){
		return errors.New("Call to add_order(): floor does not exist.")
	}

	if activeButton.Type == driver.BUTTON_CALL_INSIDE{
		source.Orders[floor][BUTTON_CALL_INSIDE] = true
	}else if activeButton.Type == driver.BUTTON_CALL_UP{
		globalOrders[floor][BUTTON_CALL_UP] = true
	}else if activeButton.Type == driver.BUTTON_CALL_DOWN{
		globalOrders[floor][BUTTON_CALL_DOWN] = true
	}else{
		log.Fatal("Ordretypen finnes ikke!")
	}
	return nil
}

func print_queues(elevators map[string]Elevator_t){

	elevatorList := make([]string, 0, len(elevators))
	for elevatorIP := range elevators{
		elevatorList = append(elevatorList, elevatorIP)
	}
	if len(elevators) == 0{
		return
	}
	sort.Strings(elevatorList)
	Println("")
	for i := 0; i < len(elevators); i++ {
		Print("***********************************")
	}
	Println("")
	for i := range elevatorList{
		Print("Elevator: ", elevatorList[i], "		")
	}
	Println("")
	for i := range elevatorList{
		var status string
		if elevators[elevatorList[i]].Status == UNKNOWN {status = "UNKNOWN"}
		if elevators[elevatorList[i]].Status == MOVING_UP {status = "MOVING UP			"}
		if elevators[elevatorList[i]].Status == MOVING_DOWN {status = "MOVING DOWN			"}
		if elevators[elevatorList[i]].Status == IDLE {status = "IDLE				"}
		Print("Status: ", status)
	}
	Println("")
	for i := range elevatorList{
		time, _ := calculate_cost(elevators[elevatorList[i]].Position, elevators[elevatorList[i]].Status, elevators[elevatorList[i]].Orders) 
		Printf("Workload: %.1f", time)
		Print(" [s]", "			")
	}
	Println("")
	for i := 0; i < len(elevators); i++ {
		Print("Up  	Down 	Inside	Pos		")
	}
	Println("")
	for position := elevators[elevatorList[0]].NumPositions-1; position >= 0; position--{
		for i := range elevatorList{
			if position %2 == 0 && elevators[elevatorList[i]].Orders[position/2][BUTTON_CALL_UP] {
				Print("X	")
			}else if position % 2 == 0 { 
				Print(".	")
			}else{
				Print("	")
			}
			if position %2 == 0 && elevators[elevatorList[i]].Orders[position/2][BUTTON_CALL_DOWN] {
				Print("X	")
			}else if position % 2 == 0 { 
				Print(".	")
			}else{
				Print("	")
			}
			if position %2 == 0 && elevators[elevatorList[i]].Orders[position/2][BUTTON_CALL_INSIDE] {
				Print("X	")
			}else if position % 2 == 0 { 
				Print(".	")
			}else{
				Print("	")
			}
			if elevators[elevatorList[i]].Position == position {
				Print("O		")
			}else { Print("		")}
			//Print(elevators[elevatorIP].Orders[floor][BUTTON_CALL_UP], "	", elevators[elevatorIP].Orders[floor][BUTTON_CALL_DOWN], "	", elevators[elevatorIP].Orders[floor][BUTTON_CALL_INSIDE], "		")
		}
		Println("")
	}
	for i := 0; i < len(elevators); i++ {
		Print("***********************************")
	}
	Println("\n")
}



// //Trenger ikke disse:
// func order_was_added(newQueue, oldQueue [][]bool) bool {
// 	for i := range newQueue{
// 		for j := range newQueue[i]{
// 			if newQueue[i][j] && !oldQueue[i][j]{ 
// 				return true
// 			}
// 		}
// 	}
// 	return false
// }

// func compare_bool_matrix(a,b [][]bool) bool {
// 	if len(a) != len(b){
// 		return false
// 	}
// 	if a == nil && b == nil{
// 		return true
// 	}
// 	for i := range a{
// 		if len(a[i]) != len(b[i]){
// 			return false
// 		}
// 		for j := range a[i]{
// 			if a[i][j] != b[i][j]{ 
// 				return false
// 			}
// 		}
// 	}
// 	return true
// }

// //Trenger ikke denne
// func copy_map(dst, src map[string]Elevator_t){
// 	for key, _ := range dst{
// 		if _, ok := src[key]; !ok{
// 			delete(dst, key)
// 		}
// 	}
// 	if len(src) == 0{
// 		log.Fatal("Len(src) etter = ", len(src))
// 	}
// 	for key, element := range src{
// 		dst[key] = element
// 	}
// }

// //Trenger ikke denne
// func update_elevator_position(elevator *Elevator_t, position int) error{
// 	if position < 0 || position > elevator.NumPositions {
// 		Println("Attempted to update position: position out of range.")
// 		return errors.New("Attempted to update position: position out of range.")
// 	}
// 	elevator.Position = position
// 	return nil
// }