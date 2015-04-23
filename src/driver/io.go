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

func init_IO() bool {
	if (int(C.io_init()) == 0) {
		return false
	} 
	set_stop_lamp(0)
	set_door_lamp(false)
	set_floor_indicator(0)
	return true
}

func set_motor_direction(dir MotorDirection_t) {
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

func set_floor_indicator(floor int) {
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

func set_button_lamp(button Button_t, value bool) {
	if button.Floor < 0 || button.Floor >= N_FLOORS {
		return
	}
	if ((button.Type == BUTTON_CALL_UP && button.Floor == N_FLOORS - 1) || (button.Type == BUTTON_CALL_DOWN && button.Floor == 0)){
		return
	}
	if !(button.Type == BUTTON_CALL_UP || button.Type == BUTTON_CALL_DOWN || button.Type == BUTTON_CALL_INSIDE) {
		return
	}
	if value {
		C.io_set_bit(C.int(lampChannelMatrix[button.Floor][button.Type]))
	}else {
		C.io_clear_bit(C.int(lampChannelMatrix[button.Floor][button.Type]))
	}
}

func set_door_lamp(value bool) {
	if value {
		C.io_set_bit(LIGHT_DOOR_OPEN)
	}else {
		C.io_clear_bit(LIGHT_DOOR_OPEN)
	}
}

func set_stop_lamp(value int) {
	if(value == 1){
		C.io_set_bit(LIGHT_STOP)
	}else if value == 0 {
		C.io_clear_bit(LIGHT_STOP)
	}
}

func get_floor_sensor_signal() int{
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

func get_button_signal(button Button_t) bool {
	if button.Floor < 0 || button.Floor >= N_FLOORS {
		return false
	}
	if ((button.Type == BUTTON_CALL_UP && button.Floor == N_FLOORS - 1) || (button.Type == BUTTON_CALL_DOWN && button.Floor == 0)){
		return false
	}

	if(int(C.io_read_bit(C.int(buttonChannelMatrix[button.Floor][button.Type]))) == 1) {
		return true
	}else {
		return false
	}
}

func get_stop_signal() bool {

	return ( int(C.io_read_bit(STOP)) == 1)
}

func get_obstruction_signal() bool {

	return ( int(C.io_read_bit(OBSTRUCTION)) == 1)
}






