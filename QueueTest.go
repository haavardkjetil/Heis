package main


import (
.	"queueManager"
"network"
 //"runtime"
.	"fmt"
	"time"
	"math/rand"
)

func update_random_order(orderChan chan Order_t){
	for{
		time.Sleep(3000*time.Millisecond)
		orderOperation := ADD
		if rand.Intn(4) == 1 {
			//print("DELETE order ")
			orderOperation = DELETE
		}else{
			//print("ADD order ")
		}
		buttonCall := BUTTON_CALL_UP
		if rand.Intn(3) == 1 {
			//print("DOWNWARDS ")
			buttonCall = BUTTON_CALL_DOWN
		}else if rand.Intn(10) == 2{
			//print("from INSIDE ")
			buttonCall = BUTTON_CALL_INSIDE
		}else{
			//print("UPWARDS ")
		}
		floor := rand.Intn(4)
		//print("on floor ", floor)
		orderUpdate := Order_t{
			orderOperation, buttonCall, floor,
		}
		//Println("")
		orderChan <- orderUpdate
	}
}


func main() {

	//runtime.GOMAXPROCS(4)
	
	localIP := network.Global_get_ip()
	numFloors := 4
	networkReceive := make(chan UpdatePacket_t)
	networkTransmit := make(chan UpdatePacket_t)

	go network.Run(4,networkTransmit,networkReceive)

	orderChan := make(chan Order_t)
	positionChan := make(chan int)
	quitChan := make(chan int)

	go Run(localIP, numFloors, networkReceive, networkTransmit, orderChan, positionChan, quitChan)
	positionChan <- 2
	
	update_random_order(orderChan)

	quitChan <- 1
	Println("Done")
}