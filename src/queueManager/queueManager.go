package queueManager

import (
.	"math"
"errors"
.	"fmt"
"sort"
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

//QueueDatagram_t
type UpdatePacket_t struct{
	Elevators map[string]Elevator_t
	GlobalOrders[][]bool
}

type Order_t struct{
	Operation OrderCommand_t
	ButtonCall ButtonCall_t
	Floor int
}

func copy_map(dst, src map[string]Elevator_t){
	for key, _ := range dst{
		delete(dst, key)
	}
	for key, element := range src{
		dst[key] = element
	}
}

func RunQueueManager(localIP string, 
	numFloors int, 
	networkReceive, networkTransmit chan UpdatePacket_t, 
	orderChan chan Order_t, 
	statusChan chan ElevatorStatus_t, 
	elevatorCommandChan chan ElevatorCommand_t, 
	positionChan, quitChan chan int){

	globalElevators := make( map[string]Elevator_t )
	localElevator := Elevator_t{
		IDLE, 0, numFloors*2-1, numFloors, make([][]bool, numFloors),
	}
	var globalOrders = make( [][]bool, numFloors)
	for floor := 0; floor < numFloors; floor++ {
		localElevator.Orders[floor] = make([]bool, 3)
		globalOrders[floor] = make([]bool, 2)
		localElevator.Orders[floor][BUTTON_CALL_UP] = false
		localElevator.Orders[floor][BUTTON_CALL_DOWN] = false
		localElevator.Orders[floor][BUTTON_CALL_INSIDE] = false
		globalOrders[floor][BUTTON_CALL_UP] = false
		globalOrders[floor][BUTTON_CALL_DOWN] = false
	}

	globalElevators[localIP] = localElevator
	var futureOrderUpdates []Order_t 
	initialPosition := <- positionChan
	update_elevator_position(&localElevator, initialPosition)
	networkTransmit <- UpdatePacket_t{
		globalElevators, globalOrders,
	}
	Println("Elevator queue system initialized.")
	for{
		select{
			case newStatus := <- statusChan:
				localElevator.Status = newStatus
			case newOrderUpdate := <- orderChan:
				if newOrderUpdate.Operation == ADD{
					add_order(&localElevator, newOrderUpdate.Floor, newOrderUpdate.ButtonCall, globalOrders)
				}else{
					delete_order(&localElevator, newOrderUpdate.Floor, newOrderUpdate.ButtonCall, globalOrders)
				}
				redistribute_orders(&globalElevators, globalOrders)
				PrintOrderQueues(globalElevators)
				futureOrderUpdates = append(futureOrderUpdates, newOrderUpdate)
				//Calculate action -> send recommended action
			case newPosition := <- positionChan:
				update_elevator_position(&localElevator, newPosition)
				//Calculate action -> send recommended action
			case networkUpdate := <- networkReceive:
				//copy public system info:
				copy_map(globalElevators, networkUpdate.Elevators)
				merge_bool_matrix(globalOrders, networkUpdate.GlobalOrders)
				//add new local system info:
				//networkUpdate.Elevators[localIP] = localElevator

				//merge global orders:
				for _, newOrder := range futureOrderUpdates{
					if newOrder.Operation == ADD{
						add_order(&localElevator, newOrder.Floor, newOrder.ButtonCall, globalOrders)
					}else{
						delete_order(&localElevator, newOrder.Floor, newOrder.ButtonCall, globalOrders)
					}
				}
				futureOrderUpdates = nil
				globalElevators[localIP] = localElevator
				//redistribute
				redistribute_orders(&globalElevators, globalOrders)
				copy_map(networkUpdate.Elevators, globalElevators)
				merge_bool_matrix(networkUpdate.GlobalOrders, globalOrders)
				//deliver updated packet
				networkTransmit <- networkUpdate
				PrintOrderQueues(networkUpdate.Elevators)
				//Calculate action -> send recommended action
			case <- quitChan:
				redistribute_orders(&globalElevators, globalOrders)
				PrintOrderQueues(globalElevators)
				return 
			default:
				break
		}
	}
}

func update_elevator_position(elevator *Elevator_t, position int) error{
	if position < 0 || position > elevator.NumFloors {
		return errors.New("Attempted to update position: position out of range.")
	}
	elevator.Position = position
	return nil
}

func find_optimal_elevator(elevators map[string]Elevator_t, buttonCall ButtonCall_t, orderedFloor int) string {  
	bestTime := Inf(1)
	sortedElevatorIPs := make([]string, 0, len(elevators))
	for elevatorIP := range elevators{
		sortedElevatorIPs = append(sortedElevatorIPs, elevatorIP)
	}
	sort.Strings(sortedElevatorIPs)
	bestElevator := sortedElevatorIPs[0]

	for _, elevatorIP := range sortedElevatorIPs {
		elevator := elevators[elevatorIP]
		elevStatus := elevator.Status
		if elevStatus == UNKNOWN {continue}
		tempOrders := make( [][]bool, elevator.NumFloors )
		copy_bool_matrix(tempOrders, elevator.Orders) 
		previousTravelTime, err := calculate_cost(elevator.Position, elevator.NumPositions, elevator.NumFloors, elevator.Status, tempOrders)
		if err != nil{
			Println("Error in calculated_traveltime(", elevatorIP, "): ", err)
			continue
		}
		tempOrders[orderedFloor][buttonCall] = true
		newTravelTime, err := calculate_cost(elevator.Position, elevator.NumPositions, elevator.NumFloors, elevator.Status, tempOrders)
		if err != nil{
			Println(err)
			continue
		}
		if newTravelTime == previousTravelTime {
			bestTime = newTravelTime
			bestElevator = elevatorIP
			break
		}else if newTravelTime < bestTime {
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

func calculate_cost(initialPosition, numPositions, numFloors int, initialStatus ElevatorStatus_t, orders [][]bool) (float64, error) {
	if initialPosition < 0 || initialPosition >= numPositions{
		return Inf(1), errors.New("Call to calculate_cost(): Initial position outside range.")
	}else if len(orders) != numFloors {
		return Inf(1), errors.New("Call to calculate_cost(): Number of floors doesnt match length of orderlist.")
	}
	if initialStatus == UNKNOWN{
		return Inf(1), nil
	}
	tempOrders := make( [][]bool, numFloors )
	copy_bool_matrix(tempOrders, orders) 
	var driveTime float64 = 0
	var waitTime float64 = 0
	var totalTime float64
	if (initialStatus == MOVING_UP || initialStatus == IDLE) {
		var floorsPassedUpwards float64 = 0
		for position := initialPosition; position < numPositions; position++{
			if position % 2 == 0{
				floor := position/2
				if tempOrders[floor][BUTTON_CALL_UP] || tempOrders[floor][BUTTON_CALL_INSIDE] {
					driveTime += floorsPassedUpwards
					waitTime += 3
					tempOrders[floor][BUTTON_CALL_UP] = false
					tempOrders[floor][BUTTON_CALL_DOWN] = false
					tempOrders[floor][BUTTON_CALL_INSIDE] = false
					floorsPassedUpwards = 0
				}
				floorsPassedUpwards++
			}
		}
		floorsPassedUpwards--
		var floorsPassedDownwards float64 = 0
		for position := numPositions; position >= 0; position--{
			if position % 2 == 0{
				floor := position/2
				if tempOrders[floor][BUTTON_CALL_DOWN] || tempOrders[floor][BUTTON_CALL_INSIDE] {
					driveTime += Abs(floorsPassedDownwards - floorsPassedUpwards)
					waitTime += 3
					tempOrders[floor][BUTTON_CALL_UP] = false
					tempOrders[floor][BUTTON_CALL_DOWN] = false
					tempOrders[floor][BUTTON_CALL_INSIDE] = false
					floorsPassedDownwards = 0
					floorsPassedUpwards = 0
				}
				floorsPassedDownwards++
			}
		}
		floorsPassedDownwards--
		for position := 0; position <= initialPosition; position++{
			if position % 2 == 0{
				floor := position/2
				if tempOrders[floor][BUTTON_CALL_UP] || tempOrders[floor][BUTTON_CALL_INSIDE] {
					driveTime += Abs(floorsPassedUpwards - floorsPassedDownwards)
					waitTime += 3
					tempOrders[floor][BUTTON_CALL_UP] = false
					tempOrders[floor][BUTTON_CALL_DOWN] = false
					tempOrders[floor][BUTTON_CALL_INSIDE] = false
					floorsPassedDownwards = 0
					floorsPassedUpwards = 0
				}
				floorsPassedUpwards++
			}
		}
		totalTime = driveTime + waitTime
	}
	driveTime = 0
	waitTime = 0
	copy_bool_matrix(tempOrders, orders)
	if (initialStatus == MOVING_DOWN || initialStatus == IDLE) {
		var floorsPassedDownwards float64 = 0
		for position := initialPosition; position >= 0; position--{
			if position % 2 == 0{
				floor := position/2
				if tempOrders[floor][BUTTON_CALL_DOWN] || tempOrders[floor][BUTTON_CALL_INSIDE] {
					driveTime += floorsPassedDownwards
					waitTime += 3
					tempOrders[floor][BUTTON_CALL_UP] = false
					tempOrders[floor][BUTTON_CALL_DOWN] = false
					tempOrders[floor][BUTTON_CALL_INSIDE] = false
					floorsPassedDownwards = 0
				}
				floorsPassedDownwards++
			}
		}
		floorsPassedDownwards--
		var floorsPassedUpwards float64 = 0
		for position := 0; position < numPositions; position++{
			if position % 2 == 0{
				floor := position/2
				if tempOrders[floor][BUTTON_CALL_UP] || tempOrders[floor][BUTTON_CALL_INSIDE] {
					driveTime += Abs(floorsPassedUpwards - floorsPassedDownwards)
					waitTime += 3
					tempOrders[floor][BUTTON_CALL_UP] = false
					tempOrders[floor][BUTTON_CALL_DOWN] = false
					tempOrders[floor][BUTTON_CALL_INSIDE] = false
					floorsPassedDownwards = 0
					floorsPassedUpwards = 0
				}
				floorsPassedUpwards++
			}
		}
		floorsPassedUpwards--
		for position := numPositions; position >= initialPosition; position--{
			if position % 2 == 0{
				floor := position/2
				if tempOrders[floor][BUTTON_CALL_DOWN] || tempOrders[floor][BUTTON_CALL_INSIDE] {
					driveTime += Abs(floorsPassedDownwards - floorsPassedUpwards)
					waitTime += 3
					tempOrders[floor][BUTTON_CALL_UP] = false
					tempOrders[floor][BUTTON_CALL_DOWN] = false
					tempOrders[floor][BUTTON_CALL_INSIDE] = false
					floorsPassedDownwards = 0
					floorsPassedUpwards = 0
				}
				floorsPassedDownwards++
			}
		}
		if initialStatus == IDLE {
			totalTime = Min( totalTime, driveTime + waitTime )
		}else{
			totalTime = driveTime + waitTime
		}
	}
	return totalTime, nil
}

func redistribute_orders(elevators *(map[string]Elevator_t), sharedOrders [][]bool){
	for floor := range sharedOrders{
		for _, elevator := range *elevators{
			elevator.Orders[floor][BUTTON_CALL_UP] = false
			elevator.Orders[floor][BUTTON_CALL_DOWN] = false
		}
	}
	for floor := range sharedOrders{
		if sharedOrders[floor][BUTTON_CALL_UP] {
			bestElevator := find_optimal_elevator(*elevators, BUTTON_CALL_UP, floor)
			(*elevators)[bestElevator].Orders[floor][BUTTON_CALL_UP] = true
		}
		if sharedOrders[floor][BUTTON_CALL_DOWN] {
			bestElevator := find_optimal_elevator(*elevators, BUTTON_CALL_DOWN, floor)
			(*elevators)[bestElevator].Orders[floor][BUTTON_CALL_DOWN] = true
		}
	}
}

func merge_bool_matrix(dst, src [][]bool) bool {
	if dst == nil || src == nil || len(dst) != len(src){
		return false
	}
	for i := range src{
		if len(dst[i]) != len(src[i]){
			return false
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
		dst[i] = make([]bool, len(src[i]))
		copy(dst[i], src[i])
	}
	return true
}

func delete_order(source *Elevator_t, floor int, buttonCall ButtonCall_t, globalOrders [][]bool) error{
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

func add_order(source *Elevator_t, floor int, buttonCall ButtonCall_t, globalOrders [][]bool) error{
	if floor < 0 || floor >= source.NumFloors || floor >= len(globalOrders){
		return errors.New("Call to add_order(): floor does not exist.")
	}else if !(buttonCall == BUTTON_CALL_UP || buttonCall == BUTTON_CALL_DOWN || buttonCall == BUTTON_CALL_INSIDE) {
		return errors.New("Call to add_order(): invalid buttoncall.")
	}
	if buttonCall == BUTTON_CALL_INSIDE{
		source.Orders[floor][BUTTON_CALL_INSIDE] = true
	}else{
		globalOrders[floor][buttonCall] = true
	}
	return nil
}

// Ikke testet enda, sikkert full av bugs
func calculate_next_action(elevator Elevator_t) ElevatorCommand_t{
	shouldDoSomething := false
	position := elevator.Position
	status := elevator.Status
	for floor := 0; floor < elevator.NumFloors; floor++{
		if elevator.Orders[floor][BUTTON_CALL_UP] || elevator.Orders[floor][BUTTON_CALL_INSIDE]{
			if(position%2 == 0 && position/2 == floor && ( status == MOVING_UP || status == IDLE)){
				return OPEN_DOORS
			}
			shouldDoSomething = true
			break
		}
		if elevator.Orders[floor][BUTTON_CALL_DOWN] || elevator.Orders[floor][BUTTON_CALL_INSIDE]{
			if(position%2 == 0 && position/2 == floor && ( status == MOVING_DOWN || status == IDLE)){
				return OPEN_DOORS
			}
			shouldDoSomething = true
			break
		}
	}
	if !shouldDoSomething{
		return GO_IDLE
	}
	switch status{
	case IDLE:
		elevator.Status = MOVING_UP
		UPtime, _ := calculate_cost(elevator.Position, elevator.NumPositions, elevator.NumFloors, elevator.Status, elevator.Orders)
		elevator.Status = MOVING_DOWN
		DOWNtime, _ := calculate_cost(elevator.Position, elevator.NumPositions, elevator.NumFloors, elevator.Status, elevator.Orders)
		elevator.Status = IDLE
		if UPtime < DOWNtime{
			return GO_UP
		}else{
			return GO_DOWN
		}
	case MOVING_UP:
		for floor := position/2 +1; floor < elevator.NumFloors; floor++{
			if elevator.Orders[floor][BUTTON_CALL_UP] || elevator.Orders[floor][BUTTON_CALL_DOWN] || elevator.Orders[floor][BUTTON_CALL_INSIDE]{
				return GO_UP
			}
		}
	case MOVING_DOWN:
		for floor := (position+1)/2-1; floor >= 0; floor--{
			if elevator.Orders[floor][BUTTON_CALL_UP] || elevator.Orders[floor][BUTTON_CALL_DOWN] || elevator.Orders[floor][BUTTON_CALL_INSIDE]{
				return GO_DOWN
			}
		}
	}
	return GO_IDLE
}

func PrintOrderQueues(elevators map[string]Elevator_t){
	elevatorList := make([]string, 0, len(elevators))
	for elevatorIP := range elevators{
		elevatorList = append(elevatorList, elevatorIP)
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
		if elevators[elevatorList[i]].Status == MOVING_UP {status = "MOVING UP"}
		if elevators[elevatorList[i]].Status == MOVING_DOWN {status = "MOVING DOWN"}
		if elevators[elevatorList[i]].Status == IDLE {status = "IDLE"}
		Print("Status: ", status, "				")
	}
	Println("")
	for i := range elevatorList{
		time, _ := calculate_cost(elevators[elevatorList[i]].Position, elevators[elevatorList[i]].NumPositions, elevators[elevatorList[i]].NumFloors, elevators[elevatorList[i]].Status, elevators[elevatorList[i]].Orders) 
		Print("Workload: ", time, " [s]", "				")
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

