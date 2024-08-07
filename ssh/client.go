package ssh

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

var Ciphers = []string{
	"aes128-ctr",
	"aes192-ctr",
	"aes256-ctr",
	"aes128-gcm@openssh.com",
	"arcfour256",
	"arcfour128",
	"aes128-cbc",
	"3des-cbc",
	"aes192-cbc",
	"aes256-cbc",
}

type Cmd struct {
	addr           string
	username       string
	password       string
	keyFile        string
	client         *ssh.Client
	connectTimeout time.Duration
	ioTimeout      time.Duration
}

var logger = log.New(os.Stdout, "ssh", 1)

func NewCmd(addr string) *Cmd {
	return &Cmd{addr: addr, username: "root", connectTimeout: 10 * time.Second, ioTimeout: 10 * time.Second}
}

func (c *Cmd) User(username string) *Cmd {
	c.username = username
	return c
}

func (c *Cmd) Password(password string) *Cmd {
	c.password = password
	return c
}

func (c *Cmd) KeyFile(keyFile string) *Cmd {
	c.keyFile = keyFile
	return c
}

func (c *Cmd) ConnectTimeout(connectTimeout time.Duration) *Cmd {
	if connectTimeout > 0 {
		c.connectTimeout = connectTimeout
	}
	return c
}

func (c *Cmd) IoTimeout(ioTimeout time.Duration) *Cmd {
	if ioTimeout > 0 {
		c.ioTimeout = ioTimeout
	}
	return c
}

func (c *Cmd) Connect() (*Cmd, error) {
	auth, err := c.genAuthMethod()
	if err != nil {
		return c, err
	}
	clientConfig := &ssh.ClientConfig{
		Auth:            auth,
		User:            c.username,
		Timeout:         c.connectTimeout,
		Config:          ssh.Config{Ciphers: Ciphers},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil },
	}
	conn, err := net.DialTimeout("tcp", c.addr, c.connectTimeout)
	if err != nil {
		return nil, fmt.Errorf("net.dial failed. %w", err)
	}
	timeoutConn := &Conn{conn, c.ioTimeout, c.ioTimeout}
	sshConn, chans, reqs, err := ssh.NewClientConn(timeoutConn, c.addr, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("ssh.conn failed. %w", err)
	}

	c.client = ssh.NewClient(sshConn, chans, reqs)
	return c, nil
}

func (c *Cmd) Close() {
	if c.client != nil {
		err := c.client.Close()
		if err != nil {
			return
		}
		c.client = nil
	}
}

