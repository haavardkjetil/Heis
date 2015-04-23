package driver

import(
"log"
"time"
)

type ButtonType_t int
const(
	BUTTON_CALL_UP ButtonType_t = iota
	BUTTON_CALL_DOWN 
	BUTTON_CALL_INSIDE 
)

type Button_t struct {
	Type ButtonType_t
	Floor int
}

type ButtonLampUpdate_t struct {
	Button Button_t
	Value bool
}

const N_FLOORS = 4

type MotorDirection_t int
const(
	DIR_DOWN MotorDirection_t = iota
	DIR_STOP 
	DIR_UP
)

func Run( buttonLamp_c chan ButtonLampUpdate_t,
	      buttonSensor_c chan Button_t,
	      floorSensor_c chan int,
	      motorDir_c chan MotorDirection_t,
	      doorLamp_c chan bool) {
	
	if !init_IO(){
		log.Fatal("Could not initialize I/O driver")
	}

	go poll_floor_sensor( floorSensor_c )
	go poll_button_signal( buttonSensor_c )

	for{
		select{
			case dir := <- motorDir_c:
				set_motor_direction( dir )
			case value := <- doorLamp_c:
				set_door_lamp( value )
			case buttonLampUpdate := <- buttonLamp_c:
				set_button_lamp( buttonLampUpdate.Button, buttonLampUpdate.Value )
			default:
				time.Sleep(time.Millisecond)
		}

	}

}

func poll_floor_sensor(floorChan chan int) int{
	currentSensorSignal := -1
	for{
		for i := 0; i<N_FLOORS; i++ {
			if currentSensorSignal != get_floor_sensor_signal(){
				currentSensorSignal = get_floor_sensor_signal()
				floorChan <- currentSensorSignal
				if currentSensorSignal != -1 { set_floor_indicator( currentSensorSignal ) }
			}
		}
		time.Sleep(time.Millisecond)
	}
}

func poll_button_signal(buttonChan chan Button_t){
	var currentSensorSignal = [N_FLOORS][3]bool{}

	for{
		for i := 0; i<(N_FLOORS-1); i++{
			if currentSensorSignal[i][0] != get_button_signal(Button_t{BUTTON_CALL_UP,i}) {
				currentSensorSignal[i][0] = !currentSensorSignal[i][0]
				if currentSensorSignal[i][0] {buttonChan <- Button_t{BUTTON_CALL_UP,i}}
			}
		}
		for i := 1; i<N_FLOORS; i++{
			if currentSensorSignal[i][1] != get_button_signal(Button_t{BUTTON_CALL_DOWN,i}) {
				currentSensorSignal[i][1] = !currentSensorSignal[i][1]
				if currentSensorSignal[i][1] {buttonChan <- Button_t{BUTTON_CALL_DOWN,i}}
			}
		}
		for i := 0; i<N_FLOORS; i++{
			if currentSensorSignal[i][2] != get_button_signal(Button_t{BUTTON_CALL_INSIDE,i}) {
				currentSensorSignal[i][2] = !currentSensorSignal[i][2]
				if currentSensorSignal[i][2] {buttonChan <- Button_t{BUTTON_CALL_INSIDE,i}}
			}
		}
		time.Sleep(time.Millisecond)	
	}
}


