package rpc

import (
	"context"
	"errors"
	"net"
	"net/rpc"
	"os/exec"
	"time"

	"github.com/lima-vm/lima/pkg/driver"
	"github.com/lima-vm/lima/pkg/limayaml"
	"github.com/lima-vm/lima/pkg/plugin"
	"github.com/sirupsen/logrus"
)

type RPCDriver struct {
	base   *driver.BaseDriver
	client *rpc.Client
	cmd    *exec.Cmd
}

func New(base *driver.BaseDriver) *RPCDriver {
	cmd := exec.Command("/lima/qemu-plugin/lima-qemu-plugin")

	if err := cmd.Start(); err != nil {
		logrus.Errorf("[RPCDriver] failed to start external driver: %v", err)
		return nil
	}

	if err := waitForPlugin("127.0.0.1:9991", 10*time.Second); err != nil {
		logrus.Errorf("[RPCDriver] failed to wait for plugin: %v", err)
		return nil
	}

	client, err := rpc.Dial("tcp", "127.0.0.1:9991")
	if err != nil {
		logrus.Errorf("[RPCDriver] failed to dial driver RPC: %v", err)
		return nil
	}

	return &RPCDriver{
		base:   base,
		client: client,
		cmd:    cmd,
	}
}

// waitForPlugin waits for the RPC server to be available within the timeout period.
func waitForPlugin(address string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, 1*time.Second)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(500 * time.Millisecond) // Retry interval
	}
	return errors.New("timeout waiting for plugin to start")
}

func (d *RPCDriver) Start(ctx context.Context) (chan error, error) {
	logrus.Info("[RPCDriver] Sending Start request to RPC server")
	ch := make(chan error, 1)

	go func() {
		defer close(ch)

		configBytes, err := limayaml.Marshal(d.base.Instance.Config, true)
		if err != nil {
			logrus.Errorf("[RPCDriver] Config marshal error: %v", err)
			ch <- err
			return
		}

		args := plugin.StartArgs{
			InstanceName: d.base.Instance.Name,
			Config:       configBytes,
		}

		var reply bool
		if err := d.client.Call("QemuPlugin.Start", args, &reply); err != nil {
			logrus.Errorf("[RPCDriver] Start failed: %v", err)
			ch <- err
			return
		}

		logrus.Infof("[RPCDriver] Start succeeded (reply: %v)", reply)
		ch <- nil
	}()

	return ch, nil
}

func (d *RPCDriver) Stop(_ context.Context) error {
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
