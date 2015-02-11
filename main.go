package main

import(
"network"
"driver"
//"fmt"
"log"
)

func main() {
	testNetwork()
}


func testNetwork(){
    network.Run(4)


	// transmitChannel := make(chan network.Packet,5)
	// go network.ReceiveMessage(transmitChannel)
	// for {
	// 	message := <- transmitChannel
	// 	fmt.Println(message.Type)
	// 	fmt.Println(message.Postition)
	// }
}

func testIO() {
	// Initialize hardware
    if !driver.Init() {
        log.Fatal("Unable to initialize elevator hardware!")
    }

    println("Press STOP button to stop elevator and exit program.")
    driver.Set_motor_direction(driver.DIR_DOWN)

    for {
        // Change direction when we reach top/bottom floor
    	if driver.Get_floor_sensor_signal() == 3 {
            driver.Set_motor_direction(driver.DIR_DOWN)
        } else if driver.Get_floor_sensor_signal() == 0 {
            driver.Set_motor_direction(driver.DIR_UP)
        }

        // Stop elevator and exit program if the stop button is pressed
        if driver.Get_stop_signal() {
        	driver.Set_motor_direction(driver.DIR_STOP)
            break
        }
    }
}