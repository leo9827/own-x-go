package _func

import (
	"fmt"
	"sync"
)

type Center struct {
	mu           sync.Mutex
	funcRegistry map[string]func(data interface{})
}

var defaultC *Center = &Center{mu: sync.Mutex{}, funcRegistry: make(map[string]func(data interface{}))}

func (c *Center) RegistryFunc(name string, f func(data interface{})) {
	c.mu.Lock()
	c.funcRegistry[name] = f
	c.mu.Unlock()
}

func recv(_ interface{}) {
	fmt.Println("this in [func recv]")
}

type Receiver struct {
	name string
}

func (r *Receiver) recv(_ interface{}) {
	fmt.Println("this in [func (r Receiver) recv], r.name: ", r.name)
}

var _globalR *Receiver = &Receiver{name: "global_receiver"}

func init() {
	defaultC.RegistryFunc("init_recv", recv)
	defaultC.RegistryFunc("init_g_r_recv", _globalR.recv)
	// You can use to declare and init an immutable object and registry to run with repeated!
	var r *Receiver = &Receiver{name: "receiver_in_init"}
	defaultC.RegistryFunc("init_r_recv", r.recv)
}
