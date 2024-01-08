package cluster

import (
	"ownx/nio"
	"runtime/debug"
	"sort"
	"time"
)

func (cluster *Cluster) openService() {

	if err := cluster.openServiceAcceptor(); err != nil {
		cluster.logger.Error("[openService] %s Create serviceAcceptor failed. Cause %s", cluster.nodeMy.id, err)
	}

	go cluster.holdConnection()

	time.Sleep(100 * time.Millisecond)

	go cluster.fightElection()

	go cluster.heartbeat()

	go cluster.stat()
}

func (cluster *Cluster) stat() {
	defer func() {
		if r := recover(); r != nil {
			cluster.logger.Error("[stat] Print Stat. got a error：%s\n%s", r, string(debug.Stack()))
			go cluster.stat()
		}
	}()

	for {
		leaderId := ""
		if leaderNode := cluster.GetLeaderNode(); leaderNode != nil {
			leaderId = leaderNode.GetId()
		}
		cluster.logger.Debug("[stat] %s %t %v %v %v",
			leaderId, cluster.IsLeader(), cluster.GetAliveNodeNames(), cluster.GetMyNode().voteCount, cluster.GetMyNode().voteNode)
		time.Sleep(10 * time.Minute)
	}
}

func (cluster *Cluster) openServiceAcceptor() error {

	serviceAcceptorHandler := cluster.genServiceAcceptorHandler()

	serviceAcceptor, err := nio.CreateAcceptor(cluster.nodeMy.ip, ToString(cluster.nodeMy.port), serviceAcceptorHandler, nil, false)
	if err != nil {
		return err
	}

	for i := 1; i <= 4; i++ {
		if err := serviceAcceptor.Open(); err != nil {
			if i == 4 {
				panic(err)
			}
			cluster.logger.Error("[openServiceAcceptor] %s Open serviceAcceptor failed. Cause %s", cluster.nodeMy.id, err.Error())
			time.Sleep(time.Duration(i) * 2 * time.Second)
			continue
		}
		break
	}

	cluster.serviceAcceptor = serviceAcceptor

	return nil
}

func (cluster *Cluster) holdConnection() {

	defer func() {
		if r := recover(); r != nil {
			cluster.logger.Error("[holdConnection] Cluster - Hold All Node Connection. got a error：%s\n%s", r, string(debug.Stack()))
			go cluster.holdConnection()
		}
	}()

	for {
		switch cluster.status {
		case statusClosed:
			cluster.logger.Info("[holdConnection] Cluster closed. Hold All Node Connection -> release and exit.")
			return
		default:

			cluster.nodesAll.Range(
				func(key, val interface{}) bool {

					if _node, succ := val.(*Node); succ && _node.id != cluster.nodeMy.id && _node.nodeConnector == nil {
						cluster.connectToNode(_node)
					}
					return true
				})
		}
		time.Sleep(2 * time.Second)
	}
}

func (cluster *Cluster) connectToNode(targetNode *Node) bool {

	nodeHandler := cluster.genNodeHandler(targetNode)

	nodeConnector, err := nio.CreateConnector(targetNode.ip, ToString(targetNode.port), nodeHandler, nil, false)
	if err != nil {
		cluster.logger.Error("[connectToNode] Create connector %s failed. Cause of %s", targetNode.id, err)
		return false
	}

	if err = nodeConnector.Connect(); err != nil {
		cluster.logger.Error("[connectToNode] Connect to server %s failed. Cause of %s", targetNode.id, err)
		return false
	}

	targetNode.clear()
	targetNode.nodeConnector = nodeConnector

	return true
}

