// Package node
// 说明：
//
// 调度分批的维度是 node-ip-list, 分区规则是：均分，
//
// 为什么没有按照module 或者 monitor-item 的维度来调度 ?
// => 0公平分配: 尽量照顾到所有的 ip, 让每个 ip 都有机会执行任务, 不能出现一些 ip 执行快，抢占其他 ip 执行机会
// => 1考虑到最终每个 ip 需要执行多个脚本并得到结果，那么尽量在每一次ssh到ip时，把需要执行的 script-list 都执行了，减少 ssh 连接的建立次数
// => 2为什么不按 module 分配, 也是考虑到尽可能将 ip-list 进行均分（部分 module可能拥有巨量 ip-list，部分 module 可能只有少数 ip-list)
// => 3优化思路：按照 module-ip-list 再优化，按照 module-list 进行分区，但要求每个分区下的 ip-list 总量差距不大
//
// 集群中, 每个 cluster node 负责一部分 node-ip-list 的监控项执行
// 每个 ip 要执行的 script-list 在每个节点接收到当前节点分区任务后，开始执行当前分区任务时进行确定（会查询并得到 ip-scriptList 这样的结构）
//
// 插队任务，插队任务会被立即执行，不计入每一轮的执行
//
// 关于执行失败的处理：
// => 单个 ip 执行失败后目前不做处理, 等下一轮调度再执行
//
// cluster 节点挂掉怎么办？
// => 会判断其他节点是否执行完成来结束当前轮次进行下一轮调度, 下一轮调度重新分配任务给存活的节点
//
// 调度策略
// => 所有节点执行完毕为一轮,每一轮执行完成通知 scheduler 开启下一轮
package node