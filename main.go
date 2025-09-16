// main.go
package main

import (
	"project/cmd"
)

func main() {
	cmd.RegisterCommands(
		cmd.NewServeCommand(),
		cmd.NewPubSubConsumerCommand(),
	)

	cmd.Execute()
}
