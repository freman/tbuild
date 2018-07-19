package main

import (
	"flag"
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/freman/tbuild"
	fsnotify "gopkg.in/fsnotify.v1"
)

func main() {
	remote := flag.String("remote", "", "remote address to notify")
	flag.Parse()

	if *remote == "" {
		fmt.Println("need a remote")
		return
	}

	if !strings.Contains(*remote, ":") {
		*remote = *remote + ":" + strconv.Itoa(tbuild.DefaultPort)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println(err)
		return
	}

	if err := watcher.Add("."); err != nil {
		fmt.Println(err)
		return
	}

	t := time.AfterFunc(500*time.Millisecond, func() {
		fmt.Println("Notifying", *remote)
		conn, err := net.Dial("udp", *remote)
		if err != nil {
			panic(err)
		}
		if _, err := conn.Write([]byte("build")); err != nil {
			panic(err)
		}
		conn.Close()
	})
	t.Stop()

	for event := range watcher.Events {
		if filepath.Ext(event.Name) == ".go" {
			if event.Op == fsnotify.Write {
				t.Reset(500 * time.Millisecond)
			}
		}
	}
}
