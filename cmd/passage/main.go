package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

const name = "passage"
const version = "0.0.1"

var (
	rootCmd = &cobra.Command{
		Use: "passage",
		Short: "ssh tunnel management server",
		Long: `
passage is a utility for programmatically creating and managing SSH tunnels.
The chief use case is to serve as a secure bridge between SaaS providers and resources that need to be accessed within customer environments.
`,
		Run: runServer,
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
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

