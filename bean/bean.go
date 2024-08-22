package bean

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"
)

// bean对象管理
// 实例化的接口，可通过调用 bean.AddBean(interface) 或 bean.SetBean(name, interface) 加入到bean管理中
// 所有对象都加入到bean管理后，调用 bean.ioc 进行依赖注入
// 依赖注入可实现以下功能：
// 引用的对象可通过 bean 中的对象进行自动注入，不需要手工实例化
// 引用方式例子：
// type Object struct {
//     InstanceService jcs.InstanceService `autowrite`    //自动注入：字段类型是interface接口，字段名称必须与实现类名称一致，并且实现类与接口的package要一致
//     OrderDao        *dao.OrderDao       `autowrite`    //自动注入：字段类型是struct类型指针
//     LogMessage      mq.Mq               `autowrite`    //自动注入：字名名称是自定义的，此时必须使用 bean.SetBean 自定义一个名字，这里才会识别到并自动注入
//     ImageApi        api.JcsImageApi     `bean:"SpecialImageApi" autowrite` //自动注入：通过bean标签显示说明，要从bean中取SpecialImageApi注入给自己
// }

const (
	K_AUTOWRITE = "autowrite"
)

var AUTOWRITE_REG = regexp.MustCompile(`\b` + K_AUTOWRITE + `\b`)

type beans struct {
	mu sync.RWMutex
	m  map[string]interface{}
}

var beanContext = &beans{
	m: make(map[string]interface{}),
}

// 设置依赖注入
func Ioc() {
	beanContext.mu.RLock()
	defer beanContext.mu.RUnlock()
	for beanName, bean := range beanContext.m {
		findAndSet(beanName, bean)
	}
}

// 开始查找autowrite标签的字段，进行注入(必须是指针字段)
func findAndSet(beanName string, beanInterface interface{}) {

	// Bean类型
	beanType := reflect.TypeOf(beanInterface)

	// Bean目标
	bean := reflect.Indirect(reflect.ValueOf(beanInterface))

	// 遍历Bean的字段
	for i := 0; i < bean.NumField(); i++ {

		// 得到一个字段
		field := bean.Field(i)

		// 获得字段类型
		var fieldType reflect.StructField
		if field.Kind() == reflect.Interface || field.Kind() == reflect.Ptr {
			fieldType = beanType.Elem().Field(i)
		} else if field.Kind() == reflect.Struct {
			fieldType = bean.Addr().Elem().Type().Field(i)
		} else {
			continue
		}

		// 如果没有autowrite标签，忽略
		if isAutowrite(string(fieldType.Tag)) == false {
			continue
		}

		// 不支持非指针或非接口
		if field.Kind() == reflect.Struct {
			panic(fmt.Sprintf("ioc autowrite fail. dependent bean is not ptr or interface. Bean=%s DependentBean=%s.", beanName, fieldType.Name))
		}

		// 如果字段值不是空的，忽略
		if field.IsNil() == false {
			continue
		}

		// 如果字段不可设置，错误
		if field.CanSet() == false {
			panic(fmt.Sprintf("ioc autowrite fail. field can not set. Bean=%s DependentBean=%s.", beanName, fieldType.Name))
		}

		// 查找依赖时的优先级处理
		// 一，取tag中设置的bean名字
		// 二，取变量名称
		// 三，取package.StructName
		// 四，取package.InterfaceName

		// 1. 偿试使用tag中的名字
		dependentBeanName := fieldType.Tag.Get("bean")

		// 2. 偿试使用变量名称
		if dependentBeanName == "" {
			dependentBeanName = fieldType.Name
		}

		// 从BeanContext中获得依赖Bean
		propBean := GetBean(dependentBeanName)

		// 3. 尝试使用package.StructName
		if propBean == nil {
			dependentBeanName = strings.Replace(fieldType.Type.String(), "*", "", 1) // 得到变量类型的名字
			propBean = GetBean(dependentBeanName)                                    // 得到变量的名字
		}

		// 4. 如果是接口类型，会出现类名与接口名不一致的情况，将Bean名称修正为真实类的名称，偿试使用package.InterfaceName
		if propBean == nil && field.Kind() == reflect.Interface {
			dependentBeanName = tools.Replace(fieldType.Type.String(), fieldType.Type.Name(), fieldType.Name) // 将接口的名字替换为变量的名字，如定义："UserServiceImpl api.UserService"，得到的结果为 api.UserServiceImpl
			propBean = GetBean(dependentBeanName)
		}

		if propBean != nil {
			field.Set(reflect.ValueOf(propBean))
		} else {
			panic(fmt.Sprintf("ioc autowrite fail. dependent bean not found. Bean=%s DependentBean=%s.", beanName, dependentBeanName))
		}

		// 递归
		if field.Kind() == reflect.Ptr {
			findAndSet(fieldType.Name, field.Interface())
		}
	}
}

func isAutowrite(tag string) bool {
	return AUTOWRITE_REG.Match([]byte(tag))
}

// 设置指定名称的Bean
func SetBean(beanName string, bean interface{}) {
	if strings.Index(beanName, ".") < 0 {
		beanName = strings.Title(beanName)
	}
	beanContext.mu.Lock()
	defer beanContext.mu.Unlock()
	if beanContext.m[beanName] != nil {
		panic(fmt.Sprintf("add bean fail. bean %s already exists.", beanName))
		return
	}
	beanContext.m[beanName] = bean
}

// 设置Bean，名称等于类名称，首字母小写
func AddBean(bean interface{}) {
	// 必须使用指针变量或接口类型变量
	if reflect.TypeOf(bean).Kind() == reflect.Struct {
		panic(fmt.Sprintf("add bean fail. bean %s must be interface or ptr.", reflect.TypeOf(bean).String()))
	}
	beanName := basic.GetClassName(bean)
	part := strings.Split(beanName, ".")
	beanName = part[0] + "." + strings.Title(part[1])
	SetBean(beanName, bean)
}

// 获得Bean
func GetBean(beanName string) interface{} {
	beanContext.mu.RLock()
	defer beanContext.mu.RUnlock()
	return beanContext.m[beanName]
}
