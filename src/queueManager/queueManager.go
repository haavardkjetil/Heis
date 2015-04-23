package queueManager

import (
"stateMachine"
"driver"
)

import(
"math"
."fmt"
"sort"
"log"
"os"
"bytes"
"encoding/gob"
"io/ioutil"
"time"
)

const BACKUP_FILE_NAME = "backup"

type Elevator_t struct{
	Status stateMachine.ElevatorStatus_t
	Position int
	NumPositions int
	NumFloors int
	Orders[][]bool
}

type Datagram_t struct{
	Elevators map[string]Elevator_t
	GlobalOrders[][]bool
}

type buttonCall_t int
const(
	BUTTON_CALL_UP 		buttonCall_t = iota
	BUTTON_CALL_DOWN
	BUTTON_CALL_INSIDE
)

type  orderType_t int
const(
	DELETE	orderType_t = -1
	ADD 	orderType_t = 1
)

type queueUpdate_t struct{
	Type orderType_t
	Button driver.Button_t
}


func Run(localID string, 
					numFloors int, 
					networkToQueue_c, queueToNetwork_c chan Datagram_t, 
					status_c chan stateMachine.StatusUpdate_t, 
	      			buttonSensor_c chan driver.Button_t,
	      			buttonLamp_c chan driver.ButtonLampUpdate_t,
					destination_c chan int){
	var statusUpdate stateMachine.StatusUpdate_t
	init_timer := time.NewTimer( 4*time.Second )
	select{
		case statusUpdate = <- status_c:
			break
		case <- init_timer.C:
			log.Fatal("QueueManager was not initialized.")
	}
	localStatus := statusUpdate.Status
	localPosition := statusUpdate.Position

	var globalOrders = make( [][]bool, numFloors)
	for floor := 0; floor < numFloors; floor++ {
		globalOrders[floor] = make([]bool, 2)
	}
	globalElevators := make( map[string]Elevator_t )
	globalElevators[localID] = make_elevator(numFloors, localPosition, localStatus)

	get_backup(BACKUP_FILE_NAME, globalElevators[localID].Orders)
	queueToNetwork_c <- Datagram_t{ globalElevators, globalOrders}

	numElevators := len(globalElevators)
	var queueUpdateBuffer []queueUpdate_t 
	for{
		select{
		case newStatusUpdate := <- status_c:
			localPosition = newStatusUpdate.Position
			if newStatusUpdate.Status == stateMachine.DOOR_OPEN{
				var newQueueUpdate queueUpdate_t
				newQueueUpdate.Type = DELETE
				newQueueUpdate.Button.Floor = localPosition/2
				queueUpdateBuffer = append(queueUpdateBuffer, newQueueUpdate)
			}else{
				localStatus = newStatusUpdate.Status
			}

		case newButtonCall := <- buttonSensor_c:
			var newQueueUpdate queueUpdate_t
			newQueueUpdate.Type = ADD
			newQueueUpdate.Button = newButtonCall
			queueUpdateBuffer = append(queueUpdateBuffer, newQueueUpdate)

		case networkUpdate := <- networkToQueue_c:
			shouldOptimize := false
			if len(networkUpdate.Elevators) != numElevators{
				shouldOptimize = true
				merge_matrix(globalOrders, networkUpdate.GlobalOrders)
				numElevators = len(networkUpdate.Elevators)
			}else{
				copy_matrix(globalOrders, networkUpdate.GlobalOrders)
			}

			localElevator := make_elevator(numFloors, localPosition, localStatus)
			copy_matrix(localElevator.Orders, networkUpdate.Elevators[localID].Orders)

			for _, newQueueUpdate := range queueUpdateBuffer{
				if newQueueUpdate.Type == ADD{
					shouldOptimize = true
					add_order(&localElevator, newQueueUpdate.Button, globalOrders)
				}else if newQueueUpdate.Type == DELETE{
					delete_order(&localElevator, newQueueUpdate.Button.Floor, globalOrders)
				}
			}
			queueUpdateBuffer = nil
			update_lights(numFloors, globalOrders, localElevator.Orders, buttonLamp_c)
			nextDestination := localElevator.get_next_destination()
			networkUpdate.Elevators[localID] = localElevator
			copy_matrix(networkUpdate.GlobalOrders, globalOrders)
			if shouldOptimize{
				optimize_order_distribution(&networkUpdate.Elevators, networkUpdate.GlobalOrders)
			}
			queueToNetwork_c <- networkUpdate
			destination_c <- nextDestination
			print_queues(networkUpdate.Elevators)
		}
	}
}

