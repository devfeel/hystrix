package hystrix

import (
	"time"
	"sync"
	"github.com/devfeel/polaris/control/count"
)

const(
	status_Hystrix = 1
	status_Alive = 2
	DefaultCheckHystrixInterval = 10 //unit is Second
	DefaultCheckAliveInterval = 60 //unit is Second
	DefaultCleanHistoryInterval = 60 * 5 //unit is Second
	DefaultMaxFailedNumber = 100
	DefaultReserveMinutes = 30
	DefaultAutoTryAliveInterval = 60 * 5 //unit is Second
	minuteTimeLayout = "200601021504"
)

type Hystrix interface{
	// Do begin do check
	Do()

	// RegisterAliveCheck register check Alive func
	RegisterAliveCheck(CheckFunc)
	// RegisterHystrixCheck register check Hystrix func
	RegisterHystrixCheck(CheckFunc)

	// RegisterOnTriggerHystrix register event func on trigger hystrix
	RegisterOnTriggerHystrix(triggerFunc TriggerFunc)
	// RegisterOnTriggerAlive register event func on trigger alive
	RegisterOnTriggerAlive(triggerFunc TriggerFunc)

	// GetID get Hystrix ID
	GetID() string
	// IsHystrix return is Hystrix status
	IsHystrix() bool
	// TriggerHystrix trigger Hystrix status
	TriggerHystrix()
	// TriggerAlive trigger Alive status
	TriggerAlive()
	// GetCounter get lasted Counter with time key
	GetCounter() count.Counter
	// GetExtendedData get extended data
	GetExtendedData() interface{}


	// SetID set Hystrix ID
	SetID(string)
	// SetCheckInterval set interval for doCheckHystric and doCheckAlive, unit is Second
	SetCheckInterval(int, int)
	// SetMaxFailed set max failed count for hystrix default counter
	SetMaxFailedNumber(int64)
	// SetExtendedData set extended data
	SetExtendedData(data interface{})
}

type CheckFunc func()bool
type TriggerFunc func(h Hystrix)

type StandHystrix struct{
	id string
	status int
	lastChangeStatusTime time.Time
	checkHystrixFunc CheckFunc
	checkHystrixInterval int
	checkAliveFunc CheckFunc
	checkAliveInterval int

	maxFailedNumber int64
	counters *sync.Map

	//extended data used on trigger event
	extendedData interface{}

	onTriggerAlive TriggerFunc
	onTriggerHystrix TriggerFunc
}


// NewHystrix create new Hystrix, config with CheckAliveFunc and checkAliveInterval, unit is Minute
func NewHystrix(checkAlive CheckFunc, checkHysrix CheckFunc) Hystrix{
	h := &StandHystrix{
		counters : new(sync.Map),
		status:status_Alive,
		lastChangeStatusTime:time.Now(),
		checkAliveFunc: checkAlive,
		checkHystrixFunc:checkHysrix,
		checkAliveInterval:DefaultCheckAliveInterval,
		checkHystrixInterval:DefaultCheckHystrixInterval,
		maxFailedNumber:DefaultMaxFailedNumber,
	}
	if h.checkHystrixFunc == nil{
		h.checkHystrixFunc = h.defaultCheckHystrix
	}
	if h.checkAliveFunc == nil{
		h.checkAliveFunc = h.defaultCheckAlive
	}
	return h
}

func (h *StandHystrix) Do(){
	go h.doCheck()
	go h.doCleanHistoryCounter()
}

func (h *StandHystrix) SetCheckInterval(hystrixInterval, aliveInterval int){
	h.checkAliveInterval = aliveInterval
	h.checkHystrixInterval = hystrixInterval
}

// SetMaxFailed set max failed count for hystrix default counter
func (h *StandHystrix) SetMaxFailedNumber(number int64){
	h.maxFailedNumber = number
}

// GetCounter get lasted Counter with time key
func (h *StandHystrix) GetCounter() count.Counter{
	key := getLastedTimeKey()
	var counter count.Counter
	loadCounter, exists := h.counters.Load(key)
	if !exists{
		counter = count.NewCounter()
		h.counters.Store(key, counter)
	}else{
		counter = loadCounter.(count.Counter)
	}
	return counter
}

