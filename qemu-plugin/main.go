package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/lima-vm/lima/pkg/limayaml"
	"github.com/lima-vm/lima/pkg/plugin"
	"github.com/lima-vm/lima/pkg/qemu"
	"gopkg.in/yaml.v3"
)

type QemuPlugin struct {
	processMap map[string]*exec.Cmd
}

func (q *QemuPlugin) Start(args plugin.StartArgs, reply *string) error {
	log.Printf("[QemuPlugin] Starting VM: %s", args.InstanceName)

	ctx := context.Background()
	var limaConfig limayaml.LimaYAML
	if err := yaml.Unmarshal(args.ConfigData, &limaConfig); err != nil {
		return fmt.Errorf("failed to unmarshal config: %v", err)
	}
	qCfg := qemu.Config{
		Name:         args.InstanceName,
		InstanceDir:  args.InstanceDir,
		LimaYAML:     &limaConfig,
		SSHLocalPort: args.SSHLocalPort,
		SSHAddress:   args.SSHAddress,
	}

	qExe, qArgs, _ := qemu.Cmdline(ctx, qCfg)

	var qArgsFinal []string
	applier := &qArgTemplateApplier{}
	for _, unapplied := range qArgs {
		applied, err := applier.applyTemplate(unapplied)
		if err != nil {
			return err
		}
		qArgsFinal = append(qArgsFinal, applied)
	}
	qCmd := exec.CommandContext(ctx, qExe, qArgsFinal...)
	qCmd.ExtraFiles = append(qCmd.ExtraFiles, applier.files...)
	qStdout, _ := qCmd.StdoutPipe()
	go logPipeRoutine(qStdout, "qemu[stdout]")
	qStderr, _ := qCmd.StderrPipe()
	go logPipeRoutine(qStderr, "qemu[stderr]")

	log.Printf("[QemuPlugin] Starting QEMU process")
	if err := qCmd.Start(); err != nil {
		log.Printf("[QemuPlugin] Failed to start QEMU: %v", err)
		return err
	}
	processID := qCfg.Name + "-" + qCfg.InstanceDir + "-" + strconv.Itoa(os.Getpid())
	q.processMap[processID] = qCmd
	*reply = processID
	return nil
}

func logPipeRoutine(r io.Reader, header string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("%s: %s", header, line)
	}
}

type qArgTemplateApplier struct {
	files []*os.File
}

func (a *qArgTemplateApplier) applyTemplate(qArg string) (string, error) {
	if !strings.Contains(qArg, "{{") {
		return qArg, nil
	}
	funcMap := template.FuncMap{
		"fd_connect": func(v any) string {
			fn := func(v any) (string, error) {
				s, ok := v.(string)
				if !ok {
					return "", fmt.Errorf("non-string argument %+v", v)
				}
				addr, err := net.ResolveUnixAddr("unix", s)
				if err != nil {
					return "", err
				}
				conn, err := net.DialUnix("unix", nil, addr)
				if err != nil {
					return "", err
				}
				f, err := conn.File()
				if err != nil {
					return "", err
				}
				if err := conn.Close(); err != nil {
					return "", err
				}
				a.files = append(a.files, f)
				fd := len(a.files) + 2 // the first FD is 3
				return strconv.Itoa(fd), nil
			}
			res, err := fn(v)
			if err != nil {
				panic(fmt.Errorf("fd_connect: %w", err))
			}
			return res
		},
	}
	tmpl, err := template.New("").Funcs(funcMap).Parse(qArg)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	if err := tmpl.Execute(&b, nil); err != nil {
		return "", err
	}
	return b.String(), nil
}

func runServer(address string) error {
	plugin := &QemuPlugin{
		processMap: make(map[string]*exec.Cmd), // 初始化 processMap
	}

	// Register RPC service
	if err := rpc.Register(plugin); err != nil {
		return fmt.Errorf("RPC register error: %w", err)
	}

	// Start listening
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("listen error: %w", err)
	}
	defer listener.Close()

	log.Printf("[QemuPlugin] Listening on %s", address)

	// Handle shutdown signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigs
		log.Println("[QemuPlugin] Shutting down...")
		listener.Close()
		os.Exit(0)
	}()

	// Accept and serve connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go rpc.ServeConn(conn)
	}
}

func main() {
	if err := runServer("127.0.0.1:9991"); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
