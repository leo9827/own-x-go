package env

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var programPath = ""

var hostIp = ""

func setProgramPath() {
	if programPath != "" {
		return
	}
	file, _ := exec.LookPath(os.Args[0])
	ApplicationPath, _ := filepath.Abs(file)
	programPath = strings.Replace(ApplicationPath, "\\", "/", -1)
}

func setHostIp() {
	if hostIp != "" {
		return
	}
	hostname, _ := os.Hostname()
	ips, _ := net.LookupIP(hostname)
	for _, ip := range ips {
		if ip.To4() != nil {
			hostIp = ip.To4().String()
			return
		}
	}
}

func GetProgramPath() string {
	if programPath == "" {
		setProgramPath()
	}
	return programPath
}

func GetHostIp() string {
	if hostIp == "" {
		setHostIp()
	}
	return hostIp
}
