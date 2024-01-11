package monitor

/*
     ┌────────────────────────┐
     │                        │
     │                        │
     │                        │
     │                        │
     └────────────────────────┘
	node - [job1, job2, job3]
1 execute
	[module1 - [mnitem-1, mnitem-2] -> [node1,node2] -> [job1,job2][job1,job2,job3]]
	[module2 - [mnitem-1] -> [node1,node2] -> [job1][job1]]
2 cancel -> module-mnitem
	node - [job1, job2, job3]

	支持 100+ w 节点的任务执行和调度,
	要求: 1支持插队
 // suppose: 100w nodes, support insert a job into queue head

	1. 节点列表和节点的任务列表如何管理
	2. 如何控制所有节点全部执行完成
	3. 100w+ 节点顺序执行,每一轮节点的执行顺序是否需要一致
	4. 100w+ 节点每轮执行时如果需要分批执行，如何保证控制执行的批次和中断重试
	5. 集群环境下，每个节点的任务如何分配，如何传递任务，如果节点挂掉任务如何转移
	6. 任务状态如何保存 ? 如何做到故障恢复 ? 节点初始化和重上线任务分配 ?
	7. 如何支持插队任务 ? 如果使用 round 标记节点执行轮次，插队任务是否需要更新 round ?
	8. 不需要执行的任务如何取消?

	/////////////////
	假设集群3个 worker,那么先暂时均分100w/3=33w,每个 worker 处理 33w 节点的任务执行
	先设计单个 worker 的任务执行
     ┌───────────┐
     │   worker  │
     └───────────┘
	要做到故障转移,那么 worker 可以设计为无状态或状态定时同步（那么要求任务可重复执行（这点没问题))

	首先是 worker 自己的任务执行框架
	1.task queue, 任务队列（如果选用 chan 实现,那么对插队支持不友好)
	2.goroutine pool ( thread pool), 这点可以用 goroutine + done chan 来做到, 每个 goroutine = 2kb, 计划 1000 个 goroutine = 2000kb -= 2m
	3.因为存在着 1000 个 goroutine , 所以要考虑将同一个 round 的任务分成小块, 不要引发竞争,
		 如果某些 goroutine 提前完成任务,那么考虑做 work-steal, 但是这样可能引发竞争(对待执行任务的读取和修改）
	如何分配任务让所有 goroutine 不产生竞争?
	1.worker 共享 chan, 所有 goroutine read for chan, 好处是所有任务都可以 put 进这个 chan
	2.每个 goroutine 一个 chan, 各自读取自己的 chan 完成任务, 坏处是 chan 任务分配以及内存丢失重新分配需要一套规则,并且无法做 work-steal
x	3.work 共享 hashtable-chan, 多个 goroutine 共享一个 hashtable-bucket-chan, 并可以容忍单个 goroutine 失败或超时,

	以上仅是 task-worker 队列分配考虑, 并且,需要考虑整个 task-hashtable 的执行情况监控

	然后是 33w 节点, 如何分配到这个 hashtable 中? 以及如何支持插队 task?
	首先是每轮 round 如何分配? 这里是否需要携带偏向 partition-id 来支持携带业务信息的分区执行？
	估算hashtable 每个bucket 每个 round 需要执行多少节点, 按照 id 进行 hash 分区 , 假设 hashtable len=1024, 33w/(1024*0.85)=379.13602941
	每个task支持多少节点? 如果每个 task 支持 10 个节点,那么 goroutine 中是否还要再并发 goroutine ?

	假设每个节点建立一个 ssh 连接, 每个 task 1个节点, 那么同一时刻 1000 个 goroutine 会产生 1000 个 ssh 个连接

	插队如何支持?
	chan 是 queue, 支持 fifo , 不支持 add to head

    task 的如何分配, 每个 round 执行情况如何统计, 失败任务如何处理? task 是否支持回调,
	task 函数如何执行?

	需要有哪些组件? 每个组件主要职责是什么? 组件之间如何协同?





*/

type Assigner struct {
}

var _a *Assigner = &Assigner{}

func (a *Assigner) recv(data interface{}) {

}

type Sentinel struct {
}

var _s *Sentinel = &Sentinel{}

func (s *Sentinel) recv(data interface{}) {

}

type Task struct {
}

type Worker struct {
}

func (s *Worker) recv(data interface{}) {

}

var _w *Worker = &Worker{}

func init() {
	//_ = cluster.DefaultCluster.RegisterFunc("assign", _a.recv)
	//_ = cluster.DefaultCluster.RegisterFunc("sentinel", _s.recv)
	//_ = cluster.DefaultCluster.RegisterFunc("worker", _w.recv)
}
