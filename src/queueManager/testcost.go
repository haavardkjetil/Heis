package main

import (
.	"math"
"errors"
.	"fmt"
"sort"
"time"
"math/rand"
)

type ButtonCall_t int
const(
	BUTTON_CALL_UP 		ButtonCall_t = iota
	BUTTON_CALL_DOWN
	BUTTON_CALL_INSIDE
)

type ElevatorStatus_t int
const(
	MOVING_UP 	ElevatorStatus_t = iota
	MOVING_DOWN
	IDLE
	UNKNOWN
	DOOR_OPEN
)

type Elevator_t struct{
	Status ElevatorStatus_t
	Position int
	NumPositions int
	NumFloors int
	Orders[][]bool
}

func main(){


	numFloors := 20
	orders := make([][]bool, numFloors)
	for i := 0; i < numFloors; i++ {
		orders[i] = make([]bool, 2)
	}
	ID := "Mr. Cool 	"
	ID2 := "Mr. Douch	"
	// ID3 := "Mr. Duck		"
	// ID4 := "Mr. Foxy		"
	// ID5 := "Mr. Foxy		"
	// ID6 := "Mr. Foxy		"
	globalElevators := make( map[string]Elevator_t )
	for{
		globalElevators[ID] = make_elevator(numFloors, rand.Intn(numFloors*2-1), MOVING_UP)
		globalElevators[ID2] = make_elevator(numFloors, rand.Intn(numFloors*2-1), MOVING_DOWN)

		fill_queue(orders)
		c := time.Tick(1 * time.Millisecond)
		t1 := <- c
		bestTime := optimize_order_distribution(&globalElevators, orders)
		Println("Med superalgoritmen: ", time.Since(t1))

		print_queues(globalElevators)
		t1 = <- c
		redistribute_orders(globalElevators, orders)
		Println("Med kjip algoritme...: ", time.Since(t1))
		print_queues(globalElevators)
		maxCost := 0.0
		for _, elevator := range globalElevators{
			cost, _ := calculate_cost(elevator.Position, elevator.Status, elevator.Orders)
			if cost > maxCost{
				maxCost = cost
			}
		}
		Printf("Resultat: %.1f - %.1f = %.1f", bestTime, maxCost, bestTime-maxCost)
		Println("\n\n\n\n\n\n\n")
		
	}
	
}

func optimize_order_distribution(elevators *map[string]Elevator_t, commonOrders [][]bool) float64 {
	var dummyMap = make(map[string]Elevator_t)
	copy_elev_map(dummyMap, *elevators)
	worstCase := Inf(1)
	return find_minimum_work(dummyMap, *elevators, commonOrders, &worstCase)
}

func find_minimum_work(elevators, throwawayElevators map[string]Elevator_t, commonOrders [][]bool, currentlyBest *float64) (float64) {
	var maxCost float64 = 0
	for _, elevator := range elevators{
		cost, _ := calculate_cost(elevator.Position, elevator.Status, elevator.Orders)
		if cost > maxCost{
			maxCost = cost
		}
	}
	if is_empty(commonOrders){
		if maxCost < *currentlyBest{
			*currentlyBest = maxCost
			copy_elev_map(throwawayElevators, elevators)
			return maxCost
		}
		return *currentlyBest
	}else if maxCost > *currentlyBest{
		return *currentlyBest
	}

	tempOrders := make([][]bool, len(commonOrders))
	copy_bool_matrix(tempOrders, commonOrders)
	var bestCost float64 = Inf(1)
	for i := 0; i < len(commonOrders); i++ {
		shouldBreak := false
		for j := 0; j < 2; j++ {
			if tempOrders[i][j] {
				for _, elevator := range elevators{
					elevator.Orders[i][j] = true
					elevator.Orders[i][(j+1)%2] = tempOrders[i][(j+1)%2]
					tempOrders[i][j] = false
					tempOrders[i][(j+1)%2] = false
					var cost float64
					cost = find_minimum_work(elevators, throwawayElevators, tempOrders, currentlyBest)
					if cost < bestCost{
						bestCost = cost
					}
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

func ordersInQueue(boolMatrix [][]bool) int {
	count := 0
	for i := 0; i < len(boolMatrix); i++ {
		for j := 0; j < len(boolMatrix[i]); j++ {
			if boolMatrix[i][j]{
				count++
			}			
		}		
	}
	return count
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

func fill_queue(boolMatrix [][]bool) {
	for i := 0; i < len(boolMatrix); i++ {
		for j := 0; j < len(boolMatrix[i]); j++ {
			if rand.Intn(1) == 0{
				boolMatrix[i][j] = true
			}else{
				boolMatrix[i][j] = false
			}
				
		}		
	}
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
			elevators[bestElevator].Orders[floor][BUTTON_CALL_UP] = true
		}
		if sharedOrders[floor][BUTTON_CALL_DOWN] {
			bestElevator := find_optimal_elevator(elevators, BUTTON_CALL_DOWN, floor)
			elevators[bestElevator].Orders[floor][BUTTON_CALL_DOWN] = true
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
	Println("Best elevator for ", orderedFloor, buttonCall, ": ", bestElevator, ". Time: ", bestTime)
	return bestElevator
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

func copy_elev_map(dst, src map[string]Elevator_t){
	for key, _ := range dst{
		delete(dst, key)
	}
	for ID, e := range src{
		dst[ID] = make_elevator(e.NumFloors, e.Position, e.Status)
		copy_bool_matrix(dst[ID].Orders, e.Orders)
	}
}

func make_elevator(numFloors, position int, status ElevatorStatus_t) Elevator_t{
	e := Elevator_t{status, position, numFloors*2-1, numFloors, make([][]bool, numFloors)}
	for floor := 0; floor < numFloors; floor++ {
		e.Orders[floor] = make([]bool, 3)
	}
	return e
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