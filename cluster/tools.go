package cluster

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	uuid "github.com/satori/go.uuid"
	"math/big"
	"net"
	"reflect"
	"regexp"
	"runtime/debug"
	"time"
)

func msg2node(bs []byte) *NodeMsg {
	var data NodeMsg
	_ = Unmarshal(bs, &data)
	return &data
}

func msg2job(bs []byte) *JobInfo {
	var jobInfo JobInfo
	_ = Unmarshal(bs, &jobInfo)
	return &jobInfo
}

func Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func Uuid() string {
	u1 := uuid.NewV4()
	return u1.String()
}

func GetRandomNum(n int) int {
	for {
		if random, err := rand.Int(rand.Reader, big.NewInt(int64(n))); err != nil {
			continue
		} else {
			return int(random.Int64())
		}
	}
}

func OnError(txt string) {
	if r := recover(); r != nil {
		fmt.Printf("%s - Got a runtime error %s. %s\n%s", time.Now(), txt, r, string(debug.Stack()))
	}
}

func GetIntranetIp() ([]string, error) {
	localIps := make([]string, 0)
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return localIps, err
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && !ipnet.IP.IsLinkLocalUnicast() && !ipnet.IP.IsLinkLocalMulticast() {
			if ipnet.IP.To4() != nil {
				localIps = append(localIps, ipnet.IP.String())
			}
		}
	}
	return localIps, nil
}

func ToString(v interface{}) string {
	if v == nil {
		return ""
	}
	if t := reflect.ValueOf(v); t.Kind() == reflect.Ptr && t.IsNil() {
		return ""
	}
	return fmt.Sprint(v)
}

func ToJson(v interface{}) string {
	j, _ := json.Marshal(v)
	return string(j)
}

func Replace(str string, reg string, newStr string) string {
	pattern, err := regexp.Compile(reg)
	if err != nil {
		return str
	}
	return pattern.ReplaceAllString(str, newStr)
}
