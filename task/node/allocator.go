package node

import (
	"fmt"
	"ownx/cluster"
	"ownx/task"
	"sync"
	"time"
)

type Allocator struct {
	Name               string
	mu                 sync.Mutex
	Worker             *task.Worker // should inject
	NodePartition      map[string]*task.Partition
	TaskBuildFuncs     []func(round int64, nodeIPList interface{}) ([]*task.Task, error)
	CutInLineFuncs     []func(nodeIPList interface{}) ([]*task.Task, error)
	cancelRoundTimeout chan int64
}

var DefaultNodeAllocator = newAllocator("default-node-allocator")

func newAllocator(name string) *Allocator {
	return &Allocator{
		Name:           name,
		mu:             sync.Mutex{},
		Worker:         task.DefaultWorker,
		NodePartition:  make(map[string]*task.Partition),
		TaskBuildFuncs: make([]func(round int64, idList interface{}) ([]*task.Task, error), 0),
		CutInLineFuncs: make([]func(idList interface{}) ([]*task.Task, error), 0),
	}
}

// StartAlloc should register in Scheduler
func (a *Allocator) StartAlloc(round int64) {
	now := time.Now()
	defer func() {
		fmt.Printf("%s StartAlloc time elapsed ms: %d", a.Name, time.Since(now).Milliseconds())
	}()
	defer a.roundDoneTimeout(round)

	fmt.Printf("%s StartAlloc, round: %d", a.Name, round)
	if err := a.prevCheck(); err != nil {
		fmt.Printf("%s StartAlloc: should init and inject require fields before start, err: ",
			a.Name, err)
		return
	}

	aliveNodeNames := cluster.DefaultCluster.GetAliveNodeNames()
	aliveNodesCount := len(aliveNodeNames)
	if aliveNodesCount == 0 {
		fmt.Printf("%s aliveNodesCount == 0, waiting round done timeout", a.Name)
		return
	}

	partitionCount := len(aliveNodeNames)
	eachPartitionNodes := aliveNodesCount / len(aliveNodeNames) // 均分

	// 按照某个规则将 node 排序 ( sort by) 然后开始分配
	nodePartition := make(map[string]*task.Partition)
	rangeStartIndex := 0
	rangeEndIndex := eachPartitionNodes
	for i := 0; i < partitionCount; i++ {
		p := &task.Partition{
			NodeName:   aliveNodeNames[i],
			StartIndex: rangeStartIndex + (eachPartitionNodes * i),
			EndIndex:   rangeEndIndex + (eachPartitionNodes * i),
			SortBy:     "created_at", // default is `created_at`
			Round:      round,
			BatchSize:  1000, // default is 1000
		}
		p.BatchCount = (p.EndIndex - p.StartIndex) / p.BatchSize
		if (p.EndIndex-p.StartIndex)%p.BatchSize > 0 {
			p.BatchCount = p.BatchCount + 1
		}
		nodePartition[aliveNodeNames[i]] = p
		if i == (partitionCount - 1) { // last partition
			p.EndIndex = aliveNodesCount - 1
		}
	}

	// sync all node know  partitions
	clusterJobList := make(map[string]interface{})
	for _, nodeName := range aliveNodeNames {
		clusterJobList[nodeName] = nodePartition
	}
	cluster.DefaultCluster.PublishJob(SyncPartition, clusterJobList)
}

func (a *Allocator) roundDoneTimeout(round int64) {
	go func(round int64) {
		select {
		case rd := <-a.cancelRoundTimeout:
			fmt.Printf("%s cancelRoundTimeout!, round: %d", a.Name, round)
			if rd == round {
				return
			} else {
				time.Sleep(500 * time.Millisecond)
				a.cancelRoundTimeout <- rd // 防止正确的timeout接收不到
			}
		case <-time.After(5 * time.Second):
			fmt.Printf("%s roundDoneTimeout!, round: %d", a.Name, round)
			task.DefaultScheduler.RoundDone(a.Name, round)
		}
	}(round)
}

const (
	SyncPartition     string = "task_node_sync_partition"
	SyncPartitionDone string = "task_node_sync_partition_done"
	ManualCutInLine   string = "task_node_cut_in_line"
)

