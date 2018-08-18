# devfeel/hystrix
A simple and easy go hystrix framework.


### Example
~~~ golang
package main

import (
	"fmt"
	"time"
	"github.com/devfeel/hystrix"
)

const MaxFailedNum = 10

func main(){
	fmt.Println("devfeel/hystrix example")
	curHystrix := hystrix.NewHystrix(checkAlive, nil)
	curHystrix.SetID("test hystrix")
	curHystrix.SetMaxFailedNumber(MaxFailedNum)
	curHystrix.RegisterOnTriggerAlive(onTriggerAlive)
	curHystrix.RegisterOnTriggerHystrix(onTriggerHystrix)
	curHystrix.Do()

	//do err
	go func(){
		time.Sleep(time.Second * 10)
		curHystrix.GetCounter().Inc(10)
		fmt.Println(getTimeString(), "happen 10 failed!")
		time.Sleep(time.Second * 10)
		curHystrix.GetCounter().Inc(10)
		fmt.Println(getTimeString(), "happen 10 failed! Bigger than MaxFailedNum, so will trigger hystrix.")
	}()


	time.Sleep(time.Hour)
}

func checkAlive() bool{
	fmt.Println(getTimeString(), "Do Alive Check! It will trigger alive.")
	return true
}

func onTriggerAlive(h hystrix.Hystrix){
	fmt.Println(getTimeString(), h.GetID(), "trigger alive!")
}

func onTriggerHystrix(h hystrix.Hystrix){
	fmt.Println(getTimeString(), h.GetID(), "trigger hystrix!")
}

func getTimeString() string{
	return time.Now().Format("2006-01-02 15:04:")
}
~~~