func (cluster *Cluster) heartbeat() {
	defer func() {
		if r := recover(); r != nil {
			cluster.logger.Error("[heartbeat] Cluster Heartbeat. got a error：%s\n%s", r, string(debug.Stack()))
			go cluster.heartbeat()
		}
	}()

	for {

		if cluster.IsReady() {

			if cluster.IsLeader() {

				cluster.nodesAll.Range(
					func(key, val interface{}) bool {
						if node, ok := val.(*Node); ok && node.id != cluster.nodeMy.id {

							if node.nodeConnector != nil && node.isTimeOut(cluster.configure.NodeTimeout) {
								cluster.logger.Error("[heartbeat] node timeout. %s", node.id)
								node.nodeConnector.Close()
								return true
							}

							if node.nodeConnector != nil {
								msg := &NodeMsg{NodeId: cluster.GetMyId(), Term: cluster.GetMyTerm(), LeaderId: cluster.GetMyId()}
								if err := node.SendMessage(msgSBroadcastLeader, msg); err != nil {
									cluster.logger.Error("[heartbeat] Broadcast leader message error. %s", err)
								}
							}
						}
						return true
					})

			} else {
				if cluster.nodeLeader != nil && cluster.nodeLeader.isTimeOut(cluster.configure.NodeTimeout) {
					cluster.logger.Info("[heartbeat] timeout from leader. close leader connection.")
					if cluster.nodeLeader.nodeConnector != nil {
						cluster.nodeLeader.nodeConnector.Close()
					} else {
						cluster.toNextTerm()
					}

				} else if cluster.nodeLeader == nil {
					currTime := time.Now().Unix()
					lostLeaderTime := cluster.lostLeaderTime.Unix()
					if (currTime - lostLeaderTime) > 6 {
						cluster.logger.Info("[heartbeat] long time no leader node. to next term.")
						cluster.toNextTerm()
					}
				}
			}

		} else if cluster.IsClose() {
			cluster.logger.Info("[heartbeat] Cluster closed. Stop heartbeat.")
			break
		} else {
			time.Sleep(200 * time.Millisecond)
			continue
		}

		if len(cluster.GetMyNode().voteCount) > 20 {
			key := make([]int, 0)
			for k := range cluster.GetMyNode().voteCount {
				key = append(key, k)
			}
			sort.Ints(key)
			for i := 0; i < 10; i++ {
				delete(cluster.GetMyNode().voteCount, key[i])
			}
		}
		if len(cluster.GetMyNode().voteNode) > 20 {
			key := make([]int, 0)
			for k := range cluster.GetMyNode().voteNode {
				key = append(key, k)
			}
			sort.Ints(key)
			for i := 0; i < 10; i++ {
				delete(cluster.GetMyNode().voteNode, key[i])
			}
		}

		time.Sleep(2 * time.Second)
	}
}

