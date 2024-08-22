package basic

//**************************************
// 类
//**************************************

import (
	"reflect"
	"strings"
	"sync"
)

// Class 声明类
type Class struct {
	mu      sync.RWMutex       //锁
	cls     interface{}        //目标对象
	name    string             //类名称
	methods map[string]*Method //方法集合
}

// NewClass 实例化
func NewClass(cls interface{}) *Class {
	return createClass(cls)
}

// GetName 获得类名称
func (c *Class) GetName() string {
	return c.name
}

// GetMethod 根据方法名称获得方法，如：pkg.funcname
func (c *Class) GetMethod(methodName string) *Method {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.methods[methodName]
}

// GetAllMethod 获得此类中所有方法
func (c *Class) GetAllMethod() map[string]*Method {
	return c.methods
}

// GetClassName 获得类名称
func GetClassName(v interface{}) string {
	return strings.Replace(reflect.TypeOf(v).String(), "*", "", 1)
}

func createClass(cls interface{}) *Class {

	// 实例化
	obj := &Class{
		methods: make(map[string]*Method),
		cls:     cls,
		name:    GetClassName(cls),
	}

	// 遍历所有方法并实例化并缓存起来
	ele := reflect.TypeOf(cls)
	if ele.NumMethod() == 0 {
		return obj
	}

	for i := 0; i < ele.NumMethod(); i++ {
		obj.addMethod(NewMethod(obj, ele.Method(i)))
	}
	return obj
}

func (c *Class) addMethod(method *Method) {
	if method == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.methods[method.GetName()] = method
}