func make_elevator(numFloors, position int, status stateMachine.ElevatorStatus_t) Elevator_t{
	e := Elevator_t{status, position, numFloors*2-1, numFloors, make([][]bool, numFloors)}
	for floor := 0; floor < numFloors; floor++ {
		e.Orders[floor] = make([]bool, 3)
	}
	return e
}

func (elevator *Elevator_t) get_next_destination() int {
	numPositions := elevator.NumPositions
	initialPosition := elevator.Position
	initialStatus := elevator.Status
	orders := elevator.Orders
	destinationUp := -1
	destinationDown := -1
	upTime := math.MaxInt64
	downTime := math.MaxInt64
	if (initialStatus == stateMachine.MOVING_UP || initialStatus == stateMachine.IDLE) {
		for position := initialPosition; position < numPositions; position++{
			if position % 2 == 0{
				floor := position/2
				if orders[floor][BUTTON_CALL_UP] || orders[floor][BUTTON_CALL_INSIDE] {
					destinationUp = position
					upTime = calculate_cost(initialPosition, stateMachine.MOVING_UP, elevator.Orders)
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
						upTime = calculate_cost(initialPosition, stateMachine.MOVING_UP, elevator.Orders)
						break
					}
				}
			}
		}
	}
	if (initialStatus == stateMachine.MOVING_DOWN || initialStatus == stateMachine.IDLE) {
		for position := initialPosition; position >= 0; position--{
			if position % 2 == 0{
				floor := position/2
				if orders[floor][BUTTON_CALL_DOWN] || orders[floor][BUTTON_CALL_INSIDE]{
					if initialStatus == stateMachine.IDLE {
						downTime = calculate_cost(initialPosition, stateMachine.MOVING_DOWN, elevator.Orders)
						if downTime <= upTime{
							destinationDown = position
						}
					}else{
						destinationDown = position
					}
					break
				}
			}
		}
		if destinationDown == -1 {
			for position := 0; position <= initialPosition; position++{
				if position % 2 == 0{
					floor := position/2
					if orders[floor][BUTTON_CALL_UP]{
						if initialStatus == stateMachine.IDLE {
							downTime = calculate_cost(initialPosition, stateMachine.MOVING_DOWN, elevator.Orders)
							if downTime <= upTime{
								destinationDown = position
							}
						}else{
							destinationDown = position
						}
						break
					}
				}
			}
		}
	}
	if (upTime != math.MaxInt64 || downTime != math.MaxInt64) && upTime < downTime{
		return destinationUp
	}else{
		return destinationDown 
	}
}

func optimize_order_distribution(elevators *map[string]Elevator_t, commonOrders [][]bool) int {
	var dummyMap = make(map[string]Elevator_t)
	copy_map(dummyMap, *elevators)
	
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
			copy_map(*optimalElevMap, originalElevMap)
			return maxCost
		}
		return *currentlyBestCost
	}

	tempOrders := make([][]bool, len(commonOrders))
	copy_matrix(tempOrders, commonOrders)
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