func (h *StandHystrix) GetID() string{
	return h.id
}

// GetExtendedData get extended data
func (h *StandHystrix) GetExtendedData() interface{}{
	return h.extendedData
}

func (h *StandHystrix) IsHystrix() bool{
	return h.status == status_Hystrix
}

func (h *StandHystrix) RegisterAliveCheck(check CheckFunc){
	h.checkAliveFunc = check
}

func (h *StandHystrix) RegisterHystrixCheck(check CheckFunc){
	h.checkHystrixFunc = check
}

func (h *StandHystrix) RegisterOnTriggerAlive(triggerFunc TriggerFunc){
	h.onTriggerAlive = triggerFunc
}

func (h *StandHystrix) RegisterOnTriggerHystrix(triggerFunc TriggerFunc){
	h.onTriggerHystrix = triggerFunc
}

func (h *StandHystrix) TriggerHystrix(){
	h.status = status_Hystrix
	h.lastChangeStatusTime = time.Now()
	if h.onTriggerHystrix != nil{
		h.onTriggerHystrix(h)
	}
}

func (h *StandHystrix) TriggerAlive(){
	h.status = status_Alive
	h.lastChangeStatusTime = time.Now()
	if h.onTriggerAlive != nil{
		h.onTriggerAlive(h)
	}
}

func (h *StandHystrix) SetID(id string) {
	h.id = id
}

func (h *StandHystrix) SetExtendedData(data interface{}){
	h.extendedData = data
}

// doCheck do checkAlive when status is Hystrix or checkHytrix when status is Alive
func (h *StandHystrix) doCheck(){
	if h.checkAliveFunc == nil || h.checkHystrixFunc == nil {
		return
	}
	if h.IsHystrix() {
		isAlive := h.checkAliveFunc()
		if isAlive {
			h.TriggerAlive()
			h.GetCounter().Clear()
			time.AfterFunc(time.Duration(h.checkHystrixInterval)*time.Second, h.doCheck)
		} else {
			time.AfterFunc(time.Duration(h.checkAliveInterval)*time.Second, h.doCheck)
		}
	}else{
		isHystrix := h.checkHystrixFunc()
		if isHystrix{
			h.TriggerHystrix()
			time.AfterFunc(time.Duration(h.checkAliveInterval)*time.Second, h.doCheck)
		}else{
			time.AfterFunc(time.Duration(h.checkHystrixInterval)*time.Second, h.doCheck)
		}

	}
}

func (h *StandHystrix) doCleanHistoryCounter(){
	var needRemoveKey []string
	now, _ := time.Parse(minuteTimeLayout, time.Now().Format(minuteTimeLayout))
	h.counters.Range(func(k, v interface{}) bool{
		key := k.(string)
		if t, err := time.Parse(minuteTimeLayout, key); err != nil {
			needRemoveKey = append(needRemoveKey, key)
		} else {
			if now.Sub(t) > (DefaultReserveMinutes * time.Minute) {
				needRemoveKey = append(needRemoveKey, key)
			}
		}
		return true
	})
	for _, k := range needRemoveKey {
		//fmt.Println(time.Now(), "hystrix doCleanHistoryCounter remove key",k)
		h.counters.Delete(k)
	}
	time.AfterFunc(time.Duration(DefaultCleanHistoryInterval)*time.Second, h.doCleanHistoryCounter)
}

func (h *StandHystrix) defaultCheckHystrix() bool{
	count := h.GetCounter().Count()
	if count > h.maxFailedNumber{
		return true
	}else{
		return false
	}
}

func (h *StandHystrix) defaultCheckAlive() bool{
	//default check is compare retry interval, now is use DefaultAutoTryAliveInterval
	if time.Now().Sub(h.lastChangeStatusTime).Seconds() > DefaultAutoTryAliveInterval{
		return true
	}
	return false
}

func getLastedTimeKey() string{
	key :=  time.Now().Format(minuteTimeLayout)
	if time.Now().Minute() / 2 != 0{
		key = time.Now().Add(time.Duration(-1*time.Minute)).Format(minuteTimeLayout)
	}
	return key
}