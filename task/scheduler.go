package task

import (
	"fmt"
	"sync"
	"time"
)

type Scheduler struct {
	name          string
	mu            sync.Mutex
	isRunning     bool
	done          chan interface{}
	round         map[string]int64
	roundDoneFlag map[string]bool
	roundDoneChan chan string
	roundCall     map[string]func(round int64)
}

func newScheduler(name string) *Scheduler { // init keep private
	return &Scheduler{
		name:          name,
		mu:            sync.Mutex{},
		done:          make(chan interface{}),
		round:         make(map[string]int64), // using timestamp for round
		roundCall:     make(map[string]func(round int64)),
		roundDoneFlag: make(map[string]bool),
		roundDoneChan: make(chan string),
	}
}

var DefaultScheduler = newScheduler("DefaultScheduler")

func (s *Scheduler) Name() string {
	return s.name
}

func (s *Scheduler) Start() { // 会在注册到 DefaultScheduler 之后自动被调用进行启动
	if e := s.prevCheck(); e != nil {
		fmt.Println("prevCheck error ", e)
		return
	}
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return
	} else {
		s.isRunning = true
		s.mu.Unlock()
	}
	go s.startAll()

	for {
		select {
		case name := <-s.roundDoneChan:
			s.startNewRound(name)
		case <-s.done:
			fmt.Println("Scheduler ", s.name, " is shutting down")
			return
		}
	}
}

func (s *Scheduler) MasterCall() {
	//fmt.Println("SlaveCall is not implement")
}

func (s *Scheduler) SlaveCall() {
	//fmt.Println("SlaveCall is not implement")
}

func (s *Scheduler) Waitting() {
	//fmt.Println("Waitting is not implement")
}

func (s *Scheduler) Stop() {
	s.done <- struct{}{}
}

func (s *Scheduler) RoundDone(name string, round int64) {
	if e := s.prevCheck(); e != nil {
		fmt.Println("prevCheck error ", e)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.roundDoneFlag[name]; !exists {
		return
	}
	if _, exists := s.round[name]; !exists {
		return
	}
	if !s.roundDoneFlag[name] && s.round[name] == round {
		s.roundDoneFlag[name] = true
		s.roundDoneChan <- name
	}
}

func (s *Scheduler) startAll() {
	if e := s.prevCheck(); e != nil {
		fmt.Println("prevCheck error ", e)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for name, f := range s.roundCall {
		round := time.Now().Unix()
		s.round[name] = round
		f(round)
		fmt.Println("call ", name, ", round: ", s.round)
	}
}

func (s *Scheduler) startNewRound(name string) {
	if e := s.prevCheck(); e != nil {
		fmt.Println("prevCheck error ", e)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	f, exists := s.roundCall[name]
	if !exists {
		return
	}
	round := time.Now().Unix()
	s.round[name] = round
	s.roundDoneFlag[name] = false
	f(round)
	fmt.Println("call ", name, ", round: ", s.round)
}

func (s *Scheduler) RoundCallRegistry(name string, call func(round int64)) {
	if e := s.prevCheck(); e != nil {
		fmt.Println("prevCheck error ", e)
		return
	}
	s.mu.Lock()
	s.roundCall[name] = call
	s.mu.Unlock()
}

func (s *Scheduler) prevCheck() error {
	if s.done == nil {
		return fmt.Errorf("scheduler done not init")
	}
	if s.roundDoneChan == nil {
		return fmt.Errorf("scheduler roundDoneChan not init")
	}
	if s.roundCall == nil {
		return fmt.Errorf("scheduler roundCall not init")
	}

	return nil
}
