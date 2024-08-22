package tools

import (
	"bytes"
	json1 "encoding/json"
	"fmt"
	"math"
	"net"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"text/template"
)

var funcMaps = map[string]interface{}{}

func init() {
	Register(false) // 注册公共function
}

func AddFuncs(funcs template.FuncMap, exit bool) {
	for k, v := range funcs {
		if _, ok := funcMaps[k]; ok {
			if exit {
				os.Exit(1)
			} else {
				continue
			}
		}

		funcMaps[k] = v
	}
}

func GetFuncs() template.FuncMap {
	f := make(map[string]interface{})
	for k, v := range funcMaps {
		f[k] = v
	}
	return f
}

// Register 注册公共function
func Register(exit bool) {
	AddFuncs(commonTemplateFuncs, exit)
}

var commonTemplateFuncs = template.FuncMap{
	"ManageIp":          GetManageIp,
	"LocalIpFirst":      LocalIPFirst,
	"Contains":          Contains,
	"HostName":          hostName,
	"IsDPDK":            isDpdk,
	"Plus1":             plusOne,
	"Plus":              Plus,
	"Sub1":              subOne,
	"MateSub":           Sub,
	"Iterate":           Iterate,
	"InList":            InList,
	"Int64":             toInt64,
	"Last":              Last,
	"JsonFormat":        JsonFormat,
	"JsonFormatIndent":  JsonFormatIndent,
	"LocalIPFirstList":  LocalIPFirstList,
}

func JsonFormat(i interface{}) string {
	jsonBytes, err := json1.Marshal(i)
	if err != nil {
		return ""
	} else {
		return string(jsonBytes)
	}
}

func JsonFormatIndent(i interface{}) string {
	jsonBytes, err := json1.MarshalIndent(i, "", "    ")
	if err != nil {
		return ""
	} else {
		return string(jsonBytes)
	}
}

// toInt64 converts integer types to 64-bit integers
func toInt64(v interface{}) int64 {
	if str, ok := v.(string); ok {
		iv, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return 0
		}
		return iv
	}

	val := reflect.Indirect(reflect.ValueOf(v))
	switch val.Kind() {
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		return val.Int()
	case reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return int64(val.Uint())
	case reflect.Uint, reflect.Uint64:
		tv := val.Uint()
		if tv <= math.MaxInt64 {
			return int64(tv)
		}
		// TODO: What is the sensible thing to do here?
		return math.MaxInt64
	case reflect.Float32, reflect.Float64:
		return int64(val.Float())
	case reflect.Bool:
		if val.Bool() == true {
			return 1
		}
		return 0
	default:
		return 0
	}
}

func InList(strList []string, str string) bool {
	for _, tmp := range strList {
		if tmp == str {
			return true
		}
	}
	return false
}

func Iterate(start, end int64) []int64 {
	var Items []int64
	for i := start; i < end; i++ {
		Items = append(Items, i)
	}
	return Items
}

func Sub(a, b int64) int64 {
	return a - b
}

func Plus(a, b int64) int64 {
	return a + b
}

func Last(x int, a interface{}) bool {
	return x == reflect.ValueOf(a).Len()-1
}

func Contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func LocalIPFirst(origin string) string {
	ips := strings.Split(origin, ",")
	localIp, err := GetManageIp()
	if err != nil {
		return origin
	}
	ipList := []string{localIp}
	for _, ip := range ips {
		if ip == localIp {
			continue
		} else {
			ipList = append(ipList, ip)
		}
	}
	return strings.Join(ipList, ",")
}

func LocalIPFirstList(ips []string) string {
	localIp, err := GetManageIp()
	if err != nil {
		return strings.Join(ips, ",")
	}
	ipList := []string{localIp}
	for _, ip := range ips {
		if ip == localIp {
			continue
		} else {
			ipList = append(ipList, ip)
		}
	}
	return strings.Join(ipList, ",")
}

func shellStdout(command string) (error, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return err, stdout.String(), stderr.String()
}

func GetManageIp() (string, error) {
	var ipAddr net.IP
	var err error

	var addrs []net.IP
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}
	// 尝试直接通过 hostname 获取 ip 地址
	addrs, _ = net.LookupIP(hostname)
	for _, addr := range addrs {
		if err = validateNodeIP(addr); err == nil {
			if addr.To4() != nil {
				ipAddr = addr
				break
			}
			if addr.To16() != nil && ipAddr == nil {
				ipAddr = addr
			}
		}
	}

	// 如果获取不到，通过默认路由获取
	if ipAddr == nil {
		ipAddr, err = getDefaultRouteIp()
		if err != nil {
			return "", err
		}
	}
	return ipAddr.String(), nil
}

func getDefaultRouteIp() (net.IP, error) {
	var ipAddr net.IP
	var err error
	var stdout string
	err, stdout, _ = shellStdout("ip r get 1")
	if err != nil {
		return nil, err
	}
	/*
	  ip route get 1
	  1.0.0.0 via 10.217.173.161 dev eth0 src 10.217.173.168
	    cache
	*/
	// parse stdout
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		if strings.Contains(line, "src") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "src" {
					ipAddr = net.ParseIP(fields[i+1])
					break
				}
			}
		}
	}

	if ipAddr == nil {
		return nil, fmt.Errorf("can't get manage ip")
	}
	return ipAddr, nil
}

func hostName() string {
	hostname, err := os.Hostname()
	if err != nil {
		return err.Error()
	}
	return hostname
}

// isDpdk 基于网卡名判断是否为 dpdk
func isDpdk() (bool, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return false, err
	}
	// TODO 不严谨,但是暂时没有其他方案
	for _, iface := range interfaces {
		if iface.Name == "br-phy" {
			return true, nil
		}
	}

	return false, nil
}

func validateNodeIP(nodeIP net.IP) error {
	if nodeIP.To4() == nil && nodeIP.To16() == nil {
		return fmt.Errorf("nodeIP must be a valid IP address")
	}
	if nodeIP.IsLoopback() {
		return fmt.Errorf("nodeIP can't be loopback address")
	}
	if nodeIP.IsMulticast() {
		return fmt.Errorf("nodeIP can't be a multicast address")
	}
	if nodeIP.IsLinkLocalUnicast() {
		return fmt.Errorf("nodeIP can't be a link-local unicast address")
	}
	if nodeIP.IsUnspecified() {
		return fmt.Errorf("nodeIP can't be an all zeros address")
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return err
	}
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip != nil && ip.Equal(nodeIP) {
			return nil
		}
	}
	return fmt.Errorf("node IP: %q not found in the host's network interfaces", nodeIP.String())
}

func plusOne(x int) int {
	return x + 1
}

func subOne(x int) int {
	return x - 1
}