func (cluster *Cluster) fightElection() {

	time.Sleep(time.Duration(GetRandomNum(3000)) * time.Millisecond)

	cluster.createEvent("ElectionStart")
	defer cluster.createEvent("ElectionFinish")

	switch cluster.status {
	case statusClosed, statusLeader:
		return
	}

	cluster.logger.Debug("[fightElection] init.")

	defer func() {
		if r := recover(); r != nil {
			cluster.logger.Error("[fightElection] Cluster FightElection. got a error：%s\n%s", r, string(debug.Stack()))
			go cluster.fightElection()
		}
	}()

	needNextTerm := false

	defer func() {
		if needNextTerm {
			cluster.logger.Debug("[fightElection] go to next term.")
			go cluster.fightElection()
		}
	}()

	if cluster.nodeLeader != nil {
		cluster.logger.Debug("[fightElection] leader node is ok. skip 1.")
		return
	}

	cluster.fightingLock.Lock()
	if cluster.fighting {
		cluster.fightingLock.Unlock()
		return
	}

	cluster.fighting = true
	defer func() {
		cluster.fighting = false
	}()

	cluster.fightingLock.Unlock()

	currentTerm := cluster.nodeMy.getTerm()

	currentTerm++

	cluster.logger.Debug("[fightElection] my term is %d and fight term is %d.", cluster.nodeMy.getTerm(), currentTerm)

	if !cluster.nodeMy.setTerm(currentTerm) {
		cluster.logger.Debug("[fightElection] set fight term failed. go to next term.")
		needNextTerm = true
		return
	}

	cluster.logger.Info("[fightElection] Start the fightElection (%d)", currentTerm)
	defer func() {
		cluster.logger.Info("[fightElection] Finished the fightElection (%d)", currentTerm)
	}()

	sleepTimes := 1
	for {
		if cluster.nodeLeader != nil {
			cluster.logger.Debug("[fightElection] leader node is ok. skip 2.")
			return
		}
		onlineNodesCount := len(cluster.GetAliveNodeNames())
		if onlineNodesCount > (len(cluster.GetAllNodeNames()) / 2) {
			cluster.sendAndWait(2*time.Second, msgSAskLeader, &NodeMsg{NodeId: cluster.GetMyId(), Term: currentTerm})
			break
		} else {
			if sleepTimes%10 == 0 {
				cluster.logger.Info("[fightElection] The online nodes (%d) is not enough. waiting node online.", onlineNodesCount)
			}
			sleepTimes++
		}
		time.Sleep(time.Second)
	}

	time.Sleep(200 * time.Millisecond)

	if cluster.nodeLeader != nil {
		cluster.logger.Debug("[fightElection] leader node is ok. skip 3.")
		return
	}

	switch cluster.status {
	case statusClosed:
		return
	case statusFollower:
		cluster.status = statusCandidate
	case statusCandidate:
	case statusLeader:
		return
	}

	cluster.nodeMy.clear()

	sleepTimes = 1
	for {

		if cluster.nodeLeader != nil {
			cluster.logger.Debug("[fightElection] leader node is ok. skip 4.")
			return
		}

		switch cluster.status {
		case statusCandidate:
		default:
			cluster.logger.Debug("[fightElection] current status is not candidate. go to next term.")
			needNextTerm = true
			return
		}

		if currentTerm < cluster.nodeMy.getTerm() {
			cluster.logger.Debug("[fightElection] current term is changed. go to next term.")
			needNextTerm = true
			return
		}

		onlineNodesCount := len(cluster.GetAliveNodeNames())
		if onlineNodesCount > (len(cluster.GetAllNodeNames()) / 2) {

			cluster.sendAndWait(2*time.Second, msgSGetVote, &NodeMsg{NodeId: cluster.GetMyId(), Term: cluster.GetMyTerm()})

			if cluster.nodeMy.voteToNode(currentTerm, cluster.nodeMy.GetId()) {
				cluster.nodeMy.addVoteCount(currentTerm)
			}

			break

		} else {
			if sleepTimes%10 == 0 {
				cluster.logger.Info("[fightElection] The online nodes (%d) is not enough", onlineNodesCount)
			}
			sleepTimes++
		}

		time.Sleep(time.Second)
	}

	fightDone := true

	voteCount := cluster.nodeMy.getVoteCount(currentTerm)
	cluster.logger.Info("[fightElection] voteCount：term=%d vote=%d", currentTerm, voteCount)
	if voteCount > (len(cluster.GetAllNodeNames()) / 2) {

		cluster.nodesAll.Range(
			func(key, val interface{}) bool {
				if node, ok := val.(*Node); ok && node.id != cluster.nodeMy.id {
					if currentTerm < node.getTerm() {
						fightDone = false
						return false
					}
				}
				return true
			})

		if currentTerm < cluster.GetMyNode().getTerm() {
			cluster.logger.Debug("[fightElection] current term is changed. go to next term.")
			fightDone = false
		}
	} else {
		fightDone = false
	}

	if fightDone {
		nodeMsgList := cluster.sendAndWait(
			2*time.Second,
			msgSBroadcastLeader,
			&NodeMsg{NodeId: cluster.GetMyId(), Term: currentTerm, LeaderId: cluster.GetMyId(), Flag: "1"},
		)
		if len(nodeMsgList) == 0 {
			cluster.logger.Debug("[fightElection] leader confirm failed. no any response.")
			fightDone = false
		}

		for _, nodeMsg := range nodeMsgList {
			if nodeMsg.Success == nil || *nodeMsg.Success == false {
				fightDone = false
				break
			}
		}
	}

	if !fightDone || !cluster.signLeader(cluster.nodeMy) {
		needNextTerm = true
	}
}

