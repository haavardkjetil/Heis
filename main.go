package main

import(
"network"
"fmt"
)

func main(){
	transmitChannel := make(chan network.Packet,5)
	go network.ReceiveMessage(transmitChannel)
	for {
		message := <- transmitChannel
		fmt.Println(message.Type)
		fmt.Println(message.Postition)
	}
}
