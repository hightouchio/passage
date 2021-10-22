package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

const name = "passage"
var version = "0.0.0"

var (
	rootCmd = &cobra.Command{
		Use: "passage",
		Short: "ssh tunnel management server",
		Long: `
passage is a utility for programmatically creating and managing SSH tunnels.
`,
	}
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use: "version",
		Short: "Print the current version number of passage",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s v%s\n", name, version)
		},
	})

	rootCmd.AddCommand(serverCommand)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