func (a *Allocator) StartAfterInjected() {
	if err := a.prevCheck(); err != nil {
		fmt.Printf("Allocator Start Error: should init and inject require fields before start, %s", err)
		return
	}
	err := cluster.DefaultCluster.RegisterFunc(SyncPartition, a.syncPartitions)
	if err != nil {
		fmt.Printf("Allocator Start, RegisterFunc on DefaultCluster Err! %s", err)
	}
	err1 := cluster.DefaultCluster.RegisterFunc(SyncPartitionDone, a.syncPartitionDone)
	if err1 != nil {
		fmt.Printf("Allocator Start, RegisterFunc on DefaultCluster Err! %s", err1)
	}
	err2 := cluster.DefaultCluster.RegisterFunc(ManualCutInLine, a.cutInLine)
	if err2 != nil {
		fmt.Printf("Allocator Start, RegisterFunc on DefaultCluster Err! %s", err2)
	}
	task.DefaultScheduler.RoundCallRegistry(a.Name, a.StartAlloc)
}

func (a *Allocator) syncPartitions(data interface{}) {
	if np, ok := data.(map[string]*task.Partition); ok {
		a.mu.Lock()
		a.NodePartition = np
		a.mu.Unlock()

		a.roundRun()
	}
}

func (a *Allocator) roundRun() {
	myName := cluster.DefaultCluster.GetMyName()
	if _, exists := a.NodePartition[myName]; !exists {
		fmt.Println("err: ", myName, " can't find partition, should join next round")
		return
	}

	// 每个节点持有数据自己的分组规则 range(startIndex ~ endIndex), sortBy(所有节点保持一致)
	var (
		myPartition = a.NodePartition[myName]
		offset      = myPartition.StartIndex
		batchSize   = myPartition.BatchSize
		batchCount  = myPartition.BatchCount
		currRound   = myPartition.Round
	)

	for index := 0; index < batchCount; index++ {
		nodeIPList := cluster.DefaultCluster.GetAliveNodeIps()[index:(index + batchCount)]

		if len(nodeIPList) > 0 { // build job and send to worker
			for _, f := range a.TaskBuildFuncs {
				tasks, err := f(currRound, nodeIPList)
				if err != nil {
					fmt.Printf("[%s call roundRun on %s][exec TaskBuildFunc interrupted!][offset: %d, batch-size:%d, err:%w]",
						a.Name, myName, offset, batchSize, err)
					continue
				}

				tasks = append(tasks, a.batchDoneTask(index)) // 在所有任务执行完成之后，放入一个 batchDone 的任务
				a.Worker.Put(tasks)
			}
		}
		offset += batchSize
	}
}

func (a *Allocator) batchDoneTask(batchIndex int) *task.Task {
	batchFinTask := &task.Task{
		ID:         time.Now().Unix(),
		F:          a.partitionDone,
		Data:       batchIndex,
		NeedRetry:  false,
		RetryTimes: 0,
		RetryLimit: 0,
	}
	return batchFinTask
}

