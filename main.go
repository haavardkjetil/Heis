package main

import(
"network"
"fmt"
)

func main() {
	testIO()
}


func testNetwork(){
	transmitChannel := make(chan network.Packet,5)
	go network.ReceiveMessage(transmitChannel)
	for {
		message := <- transmitChannel
		fmt.Println(message.Type)
		fmt.Println(message.Postition)
	}
}


func testIO() {
	// Initialize hardware
    if !driver.Init() {
        log.Fatal("Unable to initialize elevator hardware!\n")
    }

    println("Press STOP button to stop elevator and exit program.\n")

    driver.SetMotorDirection(-1)

    for {
        // Change direction when we reach top/bottom floor
    	if driver.GetFloorSensorSignal() == 3 {
            driver.SetMotorDirection(-1)
        } else if driver.GetFloorSensorSignal() == 0 {
            driver.SetMotorDirection(1)
        }

        // Stop elevator and exit program if the stop button is pressed
        if driver.GetStopSignal() {
        	driver.SetMotorDirection(0)
            break
        }
    }
}