func (c *Cmd) genAuthMethod() ([]ssh.AuthMethod, error) {
	var authMethods []ssh.AuthMethod
	if c.password != "" {
		authMethods = append(authMethods, ssh.Password(c.password))
	}
	if c.keyFile != "" {
		permBytes, err := os.ReadFile(c.keyFile)
		if err != nil {
			return nil, fmt.Errorf("ParseRawPrivateKey failed. " + err.Error())
		}
		key, err := ssh.ParseRawPrivateKey(permBytes)
		if err != nil {
			return nil, fmt.Errorf("ParseRawPrivateKey failed. " + err.Error())
		}
		signer, err := ssh.NewSignerFromKey(key)
		if err != nil {
			return nil, fmt.Errorf("NewSignerFromKey failed. " + err.Error())
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}
	return authMethods, nil
}

// RunCmdWithEnv execute command with env
func (c *Cmd) RunCmdWithEnv(command string, envs map[string]string, mustEnv bool) *Result {
	if len(envs) == 0 {
		return c.RunCmd(command)
	}
	logger.Printf("run command [%s] env %s", command, envs)

	session, err1 := c.client.NewSession()
	if err1 != nil {
		return &Result{command: command, err: fmt.Errorf("new session failed. " + err1.Error())}
	}
	defer func(session *ssh.Session) {
		err := session.Close()
		if err != nil {
			logger.Printf("close session failed. %v", err)
		}
	}(session)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	session.Stdout = stdout
	session.Stderr = stderr

	stdin, err := session.StdinPipe()
	if err != nil {
		return &Result{command: command, err: fmt.Errorf("session stdin pipe failed. " + err.Error())}
	}

	if err = session.Shell(); err != nil {
		return &Result{command: command, err: err}
	}

	for name, value := range envs {
		if _, err = fmt.Fprintf(stdin, "export %s=%s\n", name, value); err != nil && mustEnv {
			return &Result{command: command, err: err}
		}
	}

	// execute command
	if _, err = fmt.Fprintf(stdin, "%s\n", command); err != nil {
		return &Result{command: command, err: err}
	}
	if err = stdin.Close(); err != nil {
		return &Result{command: command, err: err}
	}

	err = session.Wait()
	return &Result{
		command: command,
		stdout:  strings.TrimSuffix(stdout.String(), "\n"),
		stderr:  strings.TrimSuffix(stderr.String(), "\n"),
		err:     err,
	}
}

// RunCmd execute command without env
func (c *Cmd) RunCmd(command string) *Result {
	logger.Printf("run command [%s]", command)
	session, err1 := c.client.NewSession()
	if err1 != nil {
		return &Result{command: command, err: fmt.Errorf("new session failed. " + err1.Error())}
	}
	defer func(session *ssh.Session) {
		err := session.Close()
		if err != nil {
			logger.Printf("close session failed. %v", err)
		}
	}(session)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	session.Stdout = stdout
	session.Stderr = stderr
	err2 := session.Run(command)
	return &Result{
		command: command,
		stdout:  strings.TrimSuffix(stdout.String(), "\n"),
		stderr:  strings.TrimSuffix(stderr.String(), "\n"),
		err:     err2,
	}
}

func (c *Cmd) PutData(data interface{}, file string, fileMode os.FileMode) (n int64, err error) {

	logger.Printf("put data [%s]", file)
	if data == nil || file == "" {
		return 0, errors.New("put file failed. invalid data or file name")
	}

	sftpClient, err := sftp.NewClient(c.client)
	if err != nil {
		return 0, fmt.Errorf("new sftp client error: %w", err)
	}
	defer sftpClient.Close()

	// create dir if not exists
	if dir, _ := filepath.Split(file); dir != "" {
		if _, err = sftpClient.Stat(dir); err != nil || !os.IsExist(err) {
			if err = sftpClient.MkdirAll(dir); err != nil {
				return 0, fmt.Errorf("mkdir %s failed: %w", dir, err)
			}
		}
	}

	target, err := sftpClient.Create(file)
	if err != nil {
		return 0, fmt.Errorf("sftp client open file %s error: %w", file, err)
	}
	defer func(target *sftp.File) {
		err := target.Close()
		if err != nil {
			logger.Printf("close file %v failed. %v", target, err)
		}
	}(target)

	// chmod
	if fileMode != 0 {
		if err := target.Chmod(fileMode); err != nil {
			return 0, fmt.Errorf("chmod file %s failed: %w", file, err)
		}
	}

	switch data.(type) {
	case []byte:
		sb := &bytes.Buffer{}
		sb.Write(data.([]byte))
		n, err = io.Copy(target, sb)
	case io.Reader:
		n, err = io.Copy(target, data.(io.Reader))
	default:
		return 0, errors.New("data only support []byte or io.Reader")
	}
	if err != nil {
		return 0, fmt.Errorf("copy file %s error: %w", file, err)
	}
	return n, nil
}

func (c *Cmd) PutFile(localPath, remotePath string) (err error) {

	logger.Printf("put local file [%s -> %s]", localPath, remotePath)
	if localPath == "" || remotePath == "" {
		return errors.New("put file failed. invalid local filepath or remote filepath")
	}

	sftpClient, err := sftp.NewClient(c.client)
	if err != nil {
		return fmt.Errorf("new sftp client error: %w", err)
	}
	defer sftpClient.Close()
	return c.putFile(localPath, remotePath, sftpClient)
}

func (c *Cmd) putFile(localPath, remotePath string, sftpClient *sftp.Client) error {

	srcFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open local file %s failed. %w", localPath, err)
	}
	defer srcFile.Close()

	stat, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("stat local file %s failed. %w", localPath, err)
	}

	if stat.IsDir() {
		files, err := srcFile.Readdir(0)
		if err != nil {
			return fmt.Errorf("read local dir %s failed. %w", localPath, err)
		}
		targetDir := filepath.Join(remotePath, stat.Name())
		logger.Printf("remote mkdir [%s %s]", stat.Mode(), targetDir)
		_ = sftpClient.MkdirAll(targetDir)
		_ = sftpClient.Chmod(targetDir, stat.Mode())
		for _, file := range files {
			if err := c.putFile(filepath.Join(localPath, file.Name()), targetDir, sftpClient); err != nil {
				return err
			}
		}
		return nil
	}

	file := filepath.Join(remotePath, stat.Name())

	if dir, _ := filepath.Split(file); dir != "" {
		if _, err = sftpClient.Stat(dir); err != nil || !os.IsExist(err) {
			logger.Printf("remote mkdir [%s]", dir)
			if err = sftpClient.MkdirAll(dir); err != nil {
				return fmt.Errorf("remote mkdir %s failed: %w", dir, err)
			}
		}
	}

	target, err := sftpClient.Create(file)
	if err != nil {
		return fmt.Errorf("sftp client open remote file %s error: %w", file, err)
	}
	defer func(target *sftp.File) {
		err := target.Close()
		if err != nil {
			logger.Printf("close remote file %v failed. %v", target, err)
		}
	}(target)

	logger.Printf("remote create file [%s %s]", stat.Mode(), file)

	if err := target.Chmod(stat.Mode()); err != nil {
		return fmt.Errorf("chmod remote file %s failed: %w", file, err)
	}

	_, err = io.Copy(target, srcFile)
	if err != nil {
		return fmt.Errorf("copy to remote file %s error: %w", file, err)
	}
	return nil
}

