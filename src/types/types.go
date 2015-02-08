package types

type (

	ButtonType_t int
	MotorDirection_t int
)

const(
	N_FLOORS = 4
	N_BUTTONS = 3

	BUTTON_CALL_UP ButtonType_t = iota
	BUTTON_CALL_DOWN 
	BUTTON_CALL_INSIDE 

	DIR_DOWN MotorDirection_t = iota
	DIR_STOP
	DIR_UP
)
