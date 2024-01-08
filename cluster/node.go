package cluster

import (
	"errors"
	"fmt"
	"ownx/nio"
	"sync"
	"time"
)

type Node struct {
	voteLock      sync.Mutex
	term          int
	id            string
	name          string
	voteCount     map[int]int
	voteNode      map[int]string
	ip            string
	port          int
	nodeConnector *nio.SocketConnector
	hbTime        time.Time
}

type NodeConf struct {
	Name string
	Ip   string
	Port int
}

type NodeMsg struct {
	NodeId   string
	Term     int
	LeaderId string `json:",omitempty"`
	ReqId    string `json:",omitempty"`
	Success  *bool  `json:",omitempty"`
	Flag     string `json:",omitempty"`
}

type NodeService interface {
	GetId() string
	GetName() string
	SendMessage(flag uint8, data interface{}) error
}

func (n *Node) SendMessage(flag uint8, data interface{}) error {
	if n.nodeConnector == nil {
		return errors.New(fmt.Sprintf("Connection for cluster node %s is not ready.", n.id))
	}
	if err := n.nodeConnector.Write(flag, data); err != nil {
		return err
	}
	return nil
}

func (n *Node) clear() {
	n.hbTime = time.Time{}
}

func (n *Node) GetId() string {
	return n.id
}

func (n *Node) GetName() string {
	return n.name
}

func (n *Node) GetHost() string {
	return n.ip
}

func (n *Node) isTimeOut(timeout int64) bool {
	if !n.hbTime.IsZero() {
		currTime := time.Now().Unix()
		lastTime := n.hbTime.Unix()
		if (currTime - lastTime) > timeout {
			return true
		}
	}
	return false
}

func (n *Node) addVoteCount(term int) {
	n.voteLock.Lock()
	defer n.voteLock.Unlock()
	if n.voteCount == nil {
		n.voteCount = map[int]int{}
	}
	n.voteCount[term] = n.voteCount[term] + 1
}

func (n *Node) getVoteCount(term int) int {
	n.voteLock.Lock()
	defer n.voteLock.Unlock()
	if n.voteCount == nil {
		n.voteCount = map[int]int{}
	}
	return n.voteCount[term]
}

func (n *Node) voteToNode(yourTerm int, yourNodeId string) bool {
	n.voteLock.Lock()
	defer n.voteLock.Unlock()
	if n.voteNode == nil {
		n.voteNode = map[int]string{}
	}
	if voteNode := n.voteNode[yourTerm]; voteNode == "" {
		if yourTerm >= n.term {
			n.voteNode[yourTerm] = yourNodeId
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}

func (n *Node) getVoteToNode(term int) string {
	n.voteLock.Lock()
	defer n.voteLock.Unlock()
	if n.voteNode == nil {
		n.voteNode = map[int]string{}
	}
	return n.voteNode[term]
}

func (n *Node) getTerm() int {
	n.voteLock.Lock()
	defer n.voteLock.Unlock()
	return n.term
}

func (n *Node) setTerm(newTerm int) bool {
	n.voteLock.Lock()
	defer n.voteLock.Unlock()
	if newTerm <= n.term {
		return false
	} else {
		n.term = newTerm
		return true
	}
}

func (n *Node) updateHbTime() {
	n.hbTime = time.Now()
}
