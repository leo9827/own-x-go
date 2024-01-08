package cluster

import (
	"github.com/prometheus/client_golang/prometheus"
)

var clusterStatusGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "cluster_status",
		Help: "集群主从状态",
	},
	[]string{"ip"},
)

// 初始化 prometheus 指标
func init() {
	// 注册
	prometheus.MustRegister(clusterStatusGauge)
}

func (cluster *Cluster) updateMetrics() {
	if cluster.GetMyNode() != nil {
		ip := cluster.GetMyNode().GetHost()
		isReady := cluster.IsReady()
		IsLeader := cluster.IsLeader()
		switch isReady {
		case false:
			// 集群状态没有就绪
			clusterStatusGauge.WithLabelValues(ip).Set(-1)
		case true:
			switch IsLeader {
			// 从节点
			case false:
				clusterStatusGauge.WithLabelValues(ip).Set(0)
			// 主节点
			case true:
				clusterStatusGauge.WithLabelValues(ip).Set(1)
			}
		}
	}
}