func (a *Allocator) partitionDone(data interface{}) error {
	// 分2部分，这里就不分成2个函数了
	// 第1部分：batch done，更新 partition 数据
	// 第2部分: partition done， 同步到其他节点，并且判断存活节点，决定是否触发 round done 事件 开启下一轮round
	batchIndex, ok := data.(int)
	if !ok {
		return fmt.Errorf("data is not a num of batch index")
	}
	fmt.Printf("%s recv batch done event, batch index: %d", a.Name, batchIndex)
	a.mu.Lock()
	myName := cluster.DefaultCluster.GetMyName()
	myPartition, exists := a.NodePartition[myName]
	if !exists {
		a.mu.Unlock()
		return fmt.Errorf("cant find my partition")
	}
	fmt.Printf("%s recv batch done event, batch index: %d, batch-count:%d",
		a.Name, batchIndex, myPartition.BatchCount)
	myPartition.FinBatchCount = myPartition.FinBatchCount + 1
	if myPartition.FinBatchCount < myPartition.BatchCount {
		a.mu.Unlock()
		return fmt.Errorf("partition done")
	} else {
		myPartition.RoundDone = true
	}
	a.mu.Unlock()

	aliveNodeNames := cluster.DefaultCluster.GetAliveNodeNames()

	intersectionNodeNames := make([]string, 0)
	for _, nodeName := range aliveNodeNames {
		if _, ok := a.NodePartition[nodeName]; ok {
			intersectionNodeNames = append(intersectionNodeNames, nodeName)
		}
	}

	if len(intersectionNodeNames) == 0 { // prev node is all dead, should start new round
		a.callRoundDone(myPartition.Round)
	}

	allAliveNodePartitionDone := true
	for _, nodeName := range intersectionNodeNames {
		if nodePart, ok := a.NodePartition[nodeName]; ok {
			if myPartition.NodeName == nodeName {
				continue
			}
			if !nodePart.RoundDone {
				allAliveNodePartitionDone = false
				break
			}
		}
	}
	if allAliveNodePartitionDone { // should start new round
		a.callRoundDone(myPartition.Round)
		return nil
	} else {
		// publish sync partition done event
		clusterJobList := make(map[string]interface{}) // node_name, data := range job_list
		for _, nodeName := range intersectionNodeNames {
			clusterJobList[nodeName] = myPartition
		}
		cluster.DefaultCluster.PublishJob(SyncPartitionDone, clusterJobList)
		return nil
	}
}

func (a *Allocator) syncPartitionDone(data interface{}) {
	pt, ok := data.(*task.Partition)
	if !ok {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if partition, ok := a.NodePartition[pt.NodeName]; ok {
		partition.RoundDone = pt.RoundDone
	}
	return
}

func (a *Allocator) callRoundDone(round int64) {
	task.DefaultScheduler.RoundDone(a.Name, round)
	a.cancelRoundTimeout <- round
}

func (a *Allocator) prevCheck() error {
	if len(a.TaskBuildFuncs) == 0 {
		return fmt.Errorf("TaskBuildFuncs not init")
	}
	if a.Worker == nil {
		return fmt.Errorf("worker not init")
	}
	return nil
}

const usingClusterThreshold = 1024

// recv from cluster
func (a *Allocator) cutInLine(data interface{}) {
	nodeIPList, ok := data.([]string)
	if !ok {
		return
	}
	if len(nodeIPList) == 0 {
		return
	}

	if len(nodeIPList) <= usingClusterThreshold {
		for _, cutInF := range a.CutInLineFuncs {
			tasks, err := cutInF(nodeIPList)
			if err != nil {
				fmt.Printf("call cut in func err! ", err)
				continue
			}
			a.Worker.Put(tasks)
		}
	} else {
		batch := (len(nodeIPList) + usingClusterThreshold - 1) / usingClusterThreshold
		batchData := make([][]string, 0, batch)
		for i := 0; i < batch; i++ {
			start := i * usingClusterThreshold
			end := start + usingClusterThreshold
			if i == batch-1 {
				end = len(nodeIPList) - 1
			}
			batchData = append(batchData, nodeIPList[start:end])
		}

		aliveNodeNames := cluster.DefaultCluster.GetAliveNodeNames()
		for index, nodeName := range aliveNodeNames {
			ipList := batchData[index]
			cluster.DefaultCluster.PublishJob(ManualCutInLine, map[string]interface{}{
				nodeName: ipList,
			})
		}

	}
}

func (a *Allocator) RegistryTaskBuild(f func(round int64, nodeIPList interface{}) ([]*task.Task, error)) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.TaskBuildFuncs == nil {
		a.TaskBuildFuncs = make([]func(round int64, nodeIPList interface{}) ([]*task.Task, error), 0)
	}
	a.TaskBuildFuncs = append(a.TaskBuildFuncs, f)
	return nil
}

func (a *Allocator) RegistryCutInLineTaskBuild(f func(nodeIPList interface{}) ([]*task.Task, error)) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.CutInLineFuncs == nil {
		a.CutInLineFuncs = make([]func(nodeIPList interface{}) ([]*task.Task, error), 0)
	}
	a.CutInLineFuncs = append(a.CutInLineFuncs, f)
	return nil
}

func (a *Allocator) RegistryWorker(worker *task.Worker) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.Worker != nil {
		return fmt.Errorf("already has worker")
	} else {
		a.Worker = worker
	}
	return nil
}
