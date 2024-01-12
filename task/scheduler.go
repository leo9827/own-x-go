package monitor

import (
	"fmt"
	"sync"
)

type Scheduler struct {
	mu        sync.Mutex
	done      chan interface{}
	round     int
	roundDone chan interface{}
	roundCall map[string]func(round int)
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		mu:        sync.Mutex{},
		done:      make(chan interface{}),
		roundDone: make(chan interface{}),
		round:     0,
	}
}

func (s *Scheduler) Start() {
	if e := s.prevCheck(); e != nil {
		fmt.Println("prevCheck error ", e)
		return
	}
	for {
		select {
		case <-s.roundDone:
			s.startNewRound()
		case <-s.done:
			return
		}
	}
}

func (s *Scheduler) RoundDone() {
	if e := s.prevCheck(); e != nil {
		fmt.Println("prevCheck error ", e)
		return
	}
	s.roundDone <- struct{}{}
}

func (s *Scheduler) startNewRound() {
	if e := s.prevCheck(); e != nil {
		fmt.Println("prevCheck error ", e)
		return
	}
	s.mu.Lock()
	s.round++
	s.mu.Unlock()
	for name, f := range s.roundCall {
		fmt.Println("call ", name, ", round: ", s.round)
		f(s.round)
	}
}

func (s *Scheduler) CallRegistry(name string, call func(round int)) {
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
