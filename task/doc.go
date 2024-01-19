// Package monitor /*
// 组件依赖关系以及启动顺序
// - Scheduler - 不依赖其他, 提供注册， 有全局默认实例 DefaultScheduler
// - Worker    - 不依赖其他， 			有全局默认实例 DefaultWorker
// - Allocator - 依赖 Worker 执行任务；需要注册到 Scheduler，由 Scheduler 进行调度，Start时注册到cluster 和 scheduler； 提供注册
// - Executor  - 依赖于 Dao 操作数据； 需要注册到 Allocator，由 Allocator 分配任务
package task
