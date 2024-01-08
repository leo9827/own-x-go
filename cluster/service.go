package cluster

import (
	"ownx/env"
	"ownx/log"
	"sync"
	"time"
)

func Create(clusterConfig *Conf, logger log.Logger) *Cluster {

	// 如果集群配置为空，初始化一个空的配置
	if clusterConfig == nil {
		clusterConfig = &Conf{}
	}

	// 校验一次集群模式，默认情况下为单节点模式
	switch clusterConfig.Model {
	case modelCluster, modelSingle:
	default:
		clusterConfig.Model = modelSingle
	}

	// 设置默认超时时间，10秒
	if clusterConfig.NodeTimeout <= 0 {
		clusterConfig.NodeTimeout = 10
	}

	// 初始化集群实例
	cluster := &Cluster{
		logger:         logger,
		configure:      clusterConfig,
		nodesAll:       &sync.Map{},
		nodesAlive:     &sync.Map{},
		taskTracker:    make(map[string]func(data interface{})),
		lostLeaderTime: time.Now(),
		events:         make(chan event, 20),
	}

	cluster.createEvent("Init")

	// 加载配置中所有节点
	cluster.loadAllNodes()

	// 标识本地节点
	cluster.findMyNode()

	// 打印本地节点信息
	if cluster.nodeMy == nil {
		cluster.logger.Error("[Cluster Service] [init] Local node not found.")
	} else {
		cluster.logger.Info("[Cluster Service] [init] finished. local node is %s - %s", cluster.nodeMy.name, cluster.nodeMy.id)
	}

	// 返回集群实例
	return cluster
}

func (cluster *Cluster) StartUp() {

	cluster.createEvent("StartUp")

	switch cluster.configure.Model {
	case modelCluster:
		if cluster.nodeMy == nil {
			// 集群模式，没有本地节点，报错
			cluster.logger.Error("[Cluster Service] [startup] failed. Local node not found. please check the configuration file.")
			return
		}
	case modelSingle:
		// 单节点模式，直接进入Leader状态
		cluster.logger.Info("[Cluster Service] [startup] ^_^ %s I'am in single model.", cluster.GetMyId())
		cluster.signLeader(cluster.nodeMy)
		return
	}

	// 全局只能启动一次
	cluster.lock.Lock()
	if cluster.status > 0 {
		cluster.lock.Unlock()
		return
	}
	cluster.status = statusFollower
	cluster.lock.Unlock()

	// 开启服务
	cluster.openService()
}

func (cluster *Cluster) Close() {

	cluster.closeOnce.Do(func() {

		cluster.logger.Info("[Cluster Service] [close] cluster node %s shutting down.", cluster.GetMyName())

		cluster.status = statusClosed
		cluster.createEvent("Close")

		// 关闭服务端端口服务
		if cluster.serviceAcceptor != nil {
			cluster.serviceAcceptor.Close()
			cluster.serviceAcceptor = nil
		}

		// 关闭事件
		if cluster.events != nil {
			close(cluster.events)
			cluster.events = nil
		}

		// 关闭与其它节点的连接
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
			// 排除自己这个节点
			if _node, ok := val.(*Node); ok && _node.id != cluster.GetMyId() {
				if _, ok := cluster.nodesAlive.Load(_node.id); !ok { //不在在线节点列表中
					nodes = append(nodes, _node.name)
				} else if _node.isTimeOut(cluster.configure.NodeTimeout) { //超时节点
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
			// 排除自己这个节点
			if _node, ok := val.(*Node); ok && _node.id != cluster.GetMyId() {
				if _, ok := cluster.nodesAlive.Load(_node.id); !ok { // 不在在线节点列表中
					ips = append(ips, _node.ip)
				} else if _node.isTimeOut(cluster.configure.NodeTimeout) { // 超时节点
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

	// 从配置中获取全部节点列表
	nodes := cluster.configure.Nodes
	if nodes == nil || len(nodes) == 0 {
		cluster.logger.Warn("[Cluster Service] [init] Can't find any node.")
		return
	}

	// 缓存
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

	cluster.logger.Info("[Cluster Service] [init] All nodes in configure: %s", ToJson(_tmp))
}

func (cluster *Cluster) findMyNode() {

	nodeDir := Replace(env.GetProgramPath(), "[/][^/]+$", "")
	nodeDir = Replace(nodeDir, ".*/", "")

	// 优先使用名称进行本地节点匹配
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

	// 如果已标识出本地节点，直接返回
	if cluster.nodeMy != nil {
		return
	}

	// 获取本机的IP地址
	myIps, err := GetIntranetIp()
	if err != nil {
		cluster.logger.Error("[Cluster Service] [init] Get local address failed. %s", err.Error())
		return
	}
	if myIps == nil || len(myIps) == 0 {
		cluster.logger.Error("[Cluster Service] [init] Any local address is not found.")
		return
	}

	cluster.logger.Info("[Cluster Service] [init] My ip list: %s, My program directory is %s", myIps, nodeDir)

	// 从所有ip中进行筛选
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

	// 如果已标识出本地节点，直接返回
	if cluster.nodeMy != nil {
		return
	}

	// 如果还没有标识出来本地节点，并且当前是单点模式，那么随机使用一个ip地址注册为本地节点
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
	cluster.lock.Lock()
	defer cluster.lock.Unlock()
	// 新的主节点任期，小于我的任期
	if newLeaderNode.getTerm() < cluster.GetMyTerm() {
		return false
	}
	// 新的主节点的任期，小于当前主节点的任期
	if cluster.nodeLeader != nil && newLeaderNode.getTerm() < cluster.nodeLeader.getTerm() {
		return false
	}
	// 主节点未变化
	if cluster.nodeLeader != nil && cluster.nodeLeader.GetId() == newLeaderNode.GetId() {
		return true
	}
	cluster.nodeLeader = newLeaderNode
	if cluster.GetMyNode().GetId() == newLeaderNode.GetId() {
		cluster.logger.Info("[Cluster Service] [ready] (term=%d) ============ I'am leader node ============", newLeaderNode.getTerm())
		cluster.status = statusLeader
		go cluster.createEvent("SignLeader")
	} else {
		cluster.logger.Info("[Cluster Service] [ready] (term=%d) ============ I'am slave node. leader is %s ============", newLeaderNode.getTerm(), newLeaderNode.GetId())
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
