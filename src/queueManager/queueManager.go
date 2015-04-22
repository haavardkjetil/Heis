package queueManager

import (
"math"
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


func Run(localID string, 
					numFloors int, 
					network_pull, network_push chan UpdatePacket_t, 
					status_pull chan ElevatorStatus_t, 
	      			buttonSensorChan_pull chan driver.Button_t,
	      			buttonLampChan_push chan driver.ButtonLampUpdate_t,
	      			deleteOrder_pull chan int,
					destinationChan_push chan int, 
					position_pull chan int){

	numPositions := numFloors*2-1
	localPosition := <- position_pull
	localStatus := UNKNOWN

	var globalOrders = make( [][]bool, numFloors)
	for floor := 0; floor < numFloors; floor++ {
		globalOrders[floor] = make([]bool, 2)
	}
	
	globalElevators := make( map[string]Elevator_t )
	globalElevators[localID] = make_elevator(numFloors, localPosition, localStatus)
	network_push <- UpdatePacket_t{ globalElevators, globalOrders}

	numElevators := len(globalElevators)
	var queueUpdateBuffer []Order_t 
	for{
		select{
		case newStatus := <- status_pull:
			if newStatus != DOOR_OPEN{
				localStatus = newStatus
			}

		case floorServed := <- deleteOrder_pull:
			var newQueueUpdate Order_t
			newQueueUpdate.Operation = DELETE
			newQueueUpdate.Button.Floor = floorServed
			queueUpdateBuffer = append(queueUpdateBuffer, newQueueUpdate)

		case localPosition = <- position_pull:

		case newButtonCall := <- buttonSensorChan_pull:
			var newQueueUpdate Order_t
			newQueueUpdate.Operation = ADD
			newQueueUpdate.Button = newButtonCall
			queueUpdateBuffer = append(queueUpdateBuffer, newQueueUpdate)

		case networkUpdate := <- network_pull:
			shouldRedistribute := false
			shouldPrint := false
			if len(networkUpdate.Elevators) != numElevators{
				shouldRedistribute = true
				merge_bool_matrix(globalOrders, networkUpdate.GlobalOrders)
				numElevators = len(networkUpdate.Elevators)
			}else{
				copy_bool_matrix(globalOrders, networkUpdate.GlobalOrders)
			}

			localElevator := make_elevator(numFloors, localPosition, localStatus)
			if localPosition != networkUpdate.Elevators[localID].Position{
				shouldPrint = true
				Println("localPosition = ", localPosition)
			}
			copy_bool_matrix(localElevator.Orders, networkUpdate.Elevators[localID].Orders)

			for _, newQueueUpdate := range queueUpdateBuffer{
				if newQueueUpdate.Operation == ADD{
					shouldRedistribute = true
					add_order(&localElevator, newQueueUpdate.Button, globalOrders)
				}else if newQueueUpdate.Operation == DELETE{
					delete_order(&localElevator, newQueueUpdate.Button.Floor, globalOrders)
				}
			}
			queueUpdateBuffer = nil

			update_lights(numFloors, globalOrders, localElevator.Orders, buttonLampChan_push)
			networkUpdate.Elevators[localID] = localElevator
			if shouldRedistribute{
				optimize_order_distribution(&networkUpdate.Elevators, globalOrders)
			}

			copy_bool_matrix(networkUpdate.GlobalOrders, globalOrders)
			network_push <- networkUpdate

			nextDestination := get_next_destination(networkUpdate.Elevators[localID], numPositions)
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

func optimize_order_distribution(elevators *map[string]Elevator_t, commonOrders [][]bool) int {
	var dummyMap = make(map[string]Elevator_t)
	copy_elev_map(dummyMap, *elevators)
	
	for floor := range commonOrders{
		for _, elevator := range dummyMap{
			elevator.Orders[floor][BUTTON_CALL_UP] = false
			elevator.Orders[floor][BUTTON_CALL_DOWN] = false
		}
	}
	worstCase := math.MaxInt64
	return minimize_work(dummyMap, elevators, commonOrders, &worstCase)
}

func minimize_work(originalElevMap map[string]Elevator_t, optimalElevMap *map[string]Elevator_t, commonOrders [][]bool, currentlyBestCost *int) (int) {
	var maxCost int = 0
	for _, elevator := range originalElevMap{
		maxCost = Max(maxCost, calculate_cost(elevator.Position, elevator.Status, elevator.Orders))
	}
	if maxCost > *currentlyBestCost{
		return *currentlyBestCost
	}else if is_empty(commonOrders){
		if maxCost < *currentlyBestCost{
			*currentlyBestCost = maxCost
			copy_elev_map(*optimalElevMap, originalElevMap)
			return maxCost
		}
		return *currentlyBestCost
	}

	tempOrders := make([][]bool, len(commonOrders))
	copy_bool_matrix(tempOrders, commonOrders)
	var bestCost = *currentlyBestCost
	for i := 0; i < len(commonOrders); i++ {
		shouldBreak := false
		for j := 0; j < 2; j++ {
			if tempOrders[i][j] {
				for _, elevator := range originalElevMap{
					elevator.Orders[i][j] = true
					elevator.Orders[i][(j+1)%2] = tempOrders[i][(j+1)%2]
					tempOrders[i][j] = false
					tempOrders[i][(j+1)%2] = false
					bestCost = Min( bestCost, minimize_work(originalElevMap, optimalElevMap, tempOrders, currentlyBestCost) )
					tempOrders[i][j] = true
					tempOrders[i][(j+1)%2] = elevator.Orders[i][ (j+1)%2 ]
					elevator.Orders[i][j] = false
					elevator.Orders[i][(j+1)%2] = false
				}
				shouldBreak = true
				break
			}
		}
		if shouldBreak {
			break
		}
	}
	return bestCost
}

func is_empty(boolMatrix [][]bool) bool {
	for i := 0; i < len(boolMatrix); i++ {
		for j := 0; j < len(boolMatrix[i]); j++ {
			if boolMatrix[i][j]{
				return false
			}			
		}		
	}
	return true
}

func calculate_cost(initialPosition int, initialStatus ElevatorStatus_t, orders [][]bool) (int) {
	numFloors := len(orders)
	numPositions := numFloors*2-1
	tempOrders := make( [][]bool, numFloors )
	copy_bool_matrix(tempOrders, orders) 
	driveTime := 0
	waitTime := 0
	totalTime := 0
	floorToFloorTime := 1
	doorOpenTime := 3
	passingFloorTime := 1
	betweenFloorsTime := 1
	if initialStatus == UNKNOWN{
		return numFloors*(doorOpenTime + floorToFloorTime)*(passingFloorTime + betweenFloorsTime)*2
	}
	if (initialStatus == MOVING_UP || initialStatus == IDLE) {
		distanceTravelledUp := 0
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
				distanceTravelledUp += passingFloorTime
			}else{
				distanceTravelledUp += betweenFloorsTime
			}
		}
		distanceTravelledUp -= passingFloorTime
		distanceTravelledDown := 0
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
				distanceTravelledDown += passingFloorTime
			}else{
				distanceTravelledDown += betweenFloorsTime
			}
		}
		distanceTravelledDown -= passingFloorTime
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
				distanceTravelledUp += passingFloorTime
			}else{
				distanceTravelledUp += betweenFloorsTime
			}
		}
		totalTime = driveTime + waitTime	
	}
	driveTime = 0
	waitTime = 0
	copy_bool_matrix(tempOrders, orders)
	if (initialStatus == MOVING_DOWN || initialStatus == IDLE) {
		distanceTravelledDown := 0
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
				distanceTravelledDown += passingFloorTime
			}else{
				distanceTravelledDown += betweenFloorsTime
			}
		}
		distanceTravelledDown -= passingFloorTime
		distanceTravelledUp := 0
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
				distanceTravelledUp += passingFloorTime
			}else{
				distanceTravelledUp += betweenFloorsTime
			}
		}
		distanceTravelledUp -= passingFloorTime
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
				distanceTravelledDown += passingFloorTime
			}else{
				distanceTravelledDown += betweenFloorsTime
			}
		}
		if initialStatus == IDLE {
			// dersom de er like returneres den siste (tiden nedover).
			totalTime = Min( totalTime, driveTime + waitTime )
		}else{
			totalTime = driveTime + waitTime
		}
	}
	return totalTime
}

