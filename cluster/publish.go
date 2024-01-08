package cluster

import (
	"errors"
)

type JobInfo struct {
	Name string
	Data interface{}
}

func (cluster *Cluster) RegisterFunc(taskName string, f func(data interface{})) error {
	cluster.lock.Lock()
	defer cluster.lock.Unlock()
	if taskName == "" {
		return errors.New("RegisterFunc TaskName is nil")
	} else {
		cluster.taskTracker[taskName] = f
	}
	return nil
}

func (cluster *Cluster) CallFunc(funcName string, nodeName string, param interface{}) {
	cluster.PublishJob(funcName, map[string]interface{}{nodeName: param})
}

func (cluster *Cluster) PublishJob(jobName string, jobList map[string]interface{}) {
	cluster.logger.Debug("PublishJob [%s, %s]", jobName, ToJson(jobList))

	for nodeName, data := range jobList {

		if cluster.GetMyNode().GetName() == nodeName {

			cluster.pushTask(jobName, data)

		} else {
			cluster.nodesAlive.Range(
				func(key, val interface{}) bool {
					if _node, ok := val.(*Node); ok && _node.GetName() == nodeName {
						if err := _node.SendMessage(msgSPublishJob, &JobInfo{Name: jobName, Data: data}); err != nil {
							cluster.logger.Error("PublishJob %s failed. Cause of %s.", jobName, err.Error())
						}
						return false
					}
					return true
				})
		}
	}
}

func (cluster *Cluster) pushTask(jobName string, data interface{}) {

	f := cluster.taskTracker[jobName]
	if f != nil {
		go func() {
			defer OnError("[panic] Push task")
			f(data)
		}()
	} else {
		cluster.logger.Error("Push task failed. Cause of job[%s] didn't set taskTracker func.", jobName)
	}
}