func (cluster *Cluster) genServiceAcceptorHandler() *nio.Handler {
	serviceAcceptorHandler := &nio.Handler{}

	serviceAcceptorHandler.OnMessageReceived = func(session *nio.Session, message *nio.Msg) {

		nodeMsg := msg2node(message.Data())

		switch message.Flag() {

		case msgSAskLeader:

			leaderId := ""
			leaderTerm := cluster.GetMyTerm()
			if leaderNode := cluster.GetLeaderNode(); leaderNode != nil && leaderNode.getTerm() >= nodeMsg.Term {
				leaderId = leaderNode.GetId()
				leaderTerm = leaderNode.getTerm()
			}

			session.Write(msgRAskLeader, &NodeMsg{NodeId: cluster.GetMyId(), Term: leaderTerm, ReqId: nodeMsg.ReqId, LeaderId: leaderId})

			if cluster.IsFighting() || cluster.GetLeaderNode() == nil {
				cluster.logger.Info("term:%d, ask msg: %s, answer leaderId=%s", cluster.GetMyTerm(), ToJson(nodeMsg), leaderId)
			}

		case msgSGetVote:

			if cluster.IsLeader() {
				cluster.releaseLeader()
			}

			chooseYou := cluster.GetMyNode().voteToNode(nodeMsg.Term, nodeMsg.NodeId)

			session.Write(msgRGetVote, &NodeMsg{NodeId: cluster.GetMyId(), Term: nodeMsg.Term, ReqId: nodeMsg.ReqId, Success: &chooseYou})

			if cluster.IsFighting() || cluster.GetLeaderNode() == nil {
				cluster.logger.Info("term:%d, get vote: %s, answer chooseYou=%t", cluster.GetMyTerm(), ToJson(nodeMsg), chooseYou)
			}

		case msgSBroadcastLeader:

			if nodeMsg.Term < cluster.GetMyTerm() {
				success := false
				session.Write(msgRLeader, &NodeMsg{NodeId: cluster.GetMyId(), Term: cluster.GetMyTerm(), ReqId: nodeMsg.ReqId, Success: &success})
				cluster.logger.Info("term:%d, broadcast:%s，reject Leader.", cluster.GetMyTerm(), ToJson(nodeMsg))
				return
			}

			currentLeader := cluster.GetLeaderNode()
			if nodeMsg.Term > cluster.GetMyTerm() && currentLeader != nil && currentLeader.GetId() != nodeMsg.LeaderId {
				cluster.releaseLeader()
				cluster.logger.Info("term:%d, broadcast:%s, release Leader.", cluster.GetMyTerm(), ToJson(nodeMsg))
			}

			cluster.GetMyNode().setTerm(nodeMsg.Term)

			if value, ok := cluster.nodesAll.Load(nodeMsg.NodeId); ok {
				cacheNode := value.(*Node)
				cacheNode.setTerm(nodeMsg.Term)
				cacheNode.updateHbTime()
			}
			if value, ok := cluster.nodesAll.Load(nodeMsg.LeaderId); ok {
				cacheNode := value.(*Node)
				cacheNode.setTerm(nodeMsg.Term)
				cacheNode.updateHbTime()
			}

			session.Write(msgOk, &NodeMsg{NodeId: cluster.GetMyId(), Term: cluster.GetMyTerm()})

			if cluster.GetLeaderNode() != nil {
				if cluster.GetLeaderNode().GetId() == nodeMsg.LeaderId && cluster.GetLeaderNode().getTerm() == nodeMsg.Term {

					if nodeMsg.Flag == "1" {
						success := true
						session.Write(msgRLeader, &NodeMsg{NodeId: cluster.GetMyId(), Term: cluster.GetMyTerm(), ReqId: nodeMsg.ReqId, Success: &success})
					}
					return
				}
			}

			if value, ok := cluster.nodesAll.Load(nodeMsg.LeaderId); ok {
				leaderNode := value.(*Node)
				success := cluster.signLeader(leaderNode)
				session.Write(msgRLeader, &NodeMsg{NodeId: cluster.GetMyId(), Term: cluster.GetMyTerm(), ReqId: nodeMsg.ReqId, Success: &success})

				cluster.logger.Info("term:%d broadcast: %s、answer Success=%t", cluster.GetMyTerm(), ToJson(nodeMsg), success)
			}

		case msgSPublishJob:

			if jobInfo := msg2job(message.Data()); jobInfo != nil {
				cluster.pushTask(jobInfo.Name, jobInfo.Data)
			} else {
				cluster.logger.Error("Server received empty job.")
			}

		default:
			cluster.logger.Error("Server received unknown message, flag: %s", message.Flag())
		}
	}

	return serviceAcceptorHandler
}

