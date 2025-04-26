// gredentures
package main

import (
	"gredentures/gredentures"
	"os"
)

func main() {
	os.Exit(gredentures.CLI(os.Args[1:]))
}
