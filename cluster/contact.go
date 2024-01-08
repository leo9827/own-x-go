package cluster

import (
	"ownx/nio"
	"runtime/debug"
	"sort"
	"time"
)

// 开启集群服务
func (cluster *Cluster) openService() {

	// 打开服务端口
	if err := cluster.openServiceAcceptor(); err != nil {
		cluster.logger.Error("[Cluster Service] [startup] %s Create serviceAcceptor failed. Cause %s", cluster.nodeMy.id, err)
	}

	// 保持所有节点的连接
	go cluster.holdConnection()

	// 稍等片刻
	time.Sleep(100 * time.Millisecond)

	// 开始任期内选举
	go cluster.fightElection()

	// 心跳服务
	go cluster.heartbeat()

	// 状态打印
	go cluster.stat()
}

// 打包状态
func (cluster *Cluster) stat() {

	// panic之后再回来继续
	defer func() {
		if r := recover(); r != nil {
			cluster.logger.Error("[Cluster Service] [stat] Print Stat. 发生未知错误：%s\n%s", r, string(debug.Stack()))
			go cluster.stat()
		}
	}()

	for {
		leaderId := ""
		if leaderNode := cluster.GetLeaderNode(); leaderNode != nil {
			leaderId = leaderNode.GetId()
		}
		cluster.logger.Debug("[Cluster Service] [stat] %s %t %v %v %v",
			leaderId, cluster.IsLeader(), cluster.GetAliveNodeNames(), cluster.GetMyNode().voteCount, cluster.GetMyNode().voteNode)
		time.Sleep(10 * time.Minute)
	}
}

// 开启本地节点服务端口
// 所有其他节点都要连接到此服务
func (cluster *Cluster) openServiceAcceptor() error {

	// 创建本地节点服务Handler
	serviceAcceptorHandler := cluster.genServiceAccecptorHandler()

	// 创建服务端，收到的消息异步处理
	serviceAcceptor, err := nio.CreateAcceptor(cluster.nodeMy.ip, ToString(cluster.nodeMy.port), serviceAcceptorHandler, nil, false)
	if err != nil {
		return err
	}

	// 打开服务端口
	for i := 1; i <= 4; i++ {
		if err := serviceAcceptor.Open(); err != nil {
			if i == 4 {
				panic(err)
			}
			cluster.logger.Error("[Cluster Service] [startup] %s Open serviceAcceptor failed. Cause %s", cluster.nodeMy.id, err.Error())
			time.Sleep(time.Duration(i) * 2 * time.Second)
			continue
		}
		break
	}

	cluster.serviceAcceptor = serviceAcceptor

	return nil
}

// 保持所有节点的连接，不管发生任何异常，保持连接的Func都要一直循环运行下去
func (cluster *Cluster) holdConnection() {

	// panic之后再回来继续
	defer func() {
		if r := recover(); r != nil {
			cluster.logger.Error("[Cluster Service] [connection] Cluster - Hold All Node Connection. 发生未知错误：%s\n%s", r, string(debug.Stack()))
			go cluster.holdConnection()
		}
	}()

	// 连接目标节点
	for {
		switch cluster.status {
		case statusClosed:
			cluster.logger.Info("[Cluster Service] [close] Cluster closed. Hold All Node Connection -> release and exit.")
			return
		default:
			// 遍历所有节点，建立连接
			cluster.nodesAll.Range(
				func(key, val interface{}) bool {
					// 排除自己这个节点
					if _node, succ := val.(*Node); succ && _node.id != cluster.nodeMy.id && _node.nodeConnector == nil {
						cluster.connectToNode(_node)
					}
					return true
				})
		}
		time.Sleep(2 * time.Second)
	}
}

