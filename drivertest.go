













               floorIndicatorChan_push, motorDirChan_push, doorLampChan_push)
    buttonLampChan_push := make(chan driver.ButtonLampUpdate_t,10)
    buttonSensorChan_pull := make(chan driver.Button_t,10)
    doorLampChan_push := make(chan bool,10)
    floorIndicatorChan_push := make(chan int,10)
    floorSensorChan_pull := make(chan int,10)
    go driver.Run(buttonLampChan_push, buttonSensorChan_pull, floorSensorChan_pull, 
    motorDirChan_push := make(chan driver.MotorDirection_t,10)
    }
                motorDirChan_push <- driver.DIR_DOWN
                motorDirChan_push <- driver.DIR_STOP
                motorDirChan_push <- driver.DIR_UP
            if button.Floor == 3 {
            if floor == 0 {
            if floor == 3 {
            }
            }
            }
        case button := <- buttonSensorChan_pull:
        case floor := <- floorSensorChan_pull:
        select{
        }
            
                    buttonLampChan_push <- driver.ButtonLampUpdate_t{ driver.Button_t{driver.BUTTON_CALL_UP,floor} , false }
                    buttonLampChan_push <- driver.ButtonLampUpdate_t{ driver.Button_t{driver.BUTTON_CALL_UP,floor} , true }
                buttonLampChan_push <- driver.ButtonLampUpdate_t{ driver.Button_t{driver.BUTTON_CALL_INSIDE,0} , false }
                buttonLampChan_push <- driver.ButtonLampUpdate_t{ driver.Button_t{driver.BUTTON_CALL_INSIDE,0} , true }
                doorLampChan_push <- false
                doorLampChan_push <- true
                floorIndicatorChan_push <- floor
                if (floor != 3 && lastdir == driver.DIR_DOWN)  {
                if (floor != 3 && lastdir == driver.DIR_UP)  {
                lastdir = driver.DIR_DOWN
                lastdir = driver.DIR_UP
                motorDirChan_push <- driver.DIR_STOP
                motorDirChan_push <- driver.DIR_UP
                motorDirChan_push <- lastdir
                time.Sleep(2*time.Second)
                }       
                }                   
            if (floor != -1) {
            if button.Floor == 2 {
            if floor == 2 {
            time.Sleep(time.Millisecond)
            }
            }
            }
        default:
    //runtime.GOMAXPROCS(4)
    for {
    lastdir := driver.DIR_DOWN
    motorDirChan_push <- driver.DIR_DOWN
    println("Press NED3 to stop elevator and exit program.")
"driver"
"time"
)
//"runtime"
func main(){
import(
package main
}