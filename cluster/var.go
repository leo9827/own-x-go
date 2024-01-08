package cluster

// DefaultCluster global cluster
var DefaultCluster *Cluster

type stat = uint8

const (
	keyTargetNode = "target_node"
)

// 集群模式
const (
	modelSingle  = "single"  // 单点模式
	modelCluster = "cluster" // 集群模式
)

// 集群角色状态
const (
	statusClosed    stat = 128 // 集群已关闭
	statusLeader    stat = 1   // 领导人
	statusCandidate stat = 2   // 候选人
	statusFollower  stat = 3   // 群众
)

// 消息类型
const (
	msgOk               stat = 200 // 正常
	msgSGetVote         stat = 12  // 请求为我投票
	msgSAskLeader       stat = 13  // 询问主节点
	msgSBroadcastLeader stat = 14  // 主节点的广播消息
	msgSPublishJob      stat = 15  // 任务派发
	msgRGetVote         stat = 30  // 投票结果
	msgRAskLeader       stat = 32  // 询问主节点的应答
	msgRLeader          stat = 31  // 主节点心跳的响应
)

const (
	StateClosed    = statusClosed
	StateLeader    = statusLeader
	StateCandidate = statusCandidate
	StateFollower  = statusFollower
)