func (c *Cmd) GetFile(
	remoteFile string,
	createDir func(remoteFile string, fileInfo os.FileInfo) error,
	getWriter func(remoteFile string, fileInfo os.FileInfo) (*Writer, error),
) (err error) {

	logger.Printf("get remote file [%s]", remoteFile)
	if remoteFile == "" || createDir == nil {
		return errors.New("get file failed. invalid file name or createDir func")
	}

	sftpClient, err := sftp.NewClient(c.client)
	if err != nil {
		return fmt.Errorf("new sftp client error: %w", err)
	}
	defer sftpClient.Close()
	return c.getFile(remoteFile, sftpClient, createDir, getWriter)
}

func (c *Cmd) getFile(
	remoteFile string,
	sftpClient *sftp.Client,
	createDir func(remoteFile string, fileInfo os.FileInfo) error,
	getWriter func(remoteFile string, fileInfo os.FileInfo) (*Writer, error),
) error {

	sourceFile, err := sftpClient.Open(remoteFile)
	if err != nil {
		return fmt.Errorf("sftp client open remote file %s error: %w", remoteFile, err)
	}
	defer func(sourceFile *sftp.File) {
		err := sourceFile.Close()
		if err != nil {
			logger.Printf("close remote file %v failed. %v", sourceFile, err)
		}
	}(sourceFile)

	stat, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("stat remote file %s failed. %w", remoteFile, err)
	}

	if stat.IsDir() {
		if err = createDir(remoteFile, stat); err != nil {
			return err
		}
		files, err := sftpClient.ReadDir(remoteFile)
		if err != nil {
			return fmt.Errorf("read remote dir %s failed. %w", remoteFile, err)
		}
		for _, f := range files {
			if err = c.getFile(filepath.Join(remoteFile, f.Name()), sftpClient, createDir, getWriter); err != nil {
				return err
			}
		}
		return nil
	}

	out, err := getWriter(remoteFile, stat)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err = io.Copy(out, sourceFile); err != nil {
		return fmt.Errorf("copy remote file %s error: %w", remoteFile, err)
	}
	return nil
}

type Result struct {
	command string
	stdout  string
	stderr  string
	err     error
}

func (r *Result) HasError(printDetailWhenError bool) bool {
	hasErr := r.stderr != "" || r.err != nil
	if hasErr && printDetailWhenError {
		logger.Fatalf("[command=%s]. detail=%s", r.command, r.Detail())
	}
	return hasErr
}

func (r *Result) Detail() string {
	return fmt.Sprintf("[stdout=%s] [stderr=%s] [exit=%v]", r.GetStdout(), r.GetStderr(), r.err)
}

func (r *Result) ErrInfo() string {
	if r.stderr != "" {
		return r.stderr
	}
	if r.err != nil {
		return r.err.Error()
	}
	if r.stdout != "" {
		return r.stdout
	}
	return ""
}

func (r *Result) IsExit0() bool {
	return r.err == nil
}

func (r *Result) GetCommand() string {
	return r.command
}

func (r *Result) GetStdout() string {
	return r.stdout
}

func (r *Result) GetStderr() string {
	return r.stderr
}

type Writer struct {
	io.Writer
	CleanFunc func()
}

func (w *Writer) Close() {
	w.CleanFunc()
}

type Conn struct {
	net.Conn
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func (c *Conn) Read(b []byte) (int, error) {
	err := c.Conn.SetReadDeadline(time.Now().Add(c.ReadTimeout))
	if err != nil {
		return 0, err
	}
	return c.Conn.Read(b)
}

func (c *Conn) Write(b []byte) (int, error) {
	err := c.Conn.SetWriteDeadline(time.Now().Add(c.WriteTimeout))
	if err != nil {
		return 0, err
	}
	return c.Conn.Write(b)
}
