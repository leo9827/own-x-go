package cluster

import (
	"github.com/prometheus/client_golang/prometheus"
)

var clusterStatusGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "cluster_status",
		Help: "cluster master slave status",
	},
	[]string{"ip"},
)

func init() {
	prometheus.MustRegister(clusterStatusGauge)
}

func (cluster *Cluster) updateMetrics() {
	if cluster.GetMyNode() != nil {
		ip := cluster.GetMyNode().GetHost()
		isReady := cluster.IsReady()
		IsLeader := cluster.IsLeader()
		switch isReady {
		case false:
			clusterStatusGauge.WithLabelValues(ip).Set(-1)
		case true:
			switch IsLeader {
			case false:
				clusterStatusGauge.WithLabelValues(ip).Set(0)
			case true:
				clusterStatusGauge.WithLabelValues(ip).Set(1)
			}
		}
	}
}
