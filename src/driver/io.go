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

type ButtonType_t int
const(
	BUTTON_CALL_UP ButtonType_t = iota
	BUTTON_CALL_DOWN 
	BUTTON_CALL_INSIDE 
)

const N_FLOORS = 4


type MotorDirection_t int
const(
	DIR_DOWN MotorDirection_t = iota
	DIR_STOP 
	DIR_UP
)

var (
	lampChannelMatrix = [N_FLOORS][3]int{
		
		{LIGHT_UP1, LIGHT_DOWN1, LIGHT_COMMAND1},
		{LIGHT_UP2, LIGHT_DOWN2, LIGHT_COMMAND2},
		{LIGHT_UP3, LIGHT_DOWN3, LIGHT_COMMAND3},
		{LIGHT_UP4, LIGHT_DOWN4, LIGHT_COMMAND4},
	}

	buttonChannelMatrix = [N_FLOORS][3]int{
		{BUTTON_UP1, BUTTON_DOWN1, BUTTON_COMMAND1},
	    {BUTTON_UP2, BUTTON_DOWN2, BUTTON_COMMAND2},
	    {BUTTON_UP3, BUTTON_DOWN3, BUTTON_COMMAND3},
	    {BUTTON_UP4, BUTTON_DOWN4, BUTTON_COMMAND4},
	}
)
// func Run() bool {
// 	if(!init()){
// 		return 0
// 	}
// 	for{

// 	}
// }


//TODO: LEgg til sjekk om den er initialisert
func Init() bool {
	if (int(C.io_init()) == 0) {
		return false
	} 
	for etg := 0; etg < N_FLOORS; etg++ {
		if (etg != 0) {
			Set_button_lamp(BUTTON_CALL_DOWN, etg, 0)
		}	
		if (etg != N_FLOORS - 1) {
			Set_button_lamp(BUTTON_CALL_UP, etg, 0)
		}
		Set_button_lamp(BUTTON_CALL_INSIDE, etg, 0)
	}
	Set_stop_lamp(0)
	Set_door_lamp(0)
	Set_floor_indicator(0)
	return true
}

func Set_motor_direction(dir MotorDirection_t) {
	if dir == DIR_STOP{
		C.io_write_analog(MOTOR,0)
	}else if (dir == DIR_UP){
		C.io_clear_bit(MOTORDIR)
		C.io_write_analog(MOTOR,2800)
	}else if (dir == DIR_DOWN){
		C.io_set_bit(MOTORDIR);
		C.io_write_analog(MOTOR,2800)
	}else {
		log.Fatal( "Invalid argument; motor direction")
	}

}


func Get_floor_sensor_signal() int{
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

func Get_button_signal(button ButtonType_t, floor int) bool {
	if floor < 0 || floor >= N_FLOORS {
		log.Fatal( "Invalid floor number")
	}
	if ((button == BUTTON_CALL_UP && floor == N_FLOORS - 1) || (button == BUTTON_CALL_DOWN && floor == 0)){
		log.Fatal( "Invalid combination of floor and button")
	}

	if !(button == BUTTON_CALL_UP || button == BUTTON_CALL_DOWN || button == BUTTON_CALL_INSIDE) {
		log.Fatal( "Invalid argument; button type")
	}

	if(int(C.io_read_bit(C.int(buttonChannelMatrix[floor][button]))) == 1) {
		return true
	}else {
		return false
	}
}

func Set_floor_indicator(floor int) {
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

func Set_button_lamp(button ButtonType_t floor int, value int) {
	if floor < 0 || floor >= N_FLOORS {
		log.Fatal( "Invalid floor number")
	}
	if ((button == BUTTON_CALL_UP && floor == N_FLOORS - 1) || (button == BUTTON_CALL_DOWN && floor == 0)){
		log.Fatal( "Invalid combination of floor and button")
	}

	if !(button == BUTTON_CALL_UP || button == BUTTON_CALL_DOWN || button == BUTTON_CALL_INSIDE) {
		log.Fatal( "Invalid argument; button type")
	}

	if(value == 1) {
		C.io_set_bit(C.int(lampChannelMatrix[floor][button]))
	}else {
		C.io_clear_bit(C.int(lampChannelMatrix[floor][button]))
	}

}

func Get_stop_signal() bool {
	return ( int(C.io_read_bit(STOP)) == 1)
}

func Set_stop_lamp(value int) {
	if(value == 1){
		C.io_set_bit(LIGHT_STOP)
	}else if value == 0 {
		C.io_clear_bit(LIGHT_STOP)
	}
}

func Set_door_lamp(value int) {
	if (value == 1) {
		C.io_set_bit(LIGHT_DOOR_OPEN)
	}else {
		C.io_clear_bit(LIGHT_DOOR_OPEN)
	}
}

func Get_obstruction_signal() bool {
	return ( int(C.io_read_bit(OBSTRUCTION)) == 1)
}


