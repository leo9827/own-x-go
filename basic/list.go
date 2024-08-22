package basic

type List interface {
	Add(v interface{})
	AddLast(v interface{})
	AddFirst(v interface{})
	AddBefore(index int, v interface{}) error // 在指定位置前追加一个元素，下标从0开始
	Remove(index int) interface{}
	RemoveData(v interface{}) bool // 删除第一个匹配到的数据
	RemoveUntil(index int) error   // 从前向后开始删除，包含指定下标
	RemoveFirst() interface{}
	RemoveLast() interface{}
	Get(index int) interface{}
	Size() int
	Clear()
	Loop(callback func(index int, v interface{}) bool) // 遍历链表，返回true继续，返回false停止
}
