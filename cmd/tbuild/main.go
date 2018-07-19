package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/fatih/color"
	"github.com/freman/tbuild"
)

var builderLog = log.New(os.Stdout, color.CyanString("%10s | ", "builder"), log.LstdFlags)
var mainLog = log.New(os.Stdout, color.BlueString("%10s | ", "main"), log.LstdFlags)
var appLog = log.New(os.Stdout, color.HiWhiteString("%10s | ", "app"), log.LstdFlags)
var runnerLog = log.New(os.Stdout, color.GreenString("%10s | ", "runner"), log.LstdFlags)

var running *exec.Cmd

func main() {
	var shuttingDown bool
	doneChan := make(chan struct{})
	buildChan := make(chan struct{})
	runChan := make(chan struct{})
	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-buildChan:
				build(runChan)
			case <-doneChan:
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-runChan:
				run()
			case <-doneChan:
				if running != nil {
					running.Process.Kill()
				}
				return
			}
		}
	}()

	host, port, _ := net.SplitHostPort(config.Listen)
	if port == "" {
		port = strconv.Itoa(tbuild.DefaultPort)
	}
	if host == "" {
		host = "0.0.0.0"
	}
	listen := fmt.Sprintf("%s:%s", host, port)

	addr, err := net.ResolveUDPAddr("udp", listen)
	if err != nil {
		mainLog.Fatalf("Unable to resolve udp address %q: %v", listen, err)
	}

	listener, err := net.ListenUDP("udp", addr)
	if err != nil {
		mainLog.Fatalf("Unable to listen on udp address %q: %v", listen, err)
	}

	mainLog.Printf("Listening for udp packets on %s", listen)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			p := make([]byte, 2048)
			l, remoteaddr, err := listener.ReadFromUDP(p)
			mainLog.Printf("Received %0x from %v", p[l], remoteaddr)
			if err != nil {
				if shuttingDown {
					return
				}
				mainLog.Println(color.RedString("Dn error occurred: %v", err))
				continue
			}
			buildChan <- struct{}{}
		}
	}()

	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		shuttingDown = true
		listener.Close()
		close(doneChan)
	}()

	buildChan <- struct{}{}

	wg.Wait()
}

func build(runChan chan struct{}) {
	buildCommand := append(config.Build, "-o", ".built")
	builderLog.Printf("Kicking off build with %v", buildCommand)
	cmd, win := prepsecute(builderLog, buildCommand[0], buildCommand[1:])
	if !win {
		return
	}
	if err := cmd.Wait(); err != nil {
		builderLog.Println(color.HiRedString("Command failed due to %v", err))
		return
	}
	runChan <- struct{}{}
}

func run() {
	if running != nil {
		runnerLog.Printf("Killing PID %d", running.Process.Pid)
		running.Process.Kill()
		running = nil
	}

	if err := os.Rename(".built", "tbuild-bin"); err != nil {
		runnerLog.Println(color.HiRedString("Unable to move .built to tbuild-bin due to %v", err))
	}

	runnerLog.Printf("Kicking off run with ./tbuild-bin %v", os.Args[1:])
	cmd, win := prepsecute(appLog, "./tbuild-bin", os.Args[1:])
	if !win {
		return
	}
	running = cmd
}

func prepsecute(logger *log.Logger, name string, args []string) (*exec.Cmd, bool) {
	cmd := exec.Command(name, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.Println(color.HiRedString("Unable to pipe stdout from command due to %v", err))
		return nil, false
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		logger.Println(color.HiRedString("Unable to pipe stderr from command due to %v", err))
		return nil, false
	}

	if err := cmd.Start(); err != nil {
		logger.Println(color.HiRedString("Unable to start command due to %v", err))
		return nil, false
	}

	stdoutScanner := bufio.NewScanner(stdout)
	stderrScanner := bufio.NewScanner(stderr)

	go func() {
		for stdoutScanner.Scan() {
			logger.Println(stdoutScanner.Text())
		}
	}()

	go func() {
		for stderrScanner.Scan() {
			logger.Println(color.RedString(stderrScanner.Text()))
		}
	}()

	return cmd, true
}
