package main

import (
	"fmt"

	"github.com/freman/tbuild"
)

var config = struct {
	Listen string
	Build  []string
}{
	Listen: fmt.Sprintf(":%d", tbuild.DefaultPort),
	Build:  []string{"go", "build"},
}