func (cluster *Cluster) genNodeHandler(targetNode *Node) *nio.Handler {

	nodeHandler := &nio.Handler{}

	nodeHandler.OnSessionConnected = func(session *nio.Session) {
		session.SetAttribute(keyTargetNode, targetNode)
		cluster.nodesAlive.Store(targetNode.GetId(), targetNode)
		cluster.logger.Info("node online: %s", targetNode.GetId())
	}

	nodeHandler.OnSessionClosed = func(session *nio.Session) {
		targetNode.nodeConnector = nil
		cluster.nodesAlive.Delete(targetNode.GetId())

		cluster.logger.Info("node offline: %s", targetNode.GetId())
		if leaderNode := cluster.GetLeaderNode(); leaderNode != nil && leaderNode.GetId() == targetNode.GetId() {
			switch cluster.status {
			case statusClosed:
				return
			default:
				cluster.logger.Info("leader %s has lost. go to next term.", targetNode.GetId())
				cluster.toNextTerm()
			}
		} else {
			onlineNodesCount := len(cluster.GetAliveNodeNames())
			if onlineNodesCount <= (len(cluster.GetAllNodeNames()) / 2) {
				switch cluster.status {
				case statusClosed:
					return
				default:
					if cluster.IsFighting() {
						return
					} else {
						cluster.logger.Info("Release leader. Cause of the online nodes (%d) is not enough", onlineNodesCount)
						cluster.toNextTerm()
					}
				}
			}
		}
	}

	nodeHandler.OnMessageReceived = func(session *nio.Session, message *nio.Msg) {
		nodeMsg := msg2node(message.Data())
		if cluster.GetLeaderNode() != nil && cluster.GetMyTerm() < nodeMsg.Term {
			cluster.releaseLeader()
			return
		}
		cluster.GetMyNode().setTerm(nodeMsg.Term)

		if value, ok := cluster.nodesAll.Load(nodeMsg.NodeId); ok {
			cacheNode := value.(*Node)
			cacheNode.setTerm(nodeMsg.Term)
			cacheNode.updateHbTime()
		}
		if value, ok := cluster.nodesAll.Load(nodeMsg.LeaderId); ok {
			cacheNode := value.(*Node)
			cacheNode.setTerm(nodeMsg.Term)
			cacheNode.updateHbTime()
		}

		switch message.Flag() {

		case msgRAskLeader:

			if cluster.IsFighting() || cluster.GetLeaderNode() == nil {
				cluster.logger.Info("term:%d, ask leader: %s", cluster.GetMyTerm(), ToJson(nodeMsg))
			}
			if nodeMsg.LeaderId != "" {
				if _, ok := cluster.nodesAlive.Load(nodeMsg.LeaderId); ok == false {
					cluster.logger.Info("leader has dead. ignore your response.")
					return
				}
			}
			if nodeMsg.LeaderId != "" {
				if value, ok := cluster.nodesAll.Load(nodeMsg.LeaderId); ok {
					cluster.signLeader(value.(*Node))
				}
			}
			if cluster.sendAndWaitChan != nil {
				func() {
					defer OnError("[panic] cluster msg chan")
					cluster.sendAndWaitChan <- nodeMsg
				}()
			}

		case msgRGetVote:

			if cluster.IsFighting() || cluster.GetLeaderNode() == nil {
				cluster.logger.Info("term:%d, get vote: %s", cluster.GetMyTerm(), ToJson(nodeMsg))
			}
			if nodeMsg.Success != nil && *nodeMsg.Success {
				cluster.GetMyNode().addVoteCount(nodeMsg.Term)
			}
			if cluster.sendAndWaitChan != nil {
				func() {
					defer OnError("[panic] cluster msg chan")
					cluster.sendAndWaitChan <- nodeMsg
				}()
			}

		case msgRLeader:

			if cluster.IsFighting() || cluster.GetLeaderNode() == nil {
				cluster.logger.Info("term:%d, master confirm heartbeat: %s", cluster.GetMyTerm(), ToJson(nodeMsg))
			}
			if nodeMsg.Success != nil && *nodeMsg.Success == false {
				cluster.logger.Debug("%s refused my leader request", ToJson(nodeMsg))
				currentLeader := cluster.GetLeaderNode()
				if currentLeader != nil && currentLeader.GetId() == cluster.GetMyId() {
					cluster.releaseLeader()
				}
			}
			if cluster.sendAndWaitChan != nil {
				func() {
					defer OnError("[panic] cluster msg chan")
					cluster.sendAndWaitChan <- nodeMsg
				}()
			}

		case msgOk:

		default:
			cluster.logger.Error("Client received unknown message: %s", message.Flag())
		}
	}

	return nodeHandler
}

