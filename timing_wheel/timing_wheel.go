package timing_wheel

import (
	"fmt"
	"time"
)

type Timing interface {
	AddOnce(task TimingTask, timestamp int64)
	AddCron(task TimingTask, cron string)
}

type timingWheel struct {
	current *current
	buckets []*tierTask // should init with fixed length
}

type current struct {
	timestamp int64
	tier      int64 // default 0
	index     int   // default 0
}

func (c *current) Incr() {
	c.timestamp = c.timestamp + 1
	c.index = c.index + 1
	if c.index == defaultBucketLength {
		c.TierIncr()
	}
}
func (c *current) TierIncr() {
	c.tier = c.tier + 1
	c.index = 0
}

type tierTask struct {
	tier  int64
	tasks []TimingTask
	prev  *tierTask
	next  *tierTask
}

func (t *timingWheel) AddOnce(task TimingTask, timestamp int64) {
	fmt.Println(fmt.Sprintf("[AddOnce][task timestamp:%d, time-wheel current timestamp:%d]",
		timestamp, t.current.timestamp))
	if timestamp <= t.current.timestamp {
		timestamp = t.current.timestamp
	}
	var (
		taskRemainSec   = timestamp - t.current.timestamp
		taskTier        = t.current.tier + taskRemainSec/int64(defaultBucketLength)
		taskBucketIndex = t.current.index + int(taskRemainSec%int64(defaultBucketLength))
	)
	if taskBucketIndex >= defaultBucketLength {
		taskTier = taskTier + 1
		taskBucketIndex = taskBucketIndex % defaultBucketLength
	}

	fmt.Println(fmt.Sprintf("[AddOnce][taskRemainSec:%d, taskTier:%d, taskBucketIndex:%d]",
		taskRemainSec, taskTier, taskBucketIndex))

	t1 := []TimingTask{task}
	if tierList := t.buckets[taskBucketIndex]; tierList == nil {
		t.buckets[taskBucketIndex] = &tierTask{tier: taskTier, tasks: t1}
	} else {
		var tierPointer = tierList
		for ; tierPointer != nil; tierPointer = tierPointer.next {
			if taskTier < tierPointer.tier {
				prev := tierPointer.prev
				tierP := tierTask{
					tier:  taskTier,
					tasks: t1,
					prev:  prev,
					next:  tierPointer,
				}
				prev.next = &tierP
				tierPointer.prev = &tierP
				break
			}
			if tierPointer.tier == taskTier {
				if tierPointer.tasks != nil {
					tierPointer.tasks = append(tierPointer.tasks, task) //append to tail
				} else {
					tierPointer.tasks = t1
				}
				break
			}
			if (taskTier > tierPointer.tier && tierPointer.next == nil) ||
				(taskTier > tierPointer.tier && tierPointer.next != nil && taskTier < tierPointer.next.tier) {
				next := tierPointer.next
				tierP := tierTask{
					tier:  taskTier,
					tasks: t1,
					prev:  tierPointer,
					next:  next,
				}
				tierPointer.next = &tierP
				break
			}
		}
	}
}

func (t *timingWheel) AddCron(task TimingTask, cron string) {

}

// global variable
var defaultTimingWheel = &timingWheel{buckets: make([]*tierTask, defaultBucketLength)}
var defaultBucketLength = 32

func AddTask(t TimingTask, timestamp int64) {
	defaultTimingWheel.AddOnce(t, timestamp)
}

func Init() {
	defaultTimingWheel.current = &current{timestamp: time.Now().Unix(), index: 0, tier: 0}
	// read local task from file

	// re import tasks end

	// start timing-wheel
	go func() {
		for {
			for i := 0; i < defaultBucketLength; i++ {

				fmt.Println(fmt.Sprintf("timing-wheel executing, tier:%d, index: %d",
					defaultTimingWheel.current.tier, defaultTimingWheel.current.index))

				defaultTimingWheel.current.Incr()

				linkedList := defaultTimingWheel.buckets[i]
				if linkedList != nil && defaultTimingWheel.current.tier == linkedList.tier {
					// linkedList head reset
					defaultTimingWheel.buckets[i] = defaultTimingWheel.buckets[i].next
					if defaultTimingWheel.buckets[i] != nil {
						defaultTimingWheel.buckets[i].prev = nil
					}

					// execute tasks
					taskList := linkedList.tasks
					for _, task := range taskList {
						t := task
						go func() {
							t.Run(10)
						}()
					}
				}

				time.Sleep(time.Millisecond * 100) // bucket interval
			}
		}
	}()
}
