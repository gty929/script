package main

import (
	"github.com/bitfield/script"
)

// This program prints the first address that it pings successfully.

func main() {
	script.Stdin().ExecReduce("bash -c '[[ '{{.First}}' = 'None' ]] && ping -c 4 {{.Second}} >> ping.log && echo {{.Second}} || echo {{.First}}'", "None").Stdout()
}
