package basic

//**************************************
// 方法
//**************************************

import (
	"fmt"
	"os"
	"ownx/tools"
	"reflect"
	"runtime"
	"strings"
)

// 声明类
type Method struct {
	pkgName string         //包名
	name    string         //方法名称，格式为pkg.funcname，示例：console.DescribeUCTag
	path    string         //方法路径，示例：jd.com/jvirt/api-vm/controller/api/v1/
	class   *Class         //所属类，当方法不是某类中的方法时，则所属类是空的，只有类实现的方法，此参数才有值并有意义
	funC    interface{}    //目标方法
	args    []reflect.Type //入参类型
	returN  []reflect.Type //出参类型
}

// 实例化
func NewMethod(cls *Class, v interface{}) *Method {
	return createMethod(cls, v)
}

// **************************************
// 公开方法
// **************************************
func (this *Method) GetPkgName() string {
	return this.pkgName
}

func (this *Method) GetName() string {
	return this.name
}

func (this *Method) GetPath() string {
	return this.path
}

func (this *Method) GetFunc() interface{} {
	return this.funC
}

func (this *Method) GetArgs() []reflect.Type {
	return this.args
}

func (this *Method) GetReturn() []reflect.Type {
	return this.returN
}

func (this *Method) GetClass() *Class {
	return this.class
}

func (this *Method) GetFullName() string {
	return this.path + this.name
}

// 反射调用方法
func (this *Method) Invoke(args []reflect.Value) []reflect.Value {
	if this.GetArgs()[0].Kind() == reflect.Struct && len(args) > 0 {
		args[0] = args[0].Elem()
	}
	return reflect.ValueOf(this.funC).Call(args)
}

// **************************************
// 私有方法
// **************************************
func createMethod(cls *Class, f interface{}) *Method {

	var funcTarget interface{}
	var name, path string
	args := make([]reflect.Type, 0)
	outs := make([]reflect.Type, 0)

	// 传入是Method
	if rv, ok := f.(reflect.Method); ok {

		// 获得方法全路径
		path = runtime.FuncForPC(rv.Func.Pointer()).Name()

		// 说明本方法是某个类中的方法，去掉类信息，并且需要去掉第一个入参
		i := 0
		if tools.MatchReg(path, `[.]\([^()]*\)`) {
			i = 1
			path = tools.Replace(path, `[.]\([^()]*\)`, "")
		}

		name, path = getNameAndPath(path)

		// 处理入参
		rt := rv.Type
		for ; i < rt.NumIn(); i++ {
			args = append(args, rt.In(i))
		}

		// 处理返回值
		for i := 0; i < rt.NumOut(); i++ {
			outs = append(outs, rt.Out(i))
		}

		// 传入是Func
	} else if reflect.TypeOf(f).Kind() == reflect.Func {

		path = runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()

		//说明此方法属于某个类，去掉类信息
		if tools.MatchReg(path, `[.]\([^()]*\)`) {
			path = tools.Replace(path, `[.]\([^()]*\)`, "")
		}

		name, path = getNameAndPath(path)

		fType := reflect.TypeOf(f)
		for i := 0; i < fType.NumIn(); i++ {
			args = append(args, fType.In(i))
		}
		for i := 0; i < fType.NumOut(); i++ {
			outs = append(outs, fType.Out(i))
		}

		// 传入是其它
	} else {
		fmt.Fprintln(os.Stderr, "Create method error. unKnown param type.")
		return nil
	}

	// 如果cls不为空，说明本方法是某个类中的方法
	if cls != nil {
		_tm := reflect.ValueOf(cls.cls).MethodByName(tools.Replace(name, `[^.]*\.`, ""))
		if _tm.IsValid() == false {
			fmt.Fprintln(os.Stderr, "Create method error. get target class by name error. name", name)
			return nil
		}
		funcTarget = _tm.Interface()
	} else {
		funcTarget = f
	}

	// 实例化一个方法
	method := &Method{
		pkgName: tools.Replace(name, `[.].*`, ""),
		name:    name,
		path:    path,
		class:   cls,
		funC:    funcTarget,
		args:    args,
		returN:  outs,
	}

	return method
}

func getNameAndPath(source_path string) (name string, path string) {

	// 方法名称，格式为 pkg.funcname
	name = tools.Replace(source_path, ".*/", "")
	path = strings.Replace(source_path, name, "", 1)

	// golang 反射得到的名称后面，不知为何有时会带个-???的后缀，将其去掉
	if tools.EndWith(name, `-\w*$`) {
		name = tools.Replace(name, `-\w*$`, "")
	}

	return
}

// **************************************
// 对外静态方法
// **************************************
func GetMethodName(v interface{}) string {
	path := runtime.FuncForPC(reflect.ValueOf(v).Pointer()).Name()
	return tools.Replace(path, ".*/", "")
}
