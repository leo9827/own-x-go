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
     │   worker  │ ---- [chan1,chan2,chan3...] ---- [[goroutine1,goroutine2],[goroutine3,goroutine4]]
     └───────────┘

	要做到故障转移,那么 worker 可以设计为无状态或状态定时同步（那么要求任务可重复执行（按业务需求(执行ssh获取信息)来看没问题))

	首先是 worker 自己的任务执行框架
	1.task queue, 任务队列（如果选用 chan 实现,那么对插队支持不友好)
	2.goroutine pool ( thread pool), 这点可以用 goroutine + done chan 来做到, 每个 goroutine = 2kb, 计划 1000 个 goroutine = 2000kb -= 2m
	3.因为存在着 1000 个 goroutine , 所以要考虑将同一个 round 的任务分成小块, 不要引发竞争,
		 如果某些 goroutine 提前完成任务,那么考虑做 work-steal, 但是这样可能引发竞争(对待执行任务的读取和修改）
	如何分配任务让所有 goroutine 不产生竞争?
	1.worker 共享 chan, 所有 goroutine read for chan, 好处是所有任务都可以 put 进这个 chan
	2.每个 goroutine 一个 chan, 各自读取自己的 chan 完成任务, 坏处是 chan 任务分配以及内存丢失重新分配需要一套规则,并且无法做 work-steal
x	3.work 共享 array-chan, 多个 goroutine 共享一个 bucket-chan, 并可以容忍单个 goroutine 失败或超时,

	以上仅是 task-worker 队列分配考虑, 并且,需要考虑整个 task-array-chan 的执行情况监控

	然后是 33w 节点, 如何分配到这个 array-chan 中? 以及如何支持插队 task?
	首先是每轮 round 如何分配? 这里是否需要携带偏向 partition-id 来支持携带业务信息的分区执行？
	估算 array 每个bucket 每个 round 需要执行多少节点, 按照 id 进行 hash 分区 , 假设 array len=1024, 33w/1024=322
	每个task支持多少节点? 如果每个 task 支持 10 个节点,那么 goroutine 中是否还要再并发 goroutine ?

	>假设每个节点建立一个 ssh 连接, 每个 task 1个节点, 那么同一时刻 1000 个 goroutine 会产生 1000 个 ssh 个连接

	插队如何支持?
	chan 是 queue, 仅支持 fifo , 不支持 add in head

    task 的如何分配, 每个 round 执行情况如何统计, 失败任务如何处理? task 是否支持回调,
	task 函数如何执行?

	需要有哪些组件? 每个组件主要职责是什么? 组件之间如何协同?






list all required feature:
1.task allocate
2.task execute
3.task status update
4.



allocate feature to components
1.


components collaborative ways:
1. send event to a global eventbus: need a global eventbus(cluster can do this)
2. send message to another component: call another component method, need use global instance and known call which one
3. invoke registry callback function: components need save other component callback function, and registry on init

data transfer:
1. component communicate only using task-id
2. using task-id get task detail from librarian



components starting sequence
1. assigner
2. worker
3. librarian
4. sentinel

     ┌──────────┐
     │  worker  │
     └──────────┘



┘└┌ ┐

         ┌────────┐             ┌─────────┐
       ┌─│assigner│ <---------- │ sentinel│ <┐
       | └────────┘             └─────────┘  │
       |                                     │
       |                                     │
       |                                     │
       |                                     │
       |              ┌─chan1─┐              │
       |              └───────┘              │
       |  ┌────────┐  ┌─chan2─┐          ┌───┴────┐      ┌─────────┐
       └> │ worker │  └───────┘          │        │      └─────────┘
          │        │  ┌─chan3─┐          │ facade │      ┌─────────┐
          └────────┘  └───────┘          │        │      └─────────┘
              │                          └────────┘
           ┌─────┐							 ^
           v     v                           │
         ┌──┐   ┌──┐                         │
         │g1│   │g2│   ──────────────────────┘
         └──┘   └──┘
          │
          │
          v
       ┌─────┐
       │node1│
       └─────┘





组件及职责:
 - task-worker:    main worker, concurrency goroutines
 - task-assigner:  assign task  to worker
 - task-scheduler: round schedule
 - task-sentinel:
 -

Scheduler（调度器）：Scheduler 用于描述负责任务调度和管理的组件。它可能涉及任务的优先级管理、时间调度、资源分配等。如果你的组件的主要职责是任务的调度和管理，Scheduler 这个名字可能比较合适。
Allocator（分配器）：Allocator 通常用于描述资源或内存的分配。在任务分发组件中，Allocator 可能负责将任务分配给可用的资源或工作单元。如果你的组件的主要职责是任务的分配和资源的管理，Allocator 这个名字可能比较合适。
Dispatcher（调度器）：Dispatcher 用于描述负责任务分发的组件。它可能涉及将任务分发给不同的执行者、处理任务队列、任务优先级等。如果你的组件的主要职责是任务的分发和调度，Dispatcher 这个名字可能比较合适。
Assigner（分配器）：Assigner 用于描述负责将任务分配给特定执行者或工作单元的组件。它可能根据规则或算法选择最合适的执行者，并将任务分配给它们。如果你的组件的主要职责是将任务分配给可用的执行者，Assigner 这个名字可能比较合适。


Distributor（分发器）：描述负责任务分发的组件，将任务分发给不同的执行者或资源。
Router（路由器）：指示任务分发组件负责将任务路由到适当的执行者或处理单元。
Dispatcher（调度器）：描述负责任务调度和分发的组件，确保任务按照特定的策略或规则分配给执行者。
Coordinator（协调器）：用于描述负责协调任务分发和执行的组件，确保任务的顺序和一致性。
Orchestrator（编排器）：指示任务分发组件负责协调和编排多个任务的执行。
Assigner（分配器）：描述负责将任务分配给可用执行者或资源的组件，可能基于某些规则或算法进行分配。
Scheduler（调度器）：指示任务分发组件负责任务的调度和时间管理，确保任务按照特定的计划进行执行。
Manager（管理器）：描述负责任务分发和管理的组件，可能涉及任务队列、资源分配等。

	for i := 1; i <= 100; i++ {
		go func(id int) {
			defer func(){ fmt.Println("close channel") }()
			for f := range shareChan {
			  // 当多个goroutine同时执行该循环时，它们会竞争从通道中读取元素。通道会确保每个元素只被一个goroutine读取，即每个元素只能被消费一次。
			  // 协作的方式是，当通道中没有可读取的元素时，goroutine会阻塞在for v := range this.endTaskChan这一行，直到通道中有新的元素可供读取。这种阻塞的机制使得多个goroutine能够协调地从通道中获取元素，避免了重复读取的情况。
			  // 当有新的元素被发送到this.endTaskChan通道时，其中一个阻塞的goroutine会被唤醒，它会读取该元素并执行相应的任务。其他的goroutine仍然处于阻塞状态，等待下一个可读取的元素。
			  // 总结起来，多个goroutine之间通过通道进行协作，每个goroutine按顺序读取通道中的元素，并执行相应的任务，确保每个元素只被消费一次，并避免了重复读取的情况。
			  f()
			}
		}()
	}


*/

package monitor
