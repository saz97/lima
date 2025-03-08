package rpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/rpc"
	"os/exec"
	"time"

	"github.com/lima-vm/lima/pkg/driver"
	"github.com/saz97/lima/pkg/plugin"
)

type RPCDriver struct {
	base   *driver.BaseDriver
	client *rpc.Client
	cmd    *exec.Cmd
}

func New(base *driver.BaseDriver) *RPCDriver {
	cmd := exec.Command("/lima/qemu-plugin/lima-qemu-plugin")

	// 启动插件进程并捕获错误
	if err := cmd.Start(); err != nil {
		panic(fmt.Errorf("failed to start external driver: %w", err))
	}

	// 等待插件监听端口（最大等待 10 秒）
	timeout := time.After(10 * time.Second)
	tick := time.Tick(500 * time.Millisecond)
	for {
		select {
		case <-timeout:
			panic("timeout waiting for plugin to start")
		case <-tick:
			conn, err := net.DialTimeout("tcp", "127.0.0.1:9999", 1*time.Second)
			if err == nil {
				conn.Close()
				goto Connected
			}
		}
	}
Connected:

	client, err := rpc.Dial("tcp", "127.0.0.1:9999")
	if err != nil {
		panic(fmt.Errorf("failed to dial driver RPC: %w", err))
	}

	return &RPCDriver{
		base:   base,
		client: client,
		cmd:    cmd,
	}
}

// 下面的 Start 是与 driver.Driver 接口匹配的方法。
// 可以转发到外部 RPC 服务
func (r *RPCDriver) Start(ctx context.Context) (chan error, error) {
	ch := make(chan error, 1)
	go func() {
		var reply bool
		args := plugin.StartArgs{
			InstanceName: r.base.Instance.Name,
			Config:       []byte("...some config..."),
		}
		err := r.client.Call("QemuPlugin.Start", args, &reply)
		ch <- err
	}()
	return ch, nil
}

// Stop 方法
func (r *RPCDriver) Stop(ctx context.Context) error {
	var reply bool
	args := plugin.StopArgs{
		InstanceName: r.base.Instance.Name,
	}
	err := r.client.Call("QemuPlugin.Stop", args, &reply)
	if err != nil {
		return err
	}

	// 可在 Stop 后关闭进程
	if r.cmd.Process != nil {
		_ = r.cmd.Process.Kill()
	}
	return nil
}
func (d *RPCDriver) Validate() error {
	return nil
}

func (d *RPCDriver) Initialize(_ context.Context) error {
	return nil
}

func (d *RPCDriver) CreateDisk(_ context.Context) error {
	return nil
}

func (d *RPCDriver) CanRunGUI() bool {
	return false
}

func (d *RPCDriver) RunGUI() error {
	return nil
}

func (d *RPCDriver) Register(_ context.Context) error {
	return nil
}

func (d *RPCDriver) Unregister(_ context.Context) error {
	return nil
}

func (d *RPCDriver) ChangeDisplayPassword(_ context.Context, _ string) error {
	return nil
}

func (d *RPCDriver) GetDisplayConnection(_ context.Context) (string, error) {
	return "", nil
}

func (d *RPCDriver) CreateSnapshot(_ context.Context, _ string) error {
	return errors.New("unimplemented")
}

func (d *RPCDriver) ApplySnapshot(_ context.Context, _ string) error {
	return errors.New("unimplemented")
}

func (d *RPCDriver) DeleteSnapshot(_ context.Context, _ string) error {
	return errors.New("unimplemented")
}

func (d *RPCDriver) ListSnapshots(_ context.Context) (string, error) {
	return "", errors.New("unimplemented")
}

func (d *RPCDriver) ForwardGuestAgent() bool {
	// if driver is not providing, use host agent
	return true
}

func (d *RPCDriver) GuestAgentConn(_ context.Context) (net.Conn, error) {
	// use the unix socket forwarded by host agent
	return nil, nil
}
