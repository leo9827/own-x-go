package cluster

import (
	"ownx/cluster/env"
	"ownx/log"
	"ownx/nio"
	"sync"
	"time"
)

var DefaultCluster *Cluster

type Cluster struct {
	logger          log.Logger
	configure       *Conf
	mu              sync.Mutex
	fighting        bool
	fightingMu      sync.Mutex
	nodesAll        *sync.Map
	nodesAlive      *sync.Map
	nodeMy          *Node
	nodeLeader      *Node
	lostLeaderTime  time.Time
	status          stat
	sendAndWaitChan chan *NodeMsg
	serviceAcceptor *nio.SocketAcceptor
	taskRegistry    map[string]func(data interface{})
	events          chan event
	closeOnce       sync.Once
}

type Conf struct {
	Model       string
	NodeTimeout int64 `json:"node_timeout" yaml:"node_timeout"`
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

func Create(clusterConfig *Conf, logger log.Logger) *Cluster {

	if clusterConfig == nil {
		clusterConfig = &Conf{}
	}

	switch clusterConfig.Model {
	case modelCluster, modelSingle:
	default:
		clusterConfig.Model = modelSingle
	}

	if clusterConfig.NodeTimeout <= 0 {
		clusterConfig.NodeTimeout = 10
	}

	cluster := &Cluster{
		logger:         logger,
		configure:      clusterConfig,
		nodesAll:       &sync.Map{},
		nodesAlive:     &sync.Map{},
		taskRegistry:   make(map[string]func(data interface{})),
		lostLeaderTime: time.Now(),
		events:         make(chan event, 20),
	}

	cluster.createEvent("Init")

	cluster.loadAllNodes()

	cluster.findMyNode()

	if cluster.nodeMy == nil {
		cluster.logger.Error("Cluster Create failed, local node not found.")
	} else {
		cluster.logger.Info("Cluster Create finished, local node is [%s, %s]", cluster.nodeMy.name, cluster.nodeMy.id)
	}

	return cluster
}

func (cluster *Cluster) StartUp() {
	cluster.createEvent("StartUp")

	switch cluster.configure.Model {
	case modelCluster:
		if cluster.nodeMy == nil {

			cluster.logger.Error("[StartUp] failed. Local node not found. please check the configuration file.")
			return
		}
	case modelSingle:

		cluster.logger.Info("[StartUp] %s is in single model!", cluster.GetMyId())
		cluster.signLeader(cluster.nodeMy)
		return
	}

	cluster.mu.Lock()
	if cluster.status > 0 {
		cluster.mu.Unlock()
		return
	}
	cluster.status = statusFollower
	cluster.mu.Unlock()

	cluster.openService()
}

func (cluster *Cluster) Close() {

	cluster.closeOnce.Do(func() {

		cluster.logger.Info("[Close] cluster node %s is shutting down.", cluster.GetMyName())

		cluster.status = statusClosed
		cluster.createEvent("Close")

		if cluster.serviceAcceptor != nil {
			cluster.serviceAcceptor.Close()
			cluster.serviceAcceptor = nil
		}

		if cluster.events != nil {
			close(cluster.events)
			cluster.events = nil
		}

		cluster.nodesAll.Range(
			func(key, val interface{}) bool {
				if _node, ok := val.(*Node); ok {
					if _node.nodeConnector != nil {
						_node.nodeConnector.Close()
						_node.nodeConnector = nil
					}
				}
				return true
			})
	})
}

func (cluster *Cluster) IsClose() bool {
	return cluster.status == statusClosed
}

func (cluster *Cluster) IsReady() bool {
	switch cluster.status {
	case statusLeader, statusCandidate, statusFollower:
		return !cluster.fighting
	default:
		return false
	}
}

func (cluster *Cluster) IsFighting() bool {
	return cluster.fighting
}

func (cluster *Cluster) IsLeader() bool {
	return cluster.status == statusLeader
}

func (cluster *Cluster) IsCandidate() bool {
	return cluster.status == statusCandidate
}

func (cluster *Cluster) IsFollower() bool {
	return cluster.status == statusFollower
}

func (cluster *Cluster) GetLeaderNode() *Node {
	return cluster.nodeLeader
}

func (cluster *Cluster) GetMyNode() *Node {
	return cluster.nodeMy
}

func (cluster *Cluster) GetMyId() string {
	if cluster.nodeMy != nil {
		return cluster.nodeMy.GetId()
	}
	return ""
}

func (cluster *Cluster) GetMyIp() string {
	if cluster.nodeMy != nil {
		return cluster.nodeMy.ip
	}
	return ""
}

func (cluster *Cluster) GetMyName() string {
	if cluster.nodeMy != nil {
		return cluster.nodeMy.GetName()
	}
	return ""
}

func (cluster *Cluster) GetMyTerm() int {
	if cluster.nodeMy != nil {
		return cluster.nodeMy.getTerm()
	}
	return 0
}

func (cluster *Cluster) GetAllNodeNames() []string {
	nodes := make([]string, 0)
	cluster.nodesAll.Range(
		func(key, val interface{}) bool {
			if _node, ok := val.(*Node); ok {
				nodes = append(nodes, _node.name)
			}
			return true
		})
	return nodes
}

func (cluster *Cluster) GetAliveNodeNames() []string {
	nodes := make([]string, 0)
	nodes = append(nodes, cluster.nodeMy.name)
	cluster.nodesAlive.Range(
		func(key, val interface{}) bool {
			if _node, ok := val.(*Node); ok {
				nodes = append(nodes, _node.name)
			}
			return true
		})
	return nodes
}

func (cluster *Cluster) GetAliveNodeIps() []string {
	ips := make([]string, 0)
	ips = append(ips, cluster.nodeMy.ip)
	cluster.nodesAlive.Range(
		func(key, val interface{}) bool {
			if _node, ok := val.(*Node); ok {
				ips = append(ips, _node.ip)
			}
			return true
		})
	if len(ips) == 1 {
		if ips[0] == "" {
			return []string{}
		}
	}
	return ips
}

func (cluster *Cluster) GetLostNodeNames() []string {
	nodes := make([]string, 0)
	cluster.nodesAll.Range(
		func(key, val interface{}) bool {

			if _node, ok := val.(*Node); ok && _node.id != cluster.GetMyId() {
				if _, ok := cluster.nodesAlive.Load(_node.id); !ok {
					nodes = append(nodes, _node.name)
				} else if _node.isTimeOut(cluster.configure.NodeTimeout) {
					nodes = append(nodes, _node.name)
				}
			}
			return true
		})
	return nodes
}

func (cluster *Cluster) GetLostNodeIps() []string {
	ips := make([]string, 0)
	cluster.nodesAll.Range(
		func(key, val interface{}) bool {

			if _node, ok := val.(*Node); ok && _node.id != cluster.GetMyId() {
				if _, ok := cluster.nodesAlive.Load(_node.id); !ok {
					ips = append(ips, _node.ip)
				} else if _node.isTimeOut(cluster.configure.NodeTimeout) {
					ips = append(ips, _node.ip)
				}
			}
			return true
		})
	return ips
}

func (cluster *Cluster) Events() <-chan event {
	return cluster.events
}

func (cluster *Cluster) loadAllNodes() {

	nodes := cluster.configure.Nodes
	if nodes == nil || len(nodes) == 0 {
		cluster.logger.Warn("Can't find any node.")
		return
	}

	_tmp := make([]string, 0)
	for _, node := range nodes {
		n := &Node{
			id:   node.Ip + ":" + ToString(node.Port),
			name: node.Name,
			ip:   node.Ip,
			port: node.Port,
		}
		if n.name == "" {
			n.name = n.ip
		}
		_tmp = append(_tmp, n.id)
		cluster.nodesAll.Store(n.id, n)
	}

	cluster.logger.Info("All nodes in configure: %s", ToJson(_tmp))
}

func (cluster *Cluster) findMyNode() {

	nodeDir := Replace(env.GetProgramPath(), "[/][^/]+$", "")
	nodeDir = Replace(nodeDir, ".*/", "")

	if nodeDir != "" {
		nodeDir = "node_" + nodeDir
		cluster.nodesAll.Range(
			func(key, val interface{}) bool {
				if _node, ok := val.(*Node); ok && _node.name == nodeDir {
					cluster.nodeMy = _node
					return false
				}
				return true
			})
	}

	if cluster.nodeMy != nil {
		return
	}

	myIps, err := GetIntranetIp()
	if err != nil {
		cluster.logger.Error("Get local address failed. %s", err.Error())
		return
	}
	if myIps == nil || len(myIps) == 0 {
		cluster.logger.Error("Any local address is not found.")
		return
	}

	cluster.logger.Info("ip list: %s, program directory is %s", myIps, nodeDir)

	for _, ip := range myIps {
		cluster.nodesAll.Range(
			func(key, val interface{}) bool {
				if _node, ok := val.(*Node); ok && _node.ip == ip {
					cluster.nodeMy = _node
					return false
				}
				return true
			})
	}

	if cluster.nodeMy != nil {
		return
	}

	if cluster.nodeMy == nil && cluster.configure.Model == modelSingle {
		hip := env.GetHostIp()
		if hip == "" {
			hip = myIps[0]
		}
		cluster.nodeMy = &Node{
			name: hip, ip: hip,
		}
	}
}

func (cluster *Cluster) signLeader(newLeaderNode *Node) bool {
	cluster.mu.Lock()
	defer cluster.mu.Unlock()

	if newLeaderNode.getTerm() < cluster.GetMyTerm() {
		return false
	}

	if cluster.nodeLeader != nil && newLeaderNode.getTerm() < cluster.nodeLeader.getTerm() {
		return false
	}

	if cluster.nodeLeader != nil && cluster.nodeLeader.GetId() == newLeaderNode.GetId() {
		return true
	}
	cluster.nodeLeader = newLeaderNode
	if cluster.GetMyNode().GetId() == newLeaderNode.GetId() {
		cluster.logger.Info("current term:%d, current node [%s] is leader", newLeaderNode.getTerm(), newLeaderNode.GetId)
		cluster.status = statusLeader
		go cluster.createEvent("SignLeader")
	} else {
		cluster.logger.Info("current term:%d, current node is slave node. leader is %", newLeaderNode.getTerm(), newLeaderNode.GetId())
		cluster.status = statusFollower
		go cluster.createEvent("SignFollower")
	}
	return true
}

func (cluster *Cluster) releaseLeader() {
	cluster.nodeLeader = nil
	cluster.lostLeaderTime = time.Now()
	cluster.status = statusFollower
	go cluster.createEvent("ReleaseLeader")
}

func (cluster *Cluster) createEvent(name string) {
	if cluster.status == statusClosed {
		return
	}
	select {
	case cluster.events <- &stateEvent{name, cluster.status}:
	default:
	}
	cluster.updateMetrics()
}