func calculate_cost(initialPosition int, initialStatus stateMachine.ElevatorStatus_t, orders [][]bool) (int) {
	numFloors := len(orders)
	numPositions := numFloors*2-1
	tempOrders := make( [][]bool, numFloors )
	copy_matrix(tempOrders, orders) 
	driveTime := 0
	waitTime := 0
	totalTime := 0
	floorToFloorTime := 1
	doorOpenTime := 3
	passingFloorTime := 1
	betweenFloorsTime := 1
	if initialStatus == stateMachine.UNKNOWN{
		return numFloors*(doorOpenTime + floorToFloorTime)*(passingFloorTime + betweenFloorsTime)*2
	}
	if (initialStatus == stateMachine.MOVING_UP || initialStatus == stateMachine.IDLE) {
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
	copy_matrix(tempOrders, orders)
	if (initialStatus == stateMachine.MOVING_DOWN || initialStatus == stateMachine.IDLE) {
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
		if initialStatus == stateMachine.IDLE {
			totalTime = Min( totalTime, driveTime + waitTime )
		}else{
			totalTime = driveTime + waitTime
		}
	}
	return totalTime
}

func delete_order(source *Elevator_t, floor int, globalOrders [][]bool){
	globalOrders[floor][BUTTON_CALL_DOWN] = false
	globalOrders[floor][BUTTON_CALL_UP] = false
	source.Orders[floor][BUTTON_CALL_INSIDE] = false
	source.Orders[floor][BUTTON_CALL_UP] = false
	source.Orders[floor][BUTTON_CALL_DOWN] = false
	save_backup(BACKUP_FILE_NAME, source.Orders)
}

func add_order(source *Elevator_t, activeButton driver.Button_t, globalOrders [][]bool){
	floor := activeButton.Floor
	if activeButton.Type == driver.BUTTON_CALL_INSIDE{
		source.Orders[floor][BUTTON_CALL_INSIDE] = true
	}else if activeButton.Type == driver.BUTTON_CALL_UP{
		globalOrders[floor][BUTTON_CALL_UP] = true
	}else if activeButton.Type == driver.BUTTON_CALL_DOWN{
		globalOrders[floor][BUTTON_CALL_DOWN] = true
	}
	save_backup(BACKUP_FILE_NAME, source.Orders)
}

func update_lights(numFloors int, globalOrders, localOrders [][]bool, buttonLamp_c chan driver.ButtonLampUpdate_t){
	for floor := 0; floor < numFloors; floor++{
		button := driver.Button_t{
			driver.BUTTON_CALL_UP, floor,
		}
		if floor != numFloors-1{
			buttonLamp_c <- driver.ButtonLampUpdate_t{
				button, globalOrders[floor][BUTTON_CALL_UP],
			} 
		}
		if floor != 0{		
			button.Type = driver.BUTTON_CALL_DOWN
			buttonLamp_c <- driver.ButtonLampUpdate_t{
				button, globalOrders[floor][BUTTON_CALL_DOWN],
			}
		}
		button.Type = driver.BUTTON_CALL_INSIDE
		buttonLamp_c <- driver.ButtonLampUpdate_t{
			button, localOrders[floor][BUTTON_CALL_INSIDE],
		} 
	}
}

func save_backup(fileName string, orders [][]bool){
	f, err := os.Create(fileName)
    if err != nil {
		return
	}
    defer f.Close()

	var writeBuffer bytes.Buffer
	encoder := gob.NewEncoder(&writeBuffer)
	byteBuffer := make( []byte, 1600)

	err = encoder.Encode(orders)
	if err != nil {
		return
	}
	writeBuffer.Read(byteBuffer)
	f.Write(byteBuffer)
    f.Sync()
}

func get_backup(fileName string, orders [][]bool){
	f, err := os.Open(fileName)
    if err != nil {
		return
    }
    defer f.Close()

	byteBuffer := make( []byte, 1600)
	var readBuffer bytes.Buffer
	decoder := gob.NewDecoder(&readBuffer)
	byteBuffer, err = ioutil.ReadFile(fileName)
	if err != nil {
		return
	}
	readBuffer.Write(byteBuffer)
	var tempOrders = make([][]bool, len(orders))
	for floor := 0; floor < len(orders); floor++{
		tempOrders[floor] = make([]bool, 3)
	}

	err = decoder.Decode(&tempOrders) 
	if err != nil {
		return
	}
	for floor := 0; floor < len(orders); floor++ {
		orders[floor][BUTTON_CALL_UP] = orders[floor][BUTTON_CALL_UP] || tempOrders[floor][BUTTON_CALL_UP])
		orders[floor][BUTTON_CALL_DOWN] = orders[floor][BUTTON_CALL_DOWN] || tempOrders[floor][BUTTON_CALL_DOWN])
		orders[floor][BUTTON_CALL_INSIDE] = (orders[floor][BUTTON_CALL_INSIDE] || tempOrders[floor][BUTTON_CALL_INSIDE])
	}
}

func merge_matrix(dst, src [][]bool){
	for i := range src{
		if len(dst[i]) == len(src[i]){
			for j := range src[i]{
				if src[i][j] || dst[i][j]{ 
					dst[i][j] = true
				}
			}
		}
	}
}

func copy_matrix(dst, src [][]bool){
	for i := range src{
		tempList := make([]bool, len(src[i]))
		copy(tempList, src[i])
		dst[i] = make([]bool, len(src[i]))
		copy(dst[i], tempList)
	}
}

func copy_map(dst, src map[string]Elevator_t){
	for key, _ := range dst{
		delete(dst, key)
	}
	for ID, e := range src{
		dst[ID] = make_elevator(e.NumFloors, e.Position, e.Status)
		copy_matrix(dst[ID].Orders, e.Orders)
	}
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
		if elevators[elevatorList[i]].Status == stateMachine.UNKNOWN {status = "UNKNOWN				"}
		if elevators[elevatorList[i]].Status == stateMachine.MOVING_UP {status = "MOVING UP			"}
		if elevators[elevatorList[i]].Status == stateMachine.MOVING_DOWN {status = "MOVING DOWN			"}
		if elevators[elevatorList[i]].Status == stateMachine.IDLE {status = "IDLE				"}
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