func (cluster *Cluster) sendAndWait(timeout time.Duration, flag uint8, data *NodeMsg) []*NodeMsg {
	nodeMsgList := make([]*NodeMsg, 0)

	data.ReqId = Uuid()
	if timeout == 0 {
		timeout = 2 * time.Second
	}

	senderNodeMap := map[string]struct{}{}

	syncChan := make(chan *NodeMsg)
	defer func() {
		close(syncChan)
	}()
	cluster.sendAndWaitChan = syncChan

	cluster.nodesAll.Range(
		func(key, val interface{}) bool {
			if node, succ := val.(*Node); succ && node.id != cluster.GetMyNode().GetId() && node.nodeConnector != nil {
				if err := node.SendMessage(flag, data); err != nil {
					cluster.logger.Error("SendAndWait message error. %s", err)
				} else {
					senderNodeMap[node.GetId()+"."+data.ReqId] = struct{}{}
				}
			}
			return true
		})

loop:
	for {
		select {
		case nodeMsg, ok := <-cluster.sendAndWaitChan:
			if !ok {
				break loop
			}
			nodeMsgList = append(nodeMsgList, nodeMsg)
			delete(senderNodeMap, nodeMsg.NodeId+"."+nodeMsg.ReqId)
			if len(senderNodeMap) == 0 {
				break loop
			}
		case <-time.After(timeout):
			break loop
		}
	}

	cluster.sendAndWaitChan = nil

	return nodeMsgList
}

func (cluster *Cluster) toNextTerm() {
	cluster.releaseLeader()
	go cluster.fightElection()
}

func (cluster *Cluster) broadcastLeader(leaderNode *Node) {
	cluster.nodesAll.Range(
		func(key, val interface{}) bool {
			node, ok := val.(*Node)
			if ok &&
				node.id != cluster.GetMyNode().GetId() &&
				node.nodeConnector != nil &&
				node.GetId() != leaderNode.GetId() {
				if err := node.SendMessage(
					msgSBroadcastLeader,
					&NodeMsg{NodeId: cluster.GetMyId(), Term: cluster.GetMyTerm(), LeaderId: leaderNode.GetId()},
				); err != nil {
					cluster.logger.Error("BroadcastNewLeader message error. %s", err)
				}
			}
			return true
		})
}