func Abs(i int) int {
	return (int)(math.Abs( (float64)(i) ))
}

func Min(a, b int) int {
	return (int)(math.Min((float64)(a),(float64)(b)))
}

func Max(a, b int) int {
	if a > b{
		return a
	}
	return b
}

// Trenger ikke numPositions. Kan dessuten være medlemsfunksjon!
func get_next_destination(elevator Elevator_t, numPositions int) int {
	initialPosition := elevator.Position
	initialStatus := elevator.Status
	orders := elevator.Orders
	destinationUp := -1
	destinationDown := -1
	upTime := math.MaxInt64
	downTime := math.MaxInt64
	shouldDoSomething := false
	if (initialStatus == MOVING_UP || initialStatus == IDLE) {
		for position := initialPosition; position < numPositions; position++{
			if position % 2 == 0{
				floor := position/2
				if orders[floor][BUTTON_CALL_UP] || orders[floor][BUTTON_CALL_INSIDE] {
					destinationUp = position
					upTime = calculate_cost(initialPosition, MOVING_UP, elevator.Orders)
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
						upTime = calculate_cost(initialPosition, MOVING_UP, elevator.Orders)
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
						downTime = calculate_cost(initialPosition, MOVING_DOWN, elevator.Orders)
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
							downTime = calculate_cost(initialPosition, MOVING_DOWN, elevator.Orders)
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

	//Kan fjerne shoudDoSomething ja:
	if (shouldDoSomething && upTime < downTime) {return destinationUp} // Kan istedet skrive if (upTime =! -1 || downTime != -1) && upTime < downTime
	return destinationDown // if else
}


// Trenger ikke presisere bool. Kan fjerne logfatal mot å returnere false istedet
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


// trenger ikke presisere bool.
func copy_bool_matrix(dst, src [][]bool) bool {
	if len(dst) < len(src){
		return false
	}
	for i := range src{
		tempList := make([]bool, len(src[i]))
		copy(tempList, src[i])
		dst[i] = make([]bool, len(src[i]))
		copy(dst[i], tempList)
	}
	return true
}


// Trenger ikke presisere elev her heller da eller?
func copy_elev_map(dst, src map[string]Elevator_t){
	for key, _ := range dst{
		delete(dst, key)
	}
	for ID, e := range src{
		dst[ID] = make_elevator(e.NumFloors, e.Position, e.Status)
		copy_bool_matrix(dst[ID].Orders, e.Orders)
	}
}

// kan fjerne error her og kanskje.
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

// Kan fjerne error og logFatal.
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
		time := calculate_cost(elevators[elevatorList[i]].Position, elevators[elevatorList[i]].Status, elevators[elevatorList[i]].Orders) 
		Printf("Workload: %2.0f", (float64)(time))
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


// func find_optimal_elevator_float64(elevators map[string]Elevator_t, buttonCall ButtonCall_t, orderedFloor int) string {  
// 	bestTime := math.Inf(1)
// 	sortedIPs := make([]string, 0, len(elevators))
// 	for elevatorIP := range elevators{
// 		sortedIPs = append(sortedIPs, elevatorIP)
// 	}
// 	sort.Strings(sortedIPs)
// 	bestElevator := sortedIPs[0]
// 	for _, elevatorIP := range sortedIPs {
// 		elevator := elevators[elevatorIP]
// 		elevStatus := elevator.Status
// 		if elevStatus == UNKNOWN {continue}
// 		tempOrders := make( [][]bool, elevator.NumFloors )
// 		copy_bool_matrix(tempOrders, elevator.Orders) 
// 		tempOrders[orderedFloor][buttonCall] = true
// 		newTravelTime, _ := calculate_cost(elevator.Position, elevator.Status, tempOrders)
// 		if newTravelTime < bestTime {
// 			bestTime = newTravelTime
// 			bestElevator = elevatorIP
// 		}else if newTravelTime == bestTime {
// 			if elevatorIP < bestElevator{
// 				bestElevator = elevatorIP
// 				bestTime = newTravelTime
// 			}
// 		}
// 	}
// 	return bestElevator
// }

// func calculate_cost_float64(initialPosition int, initialStatus ElevatorStatus_t, orders [][]bool) (float64, error) {
// 	if initialPosition < 0 || initialPosition >= len(orders)*2-1{
// 		return math.Inf(1), errors.New("Call to calculate_cost(): Initial position outside range.")
// 	}
// 	if initialStatus == UNKNOWN{
// 		return math.Inf(1), nil
// 	}
// 	numFloors := len(orders)
// 	numPositions := numFloors*2-1
// 	tempOrders := make( [][]bool, numFloors )
// 	copy_bool_matrix(tempOrders, orders) 
// 	var driveTime float64 = 0
// 	var waitTime float64 = 0
// 	var totalTime float64 = 0
// 	var floorToFloorTime float64 = 2.2
// 	var doorOpenTime float64 = 3
// 	if (initialStatus == MOVING_UP || initialStatus == IDLE) {
// 		var distanceTravelledUp float64 = 0
// 		for position := initialPosition; position < numPositions; position++{
// 			if position % 2 == 0{
// 				floor := position/2
// 				if tempOrders[floor][BUTTON_CALL_UP] || tempOrders[floor][BUTTON_CALL_INSIDE] || ( floor == numFloors-1 && tempOrders[floor][BUTTON_CALL_DOWN]){
// 					driveTime += distanceTravelledUp*floorToFloorTime
// 					waitTime += doorOpenTime
// 					tempOrders[floor][BUTTON_CALL_UP] = false
// 					tempOrders[floor][BUTTON_CALL_DOWN] = false
// 					tempOrders[floor][BUTTON_CALL_INSIDE] = false
// 					distanceTravelledUp = 0
// 				}else if position == (initialPosition + 1){
// 					driveTime += distanceTravelledUp*floorToFloorTime
// 					distanceTravelledUp = 0
// 				}
// 				distanceTravelledUp += 0.1
// 			}else{
// 				distanceTravelledUp += 0.9
// 			}
// 		}
// 		distanceTravelledUp -= 0.1
// 		var distanceTravelledDown float64 = 0
// 		for position := numPositions-1; position >= 0; position--{
// 			if position % 2 == 0{
// 				floor := position/2
// 				if tempOrders[floor][BUTTON_CALL_DOWN] || tempOrders[floor][BUTTON_CALL_INSIDE] {
// 					driveTime += Abs(distanceTravelledDown - distanceTravelledUp)*floorToFloorTime
// 					waitTime += doorOpenTime
// 					tempOrders[floor][BUTTON_CALL_UP] = false
// 					tempOrders[floor][BUTTON_CALL_DOWN] = false
// 					tempOrders[floor][BUTTON_CALL_INSIDE] = false
// 					distanceTravelledDown = 0
// 					distanceTravelledUp = 0
// 				}
// 				distanceTravelledDown += 0.1
// 			}else{
// 				distanceTravelledDown += 0.9
// 			}
// 		}
// 		distanceTravelledDown -= 0.1
// 		for position := 0; position <= initialPosition; position++{
// 			if position % 2 == 0{
// 				floor := position/2
// 				if tempOrders[floor][BUTTON_CALL_UP] || tempOrders[floor][BUTTON_CALL_INSIDE] {
// 					driveTime += Abs(distanceTravelledUp - distanceTravelledDown)*floorToFloorTime
// 					waitTime += doorOpenTime
// 					tempOrders[floor][BUTTON_CALL_UP] = false
// 					tempOrders[floor][BUTTON_CALL_DOWN] = false
// 					tempOrders[floor][BUTTON_CALL_INSIDE] = false
// 					distanceTravelledDown = 0
// 					distanceTravelledUp = 0
// 				}
// 				distanceTravelledUp += 0.1
// 			}else{
// 				distanceTravelledUp += 0.9
// 			}
// 		}
// 		totalTime = driveTime + waitTime	
// 	}
// 	driveTime = 0
// 	waitTime = 0
// 	copy_bool_matrix(tempOrders, orders)
// 	if (initialStatus == MOVING_DOWN || initialStatus == IDLE) {
// 		var distanceTravelledDown float64 = 0
// 		for position := initialPosition; position >= 0; position--{
// 			if position % 2 == 0{
// 				floor := position/2
// 				if tempOrders[floor][BUTTON_CALL_DOWN] || tempOrders[floor][BUTTON_CALL_INSIDE] || ( (floor == 0) && tempOrders[floor][BUTTON_CALL_UP]){
// 					driveTime += distanceTravelledDown*floorToFloorTime
// 					waitTime += doorOpenTime
// 					tempOrders[floor][BUTTON_CALL_UP] = false
// 					tempOrders[floor][BUTTON_CALL_DOWN] = false
// 					tempOrders[floor][BUTTON_CALL_INSIDE] = false
// 					distanceTravelledDown = 0
// 				}else if position == (initialPosition - 1){
// 					driveTime += distanceTravelledDown*floorToFloorTime
// 					distanceTravelledDown = 0
// 				}
// 				distanceTravelledDown += 0.1
// 			}else{
// 				distanceTravelledDown += 0.9
// 			}
// 		}
// 		distanceTravelledDown -= 0.1
// 		var distanceTravelledUp float64 = 0
// 		for position := 0; position < numPositions; position++{
// 			if position % 2 == 0{
// 				floor := position/2
// 				if tempOrders[floor][BUTTON_CALL_UP] || tempOrders[floor][BUTTON_CALL_INSIDE] {
// 					driveTime += Abs(distanceTravelledUp - distanceTravelledDown)*floorToFloorTime
// 					waitTime += doorOpenTime
// 					tempOrders[floor][BUTTON_CALL_UP] = false
// 					tempOrders[floor][BUTTON_CALL_DOWN] = false
// 					tempOrders[floor][BUTTON_CALL_INSIDE] = false
// 					distanceTravelledDown = 0
// 					distanceTravelledUp = 0
// 				}
// 				distanceTravelledUp += 0.1
// 			}else{
// 				distanceTravelledUp += 0.9
// 			}
// 		}
// 		distanceTravelledUp -= 0.1
// 		for position := numPositions-1; position >= initialPosition; position--{
// 			if position % 2 == 0{
// 				floor := position/2
// 				if tempOrders[floor][BUTTON_CALL_DOWN] || tempOrders[floor][BUTTON_CALL_INSIDE] {
// 					driveTime += Abs(distanceTravelledDown - distanceTravelledUp)*floorToFloorTime
// 					waitTime += doorOpenTime
// 					tempOrders[floor][BUTTON_CALL_UP] = false
// 					tempOrders[floor][BUTTON_CALL_DOWN] = false
// 					tempOrders[floor][BUTTON_CALL_INSIDE] = false
// 					distanceTravelledDown = 0
// 					distanceTravelledUp = 0
// 				}
// 				distanceTravelledDown += 0.1
// 			}else{
// 				distanceTravelledDown += 0.9
// 			}
// 		}
// 		if initialStatus == IDLE {
// 			// dersom de er like returneres den siste (tiden nedover).
// 			totalTime = Min( totalTime, driveTime + waitTime )
// 		}else{
// 			totalTime = driveTime + waitTime
// 		}
// 	}
// 	return totalTime, nil
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
