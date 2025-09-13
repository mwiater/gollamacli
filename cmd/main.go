// cmd/main.go
package main

import cmd "github.com/mwiater/gollamacli/cmd/gollamacli"

// main starts the gollamacli CLI application by delegating to the
// cobra root command defined in the gollamacli package. It does not
// take any arguments and does not return a value.
func main() {
	cmd.Execute()
}
