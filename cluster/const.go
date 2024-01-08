package cluster

type stat = uint8

const (
	keyTargetNode = "target_node"
)

const (
	modelCluster = "cluster"
	modelSingle  = "single"
)

const (
	statusClosed    stat = 128
	statusLeader    stat = 1
	statusCandidate stat = 2
	statusFollower  stat = 3
)

type msg = uint8

const (
	msgOk msg = 200
)

const (
	msgSGetVote         msg = 12
	msgSAskLeader       msg = 13
	msgSBroadcastLeader msg = 14
	msgSPublishJob      msg = 15
)

const (
	msgRGetVote   msg = 30
	msgRAskLeader msg = 32
	msgRLeader    msg = 31
)
