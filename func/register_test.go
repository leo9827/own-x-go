package _func

import (
	"fmt"
	"testing"
)

func TestRegistryFunc(t *testing.T) {
	// Magic and Effective !
	for name, f := range defaultC.funcRegistry {
		fmt.Println(name)
		f(struct{}{})
	}
	_globalR.name = "modify in test"
	for name, f := range defaultC.funcRegistry {
		fmt.Println(name)
		f(struct{}{})
	}
}
