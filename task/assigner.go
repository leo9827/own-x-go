package monitor

import (
	"fmt"
	"ownx/cluster"
)

type Assigner struct {
}

func NewAssigner() *Assigner {
	a := &Assigner{}
	return a
}

func (a *Assigner) Registry(name string, f func(data interface{}) error) {

}

func (a *Assigner) Start() {

}

type Allocator interface {
}

func (a *Assigner) alloc(round int) {
	// TODO 分配任务

	onlineNodes := cluster.DefaultCluster.GetAliveNodeNames()
	for _, nodeName := range onlineNodes {
		fmt.Println(nodeName)
	}
	lostNodes := cluster.DefaultCluster.GetLostNodeNames()
	for _, nodeName := range lostNodes {
		fmt.Println(nodeName)
	}

}
