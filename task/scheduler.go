package monitor

import (
	"fmt"
	"sync"
	"time"
)

type Scheduler struct {
	name          string
	mu            sync.Mutex
	done          chan interface{}
	round         int64
	roundDoneFlag bool
	roundDone     chan interface{}
	roundCall     map[string]func(round int64)
}

func newScheduler(name string) *Scheduler { // init keep private
	return &Scheduler{
		name:      name,
		mu:        sync.Mutex{},
		done:      make(chan interface{}),
		roundDone: make(chan interface{}, 1), // default len is 1 for start
		round:     time.Now().Unix(),         // using timestamp for round
		roundCall: make(map[string]func(round int64)),
	}
}

var DefaultMonitorScheduler = newScheduler("DefaultMonitorScheduler")

func (s *Scheduler) Name() string {
	return s.name
}

func (s *Scheduler) Start() {
	if e := s.prevCheck(); e != nil {
		fmt.Println("prevCheck error ", e)
		return
	}

	for {
		select {
		case <-s.done:
			fmt.Println("Scheduler ", s.name, " is shutting down")
			return
		default:
		}
	}
}

func (s *Scheduler) MasterCall() {
	for {
		select {
		case <-s.roundDone:
			s.startNewRound()
		default:
		}
	}
}

func (s *Scheduler) SlaveCall() {
	fmt.Println("SlaveCall is not implement")
}

func (s *Scheduler) Waitting() {
	fmt.Println("Waitting is not implement")
}

func (s *Scheduler) Stop() {
	fmt.Println("Stop is not implement")
}

func (s *Scheduler) RoundDone(round int64) {
	if e := s.prevCheck(); e != nil {
		fmt.Println("prevCheck error ", e)
		return
	}
	s.mu.Lock()
	if !s.roundDoneFlag && round == s.round {
		s.roundDoneFlag = true
		s.roundDone <- struct{}{}
	}
	s.mu.Unlock()
}

func (s *Scheduler) startNewRound() {
	if e := s.prevCheck(); e != nil {
		fmt.Println("prevCheck error ", e)
		return
	}
	s.mu.Lock()
	s.round = time.Now().Unix()
	s.mu.Unlock()
	for name, f := range s.roundCall {
		fmt.Println("call ", name, ", round: ", s.round)
		f(s.round)
	}
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
	if s.roundDone == nil {
		return fmt.Errorf("scheduler roundDone not init")
	}
	if s.roundCall == nil {
		return fmt.Errorf("scheduler roundCall not init")
	}

	return nil
}
