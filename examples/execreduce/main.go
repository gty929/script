package main

import (
	"github.com/bitfield/script"
)

// This program prints out the line that has the most words.

func main() {
	script.Stdin().ExecReduce("bash -c '[[ $(echo {{.First}} | wc -w) -ge $(echo {{.Second}} | wc -w) ]] && echo {{.First}} || echo {{.Second}}'", "").Stdout()
}
