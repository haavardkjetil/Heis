package driver

/*
#cgo CFLAGS: -std=c99
#cgo LDFLAGS: -lcomedi -lm
#include "io.h"
*/
import "C"
	
import (

"log"
)

//TODO: Dette burde legge i en egen type fil
type ButtonType_t int

const(
	BUTTON_CALL_UP ButtonType_t = iota
	BUTTON_CALL_DOWN 
	BUTTON_CALL_INSIDE 
)

//TODO: Kanskje dette også?
const (
	N_FLOORS = 4
	N_BUTTONS = 3
)

//TODO: OG dette?
type MotorDirection_t int

const(
	DIR_DOWN = -1
	DIR_STOP = 0
	DIR_UP = 1
)



var (
	lampChannelMatrix = [N_FLOORS][N_BUTTONS]int{
		
		{LIGHT_UP1, LIGHT_DOWN1, LIGHT_COMMAND1},
		{LIGHT_UP2, LIGHT_DOWN2, LIGHT_COMMAND2},
		{LIGHT_UP3, LIGHT_DOWN3, LIGHT_COMMAND3},
		{LIGHT_UP4, LIGHT_DOWN4, LIGHT_COMMAND4},
	}

	buttonChannelMatrix = [N_FLOORS][N_BUTTONS]int{
		{BUTTON_UP1, BUTTON_DOWN1, BUTTON_COMMAND1},
	    {BUTTON_UP2, BUTTON_DOWN2, BUTTON_COMMAND2},
	    {BUTTON_UP3, BUTTON_DOWN3, BUTTON_COMMAND3},
	    {BUTTON_UP4, BUTTON_DOWN4, BUTTON_COMMAND4},
	}
)

func Init() bool {
	if (int(C.io_init()) == 0) {
		return false
	} 
	for etg := 0; etg < N_FLOORS; etg++ {
		if (etg != 0) {
			SetButtonLamp(BUTTON_CALL_DOWN, etg, 0)
		}	
		if (etg != N_FLOORS - 1) {
			SetButtonLamp(BUTTON_CALL_UP, etg, 0)
		}
		SetButtonLamp(BUTTON_CALL_INSIDE, etg, 0)
	}
	SetStopLamp(0)
	SetDoorLamp(0)
	SetFloorIndicator(0)
	return true
}

func SetMotorDirection(dir MotorDirection_t) {
	if dir == 0{
		C.io_write_analog(MOTOR,0)
	}else if (dir > 0){
		C.io_clear_bit(MOTORDIR)
		C.io_write_analog(MOTOR,2800)
	}else if (dir < 0){
		C.io_set_bit(MOTORDIR);
		C.io_write_analog(MOTOR,2800)
	}
}


func GetFloorSensorSignal() int{
	if(int(C.io_read_bit(C.int(SENSOR_FLOOR1))) == 1){
		return 0
	}else if (int(C.io_read_bit(C.int(SENSOR_FLOOR2))) == 1) {
		return 1
	}else if (int(C.io_read_bit(C.int(SENSOR_FLOOR3))) == 1) {
		return 2
	}else if (int(C.io_read_bit(C.int(SENSOR_FLOOR4))) == 1) {
		return 3
	}
	return -1
}

func GetButtonSignal(button ButtonType_t floor int) bool {
	if floor < 0 || floor >= N_FLOORS {
		log.Fatal( "Invalid floor number")
	}
	if ((button == BUTTON_CALL_UP && floor == N_FLOORS - 1) || (button == BUTTON_CALL_DOWN && floor == 0)|| !(button == BUTTON_CALL_UP || button == BUTTON_CALL_DOWN || button == BUTTON_CALL_INSIDE)) {
		log.Fatal( "Invalid combination of floor and button")
	}

	if(int(C.io_read_bit(C.int(buttonChannelMatrix[floor][button]))) == 1) {
		return true
	}else {
		return false
	}
}

func SetFloorIndicator(floor int) {
	if floor < 0 || floor >= N_FLOORS {
		log.Fatal( "Invalid floor number")
	}
	switch floor {

	case 0:
		C.io_clear_bit(LIGHT_FLOOR_IND1)
		C.io_clear_bit(LIGHT_FLOOR_IND2)

	case 1:
		C.io_clear_bit(LIGHT_FLOOR_IND1)
		C.io_set_bit(LIGHT_FLOOR_IND2)

	case 2:
		C.io_set_bit(LIGHT_FLOOR_IND1)
		C.io_clear_bit(LIGHT_FLOOR_IND2)

	case 3:
		C.io_set_bit(LIGHT_FLOOR_IND1)
		C.io_set_bit(LIGHT_FLOOR_IND2)
	
	default:
	}

}

func SetButtonLamp(button ButtonType_t floor int, value int) {
	if floor < 0 || floor >= N_FLOORS {
		log.Fatal( "Invalid floor number")
	}
	if ((button == BUTTON_CALL_UP && floor == N_FLOORS - 1) || (button == BUTTON_CALL_DOWN && floor == 0)|| !(button == BUTTON_CALL_UP || button == BUTTON_CALL_DOWN || button == BUTTON_CALL_INSIDE)) {
		log.Fatal( "Invalid combination of floor and button")
	}

	if(value == 1) {
		C.io_set_bit(C.int(lampChannelMatrix[floor][button]))
	}else {
		C.io_clear_bit(C.int(lampChannelMatrix[floor][button]))
	}

}

func GetStopSignal() bool {
	return ( int(C.io_read_bit(STOP)) == 1)
}

func SetStopLamp(value int) {
	if(value == 1){
		C.io_set_bit(LIGHT_STOP)
	}else if value == 0 {
		C.io_clear_bit(LIGHT_STOP)
	}
}

func SetDoorLamp(value int) {
	if (value == 1) {
		C.io_set_bit(LIGHT_DOOR_OPEN)
	}else {
		C.io_clear_bit(LIGHT_DOOR_OPEN)
	}
}

func GetObstructionSignal() bool {
	return ( int(C.io_read_bit(OBSTRUCTION)) == 1)
}


