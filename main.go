package main

import(
"network"
"driver"
"fmt"
"log"
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
        log.Fatal("Unable to initialize elevator hardware!")
    }

    println("Press STOP button to stop elevator and exit program.")
    driver.SetMotorDirection(driver.DIR_DOWN)

    for {
        // Change direction when we reach top/bottom floor
    	if driver.GetFloorSensorSignal() == 3 {
            driver.SetMotorDirection(driver.DIR_DOWN)
        } else if driver.GetFloorSensorSignal() == 0 {
            driver.SetMotorDirection(driver.DIR_UP)
        }

        // Stop elevator and exit program if the stop button is pressed
        if driver.GetStopSignal() {
        	driver.SetMotorDirection(driver.DIR_STOP)
            break
        }
    }
}