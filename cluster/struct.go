package cluster

import (
	"ownx/log"
	"ownx/nio"
	"sync"
	"time"
)

type Cluster struct {
	logger          log.Logger
	lock            sync.Mutex
	fightingLock    sync.Mutex
	configure       *Conf
	nodesAll        *sync.Map
	nodesAlive      *sync.Map
	nodeLeader      *Node
	lostLeaderTime  time.Time
	nodeMy          *Node
	status          stat
	fighting        bool
	sendAndWaitChan chan *NodeMsg
	serviceAcceptor *nio.SocketAcceptor
	taskTracker     map[string]func(data interface{})
	events          chan event
	closeOnce       sync.Once
}

type Conf struct {
	Model       string
	NodeTimeout int64 `yaml:"node_timeout"`
	Nodes       []*NodeConf
}

type Service interface {
	StartUp()
	Close()
	IsFighting() bool
	IsClose() bool
	IsReady() bool
	IsLeader() bool
	IsCandidate() bool
	IsFollower() bool
	GetLeaderNode() *Node
	GetMyNode() *Node
	GetMyId() string
	GetMyName() string
	GetMyTerm() int
	GetAllNodeNames() []string
	GetAliveNodeNames() []string
	GetLostNodeNames() []string
	Events() <-chan event
}

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
