package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "myapp",
	Short: "MyApp CLI",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func RegisterCommands(cmds ...*cobra.Command) {
	for _, c := range cmds {
		rootCmd.AddCommand(c)
	}
}
