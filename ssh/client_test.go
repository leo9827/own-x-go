package ssh

import (
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd("localhost:22")
	defer cmd.Close()
	conn, err := cmd.Connect()
	if err != nil {
		t.Fatalf("connect err: %v", err)
	}
	conn.RunCmd("echo 'hello'")
}