// 连接目标节点
func (cluster *Cluster) connectToNode(targetNode *Node) bool {

	// 为目标节点连接设置一个handler
	nodeHandler := cluster.genNodeHandler(targetNode)

	// 连接器
	nodeConnector, err := nio.CreateConnector(targetNode.ip, ToString(targetNode.port), nodeHandler, nil, false)
	if err != nil {
		cluster.logger.Error("[Cluster Service] [connection] Create connector %s failed. Cause of %s", targetNode.id, err)
		return false
	}

	// 打开连接
	if err := nodeConnector.Connect(); err != nil {
		cluster.logger.Error("[Cluster Service] [connection] Connect to server %s failed. Cause of %s", targetNode.id, err)
		return false
	}

	// 连接成功后， 将连接缓存到node里
	targetNode.clear()
	targetNode.nodeConnector = nodeConnector

	// 连接成功
	return true
}

// 心跳服务
func (cluster *Cluster) heartbeat() {

	// panic后再次进入
	defer func() {
		if r := recover(); r != nil {
			cluster.logger.Error("[Cluster Service] [heartbeat] Cluster Heartbeat. 发生未知错误：%s\n%s", r, string(debug.Stack()))
			go cluster.heartbeat()
		}
	}()

	// 每隔n秒钟检查或发送心跳
	for {

		// 集群已就绪
		if cluster.IsReady() {

			// 我是主节点
			if cluster.IsLeader() {

				// 给所有节点（排除自己）发送主节点心跳
				cluster.nodesAll.Range(
					func(key, val interface{}) bool {
						if _node, succ := val.(*Node); succ && _node.id != cluster.nodeMy.id {
							// 超时节点
							if _node.nodeConnector != nil && _node.isTimeOut(cluster.configure.NodeTimeout) {
								cluster.logger.Error("[Cluster Service] [heartbeat] node timeout. %s", _node.id)
								_node.nodeConnector.Close()
								return true
							}
							// 正常节点，发送心跳
							if _node.nodeConnector != nil {
								msg := &NodeMsg{NodeId: cluster.GetMyId(), Term: cluster.GetMyTerm(), LeaderId: cluster.GetMyId()}
								if err := _node.SendMessage(msgSBroadcastLeader, msg); err != nil {
									cluster.logger.Error("[Cluster Service] [heartbeat] Broadcast leader message error. %s", err)
								}
							}
						}
						return true
					})

				// 我是从节点
			} else {

				// 规定时间内没有收到主节点心跳，开始下一界任期选举
				if cluster.nodeLeader != nil && cluster.nodeLeader.isTimeOut(cluster.configure.NodeTimeout) {
					cluster.logger.Info("[Cluster Service] [heartbeat] timeout from leader. close leader connection.")
					if cluster.nodeLeader.nodeConnector != nil {
						cluster.nodeLeader.nodeConnector.Close()
					} else {
						cluster.toNextTerm()
					}

					// 如果6秒钟内没有产生产节点，开始下一界任期选举
				} else if cluster.nodeLeader == nil {
					currTime := time.Now().Unix()
					lostLeaderTime := cluster.lostLeaderTime.Unix()
					if (currTime - lostLeaderTime) > 6 {
						cluster.logger.Info("[Cluster Service] [heartbeat] long time no leader node. to next term.")
						cluster.toNextTerm()
					}
				}
			}

			// 集群已关闭
		} else if cluster.IsClose() {

			cluster.logger.Info("[Cluster Service] [close] Cluster closed. Stop heartbeat.")
			break

			// 非正常状态，短暂休眠
		} else {
			time.Sleep(200 * time.Millisecond)
			continue
		}

		// 过期数据清理
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

// 开始任期内选举
// 选举过程中任何时刻，都有可能收到来自主节点的广播，如果标识出主了，尽可能退出选举流程。
func (cluster *Cluster) fightElection() {

	// 随机休眠一下就开始
	time.Sleep(time.Duration(GetRandomNum(3000)) * time.Millisecond)

	cluster.createEvent("ElectionStart")
	defer cluster.createEvent("ElectionFinish")

	// 当前状态校验，集群已关闭、或已经有主了，就可以退出选举了
	switch cluster.status {
	case statusClosed, statusLeader:
		return
	}

	cluster.logger.Debug("[Cluster Service] [fight] init.")

	// panic后再次进入
	defer func() {
		if r := recover(); r != nil {
			cluster.logger.Error("[Cluster Service] [fight] Cluster FightElection. 发生未知错误：%s\n%s", r, string(debug.Stack()))
			go cluster.fightElection()
		}
	}()

	// 是否需要进入下一界任期选举
	needNextTerm := false

	// 当方法退出时，检查是否需要递归进入下一次选举
	defer func() {
		if needNextTerm {
			cluster.logger.Debug("[Cluster Service] [fight] go to next term.")
			go cluster.fightElection()
		}
	}()

	// 已经有主了
	if cluster.nodeLeader != nil {
		cluster.logger.Debug("[Cluster Service] [fight] leader node is ok. skip 1.")
		return
	}

	// =====================================================
	// 第一部分开始：初始阶段
	// 一次只能执行一个选举任务
	// 独占
	cluster.fightingLock.Lock()
	if cluster.fighting {
		cluster.fightingLock.Unlock()
		return
	}

	// 标识为选举中，当方法退出后，标识为非选举状态
	cluster.fighting = true
	defer func() {
		cluster.fighting = false
	}()

	// 释放独占
	// 选举方法不阻塞，防止当前有选举任务进行中，再有其它未知情况又来选举，后面会卡住很多选举任务，当前选举结束后，随后对这些卡住的任务无法控制
	// 当前有选举任务时，再有调用选举方法时，立即拒绝，由调用方进行check或重试等处理
	cluster.fightingLock.Unlock()

	// 拿到当前任期，后面需要实时检查当前任期是否有改变
	// 保存当前任期，后面在选举过程中，如果任期变了，可以通过当前缓存的任期检测出来
	currentTerm := cluster.nodeMy.getTerm()

	// 进入下一界任期
	currentTerm++

	cluster.logger.Debug("[Cluster Service] [fight] my term is %d and fight term is %d.", cluster.nodeMy.getTerm(), currentTerm)

	if !cluster.nodeMy.setTerm(currentTerm) {
		cluster.logger.Debug("[Cluster Service] [fight] set fight term failed. go to next term.")
		needNextTerm = true
		return
	}

	cluster.logger.Info("[Cluster Service] [fight] Start the fightElection (%d)", currentTerm)
	defer func() {
		cluster.logger.Info("[Cluster Service] [fight] Finished the fightElection (%d)", currentTerm)
	}()

	// 选举之前巡视一遍看看现在有没有主节点
	// 如果有主节点，在收到消息应答的时候立即就处理了，不必在此处理
	// 只有在线节点数超过总数的一半时，才开始询问
	sleepTimes := 1
	for {
		if cluster.nodeLeader != nil {
			cluster.logger.Debug("[Cluster Service] [fight] leader node is ok. skip 2.")
			return
		}
		onlineNodesCount := len(cluster.GetAliveNodeNames())
		if onlineNodesCount > (len(cluster.GetAllNodeNames()) / 2) {
			cluster.sendAndWait(2*time.Second, msgSAskLeader, &NodeMsg{NodeId: cluster.GetMyId(), Term: currentTerm})
			break
		} else {
			if sleepTimes%10 == 0 {
				cluster.logger.Info("[Cluster Service] [fight] The online nodes (%d) is not enough. waiting node online.", onlineNodesCount)
			}
			sleepTimes++
		}
		time.Sleep(time.Second)
	}

	// 稍等一下，防止还没有等到任何询问的应答，就执行下去了
	time.Sleep(200 * time.Millisecond)

	// 询问过程中有可能问到主，或者收到了主节点的广播，都会标出主节点
	if cluster.nodeLeader != nil {
		cluster.logger.Debug("[Cluster Service] [fight] leader node is ok. skip 3.")
		return
	}

	// 第一部分结束：初始阶段
	// =====================================================

	// =====================================================
	// 第二部分开始：准备

	// 当前状态校验
	switch cluster.status {
	case statusClosed:
		return // 已经关闭了
	case statusFollower:
		cluster.status = statusCandidate // 提升自己为候选人
	case statusCandidate:
	case statusLeader:
		return // 已经是领导人了
	}

	// 清空
	cluster.nodeMy.clear()

	// 第二部分结束：准备
	// =====================================================

	// =====================================================
	// 第三部分开始：拉票

	// 如果当前活着的节点数量超过集群一半，那么有希望选举成功
	// 如果当前节点数量小于集群一半，那么休息片刻再次统计当前活着节点数量，直到满足条件再进行选举
	sleepTimes = 1
	for {

		// 判断当前是否有主了
		if cluster.nodeLeader != nil {
			cluster.logger.Debug("[Cluster Service] [fight] leader node is ok. skip 4.")
			return
		}

		// 当前状态校验，只有当前还是候选状态，才可进行下一步的选举
		switch cluster.status {
		case statusCandidate:
		default:
			cluster.logger.Debug("[Cluster Service] [fight] current status is not candidate. go to next term.")
			needNextTerm = true
			return
		}

		// 当前任期校验，如果当前任期变了，就不需要继续选举了
		if currentTerm < cluster.nodeMy.getTerm() {
			cluster.logger.Debug("[Cluster Service] [fight] current term is changed. go to next term.")
			needNextTerm = true
			return
		}

		// 在线节点数量是否大于总节点数量的一半
		onlineNodesCount := len(cluster.GetAliveNodeNames())
		if onlineNodesCount > (len(cluster.GetAllNodeNames()) / 2) {

			// 开始拉票
			// 拉票应答在消息接收之后立即就结算了， 不必在此处理
			cluster.sendAndWait(2*time.Second, msgSGetVote, &NodeMsg{NodeId: cluster.GetMyId(), Term: cluster.GetMyTerm()})

			// 尝试将自己的票投给自己
			if cluster.nodeMy.voteToNode(currentTerm, cluster.nodeMy.GetId()) {
				cluster.nodeMy.addVoteCount(currentTerm)
			}

			// 退出循环
			break

		} else {
			if sleepTimes%10 == 0 {
				cluster.logger.Info("[Cluster Service] [fight] The online nodes (%d) is not enough", onlineNodesCount)
			}
			sleepTimes++
		}

		time.Sleep(time.Second)
	}
	// 第三部分结束：拉票
	// =====================================================

	// 第四部分开始：算票
	// =====================================================

	// 是否能成为主节点
	ok := true

	// 如果本次票数小于总节点数的一半，则不能成为主节点
	voteCount := cluster.nodeMy.getVoteCount(currentTerm)
	cluster.logger.Info("[Cluster Service] [fight] 票数统计：term=%d vote=%d", currentTerm, voteCount)
	if voteCount > (len(cluster.GetAllNodeNames()) / 2) {

		// 如果当前任期在集群中不处于第一位，则不能成为主节点
		cluster.nodesAll.Range(
			func(key, val interface{}) bool {
				if _node, succ := val.(*Node); succ && _node.id != cluster.nodeMy.id {
					if currentTerm < _node.getTerm() {
						ok = false
						return false
					}
				}
				return true
			})

		// 如果当前任期改变了，则不能成为主节点
		if currentTerm < cluster.GetMyNode().getTerm() {
			cluster.logger.Debug("[Cluster Service] [fight] current term is changed. go to next term.")
			ok = false
		}
	} else {
		ok = false
	}

	// 成为主节点之前，先通知其他节点，检查其他节点是否同意
	// 解决极端情况下我成为了任期1的主，而其他有一个脑裂的节点可能成为了任期2的主
	if ok {

		// 广播自己为主节点
		nodeMsgList := cluster.sendAndWait(2*time.Second, msgSBroadcastLeader, &NodeMsg{NodeId: cluster.GetMyId(), Term: currentTerm, LeaderId: cluster.GetMyId(), Flag: "1"})

		// 没人应答
		if len(nodeMsgList) == 0 {
			cluster.logger.Debug("[Cluster Service] [fight] leader confirm failed. no any response.")
			ok = false
		}

		// 只要有一个节点不同意，那么此次选举结束
		for _, nodeMsg := range nodeMsgList {
			if nodeMsg.Success == nil || *nodeMsg.Success == false {
				ok = false
				break
			}
		}
	}

	// 如果可以成为主节点，偿试标识自己为主节点，如果标识失败，休息一下再次进入选举流程
	if !ok || !cluster.signLeader(cluster.nodeMy) {
		needNextTerm = true
	}

	// 第四部分结束：算票
	// =====================================================
}

// 创建本地节点服务Handler，作为服务端主要处理消息逻辑的地方
func (cluster *Cluster) genServiceAccecptorHandler() *nio.Handler {

	// server端的IO处理器
	serviceAcceptorHandler := &nio.Handler{}

	// 接收到消息
	serviceAcceptorHandler.OnMessageReceived = func(session *nio.Session, message *nio.Msg) {

		// client端的消息内容
		nodeMsg := msg2node(message.Data())

		// 消息类型
		switch message.Flag() {

		case msgSAskLeader: // 询问主节点

			// 获取当前是否有主节点，如果没有，则发送空即可
			leaderId := ""
			leaderTerm := cluster.GetMyTerm()
			if leaderNode := cluster.GetLeaderNode(); leaderNode != nil && leaderNode.getTerm() >= nodeMsg.Term {
				leaderId = leaderNode.GetId()
				leaderTerm = leaderNode.getTerm()
			}

			// 回应当前引用的主节点ID给对方
			// 如果有主节点，应答的时候给对方主节点的任期
			// 如果没有主节点，应答的时候给对方我的任期
			session.Write(msgRAskLeader, &NodeMsg{NodeId: cluster.GetMyId(), Term: leaderTerm, ReqId: nodeMsg.ReqId, LeaderId: leaderId})

			if cluster.IsFighting() || cluster.GetLeaderNode() == nil {
				cluster.logger.Info("[Cluster Service] 任期%d 对方来询问主节点 %s、应答：leaderId=%s", cluster.GetMyTerm(), ToJson(nodeMsg), leaderId)
			}

		case msgSGetVote: // 请求投票

			// 说明当前在选举中，如果自己是主节点，释放自己
			if cluster.IsLeader() {
				cluster.releaseLeader()
			}

			// 一个任期只能投一票，偿试给对方投一票
			chooseYou := cluster.GetMyNode().voteToNode(nodeMsg.Term, nodeMsg.NodeId)

			// 回应投票结果，来的时候要的是什么任期，返回去还是什么任期
			session.Write(msgRGetVote, &NodeMsg{NodeId: cluster.GetMyId(), Term: nodeMsg.Term, ReqId: nodeMsg.ReqId, Success: &chooseYou})

			if cluster.IsFighting() || cluster.GetLeaderNode() == nil {
				cluster.logger.Info("[Cluster Service] 任期%d 对方过来请求投票 %s、应答：chooseYou=%t", cluster.GetMyTerm(), ToJson(nodeMsg), chooseYou)
			}

		case msgSBroadcastLeader: // 广播主节点的消息，有可能来自主节点，也有可能来自即将成为主节点的节点

			// 如果主节点任期，小于我的任期，直接拒绝
			if nodeMsg.Term < cluster.GetMyTerm() {
				success := false
				session.Write(msgRLeader, &NodeMsg{NodeId: cluster.GetMyId(), Term: cluster.GetMyTerm(), ReqId: nodeMsg.ReqId, Success: &success})
				cluster.logger.Info("[Cluster Service] 任期%d 对方%s任期小，拒绝对方成为Leader.", cluster.GetMyTerm(), ToJson(nodeMsg))
				return
			}

			// 如果主节点的任期大于我，并且我已经标识的Leader不是你，那么释放当前的Leader
			currentLeader := cluster.GetLeaderNode()
			if nodeMsg.Term > cluster.GetMyTerm() && currentLeader != nil && currentLeader.GetId() != nodeMsg.LeaderId {
				cluster.releaseLeader()
				cluster.logger.Info("[Cluster Service] 任期%d 对方%s任期大，释放当前的Leader.", cluster.GetMyTerm(), ToJson(nodeMsg))
			}

			// 同步任期
			// 收到消息后，如果自己的任期比对方小，直接更新自己的任期，保持与对方一致
			// 由于本地缓存了一份所有节点的信息，所以同时需要更新对方节点在本地缓存中的任期，尽量保持数据一致

			// 如果对方任期比我大，更新自己的任期
			cluster.GetMyNode().setTerm(nodeMsg.Term)

			// 更新本地缓存：对方节点的信息
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

			// 如果没有变化，直接返回
			if cluster.GetLeaderNode() != nil {
				if cluster.GetLeaderNode().GetId() == nodeMsg.LeaderId && cluster.GetLeaderNode().getTerm() == nodeMsg.Term {
					// 对方过来确认他是否能成为主节点，由于我指向的主节点已经是你了，所以直接同意即可
					if nodeMsg.Flag == "1" {
						success := true
						session.Write(msgRLeader, &NodeMsg{NodeId: cluster.GetMyId(), Term: cluster.GetMyTerm(), ReqId: nodeMsg.ReqId, Success: &success})
					}
					return
				}
			}

			// 偿试将自己的主节点引用指向对方，并告诉对方接受或拒绝
			if value, ok := cluster.nodesAll.Load(nodeMsg.LeaderId); ok {
				leaderNode := value.(*Node)
				success := cluster.signLeader(leaderNode)
				session.Write(msgRLeader, &NodeMsg{NodeId: cluster.GetMyId(), Term: cluster.GetMyTerm(), ReqId: nodeMsg.ReqId, Success: &success})

				cluster.logger.Info("[Cluster Service] 任期%d 对方广播了主节点信息 %s、是否接受：Success=%t", cluster.GetMyTerm(), ToJson(nodeMsg), success)
			}

		case msgSPublishJob: // 任务派发

			// 收到任务派发，找到任务对应的TaskTracker，将任务交给他执行
			if jobInfo := msg2job(message.Data()); jobInfo != nil {
				cluster.pushTask(jobInfo.Name, jobInfo.Data)
			} else {
				cluster.logger.Error("[Cluster Service] [job] received empty job data.")
			}

		default:
			cluster.logger.Error("[Cluster Service] [message] Server received unknown message: %s", message.Flag())
		}
	}

	return serviceAcceptorHandler
}

// 创建节点服务Handler，作为客户端主要处理消息逻辑的地方
// node表示对方节点
func (cluster *Cluster) genNodeHandler(targetNode *Node) *nio.Handler {

	// client端的IO处理
	nodeHandler := &nio.Handler{}

	// 连接成功事件
	nodeHandler.OnSessionConnected = func(session *nio.Session) {

		// 将目标节点放入会话中缓存
		session.SetAttribute(keyTargetNode, targetNode)

		// 将目标节点标为活动节点
		cluster.nodesAlive.Store(targetNode.GetId(), targetNode)

		cluster.logger.Info("[Cluster Service] [message] node online: %s", targetNode.GetId())
	}

	// 会话关闭事件
	nodeHandler.OnSessionClosed = func(session *nio.Session) {

		// 重置连接为空
		targetNode.nodeConnector = nil

		// 将目标节点从活动节点列表中移除
		cluster.nodesAlive.Delete(targetNode.GetId())

		cluster.logger.Info("[Cluster Service] [message] node offline: %s", targetNode.GetId())

		// 如果对方是主节点，开始下一界任期选举
		if leaderNode := cluster.GetLeaderNode(); leaderNode != nil && leaderNode.GetId() == targetNode.GetId() {
			switch cluster.status {
			case statusClosed:
				return
			default:
				cluster.logger.Info("[Cluster Service] [fight] leader %s has lost. go to next term.", targetNode.GetId())
				cluster.toNextTerm()
			}
		} else {

			// 如果当前在线节点数量小于总节点数的一半，释放主节点，开始下一界任期选举
			onlineNodesCount := len(cluster.GetAliveNodeNames())
			if onlineNodesCount <= (len(cluster.GetAllNodeNames()) / 2) {
				switch cluster.status {
				case statusClosed:
					return
				default:
					if cluster.IsFighting() {
						return
					} else {
						cluster.logger.Info("[Cluster Service] [fight] Release leader. Cause of the online nodes (%d) is not enough", onlineNodesCount)
						cluster.toNextTerm()
					}
				}
			}
		}
	}

	// 收到消息事件
	nodeHandler.OnMessageReceived = func(session *nio.Session, message *nio.Msg) {

		// client端的消息内容
		nodeMsg := msg2node(message.Data())

		// 如果当前已经有主节点，并且收到的消息表明对方任期比我大，释放主节点
		if cluster.GetLeaderNode() != nil && cluster.GetMyTerm() < nodeMsg.Term {
			cluster.releaseLeader()
			return
		}

		// 同步任期
		// 收到消息后，如果自己的任期比对方小，直接更新自己的任期，保持与对方一致
		// 由于本地缓存了一份所有节点的信息，所以同时需要更新对方节点在本地缓存中的任期，尽量保持数据一致

		// 如果对方任期比我大，更新自己的任期
		cluster.GetMyNode().setTerm(nodeMsg.Term)

		// 更新本地缓存：对方节点的信息
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

		// 消息类型
		switch message.Flag() {

		case msgRAskLeader: // 询问主节点的应答

			if cluster.IsFighting() || cluster.GetLeaderNode() == nil {
				cluster.logger.Info("[Cluster Service] 任期%d 询问主节点的应答 %s", cluster.GetMyTerm(), ToJson(nodeMsg))
			}

			// 对方有主节点，如果主节上不在线，忽略
			if nodeMsg.LeaderId != "" {
				if _, ok := cluster.nodesAlive.Load(nodeMsg.LeaderId); ok == false {
					cluster.logger.Info("[Cluster Service] leader has dead. ignore your response.")
					return
				}
			}

			// 对方有主节点，尝试将自己的主节点直接指向对方的主节点
			if nodeMsg.LeaderId != "" {
				if value, ok := cluster.nodesAll.Load(nodeMsg.LeaderId); ok {
					cluster.signLeader(value.(*Node))
				}
			}

			// 应答通知
			if cluster.sendAndWaitChan != nil {
				func() {
					defer OnError("[Cluster Service] [panic] cluster msg chan")
					cluster.sendAndWaitChan <- nodeMsg
				}()
			}

		case msgRGetVote: // 请求投票的应答

			if cluster.IsFighting() || cluster.GetLeaderNode() == nil {
				cluster.logger.Info("[Cluster Service] 任期%d 请求投票的应答 %s", cluster.GetMyTerm(), ToJson(nodeMsg))
			}

			// 对方将票投给我了
			if nodeMsg.Success != nil && *nodeMsg.Success {
				cluster.GetMyNode().addVoteCount(nodeMsg.Term)
			}

			// 应答通知
			if cluster.sendAndWaitChan != nil {
				func() {
					defer OnError("[Cluster Service] [panic] cluster msg chan")
					cluster.sendAndWaitChan <- nodeMsg
				}()
			}

		case msgRLeader: // 主节点偿试心跳的应答

			if cluster.IsFighting() || cluster.GetLeaderNode() == nil {
				cluster.logger.Info("[Cluster Service] 任期%d 主节点确认心跳的应答 %s", cluster.GetMyTerm(), ToJson(nodeMsg))
			}

			// 对方拒绝了我的主节点广播消息
			if nodeMsg.Success != nil && *nodeMsg.Success == false {
				cluster.logger.Debug("[Cluster Service] [message] %s refused my leader request", ToJson(nodeMsg))

				// 如果当前节点是主，那么释放自己
				currentLeader := cluster.GetLeaderNode()
				if currentLeader != nil && currentLeader.GetId() == cluster.GetMyId() {
					cluster.releaseLeader()
				}
			}

			// 应答通知
			if cluster.sendAndWaitChan != nil {
				func() {
					defer OnError("[Cluster Service] [panic] cluster msg chan")
					cluster.sendAndWaitChan <- nodeMsg
				}()
			}

		case msgOk:

		default:
			cluster.logger.Error("[Cluster Service] [message] Client received unknown message: %s", message.Flag())
		}
	}

	return nodeHandler
}

// 同步通信，发送->应答
func (cluster *Cluster) sendAndWait(timeout time.Duration, flag uint8, data *NodeMsg) []*NodeMsg {

	// 应答消息列表
	nodeMsgList := make([]*NodeMsg, 0)

	// 为了使请求-应答能一一对应，这里必须创建一个唯一ID，应答时需要将此ID再传回来，这样发送方才能感知应答的消息对应的是哪次请求
	data.ReqId = Uuid()

	// 通信等待的超时时间
	if timeout == 0 {
		timeout = 2 * time.Second
	}

	// 发送节点列表
	nodeMap := map[string]struct{}{}

	// 应答节点通道
	syncChan := make(chan *NodeMsg)
	defer func() {
		close(syncChan)
	}()
	cluster.sendAndWaitChan = syncChan

	// 发送消息
	cluster.nodesAll.Range(
		func(key, val interface{}) bool {
			if _node, succ := val.(*Node); succ && _node.id != cluster.GetMyNode().GetId() && _node.nodeConnector != nil {
				if err := _node.SendMessage(flag, data); err != nil {
					cluster.logger.Error("[Cluster Service] [message] SendAndWait message error. %s", err)
				} else {
					nodeMap[_node.GetId()+"."+data.ReqId] = struct{}{}
				}
			}
			return true
		})

	// 等待应答
t:
	for {
		select {
		case _nodeMsg, ok := <-cluster.sendAndWaitChan:
			if !ok {
				break t
			}
			nodeMsgList = append(nodeMsgList, _nodeMsg)
			delete(nodeMap, _nodeMsg.NodeId+"."+_nodeMsg.ReqId)
			if len(nodeMap) == 0 {
				break t
			}
		case <-time.After(timeout):
			break t
		}
	}

	// 置空本次同步通道
	cluster.sendAndWaitChan = nil

	return nodeMsgList
}

// 下一界任期选举
func (cluster *Cluster) toNextTerm() {
	cluster.releaseLeader()
	go cluster.fightElection()
}

// 广播消息，当前节点更换主节点之后会广播一次
func (cluster *Cluster) broadcastLeader(leaderNode *Node) {
	cluster.nodesAll.Range(
		func(key, val interface{}) bool {
			// 广播消息不发送给自己，不发送给leader_node
			if _node, succ := val.(*Node); succ && _node.id != cluster.GetMyNode().GetId() && _node.nodeConnector != nil && _node.GetId() != leaderNode.GetId() {
				if err := _node.SendMessage(msgSBroadcastLeader, &NodeMsg{NodeId: cluster.GetMyId(), Term: cluster.GetMyTerm(), LeaderId: leaderNode.GetId()}); err != nil {
					cluster.logger.Error("[Cluster Service] [message] BroadcastNewLeader message error. %s", err)
				}
			}
			return true
		})
}
