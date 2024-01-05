package main

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"net"
	"strconv"
	"time"
)

var (
	portRangeCommand = &cobra.Command{
		Use:   "port-range",
		Short: "passage port-range identifies a range of ports that are available for use by passage on a system",
		RunE:  testPortRange,
	}
)

var testRangeDuration time.Duration

func init() {
	rootCmd.AddCommand(portRangeCommand)
	portRangeCommand.Flags().DurationVarP(&testRangeDuration, "duration", "d", 30*time.Second, "The duration to run the port range test for")
}

// testPortRange tests the available passage port range
func testPortRange(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), testRangeDuration)
	defer cancel()

	fmt.Printf("Running port range test for %s\n", testRangeDuration.String())

	seen := 0
	maxPort := 0
	minPort := 65535

	for ctx.Err() == nil {
		l, err := net.ListenTCP("tcp", &net.TCPAddr{
			IP:   net.ParseIP("0.0.0.0"),
			Port: 0,
		})
		if err != nil {
			return fmt.Errorf("error listening: %w", err)
		}

		_, portStr, _ := net.SplitHostPort(l.Addr().String())
		port, _ := strconv.Atoi(portStr)
		l.Close()

		seen += 1
		if port > maxPort {
			maxPort = port
		}
		if port < minPort {
			minPort = port
		}
	}

	fmt.Printf("Finished port range test. Tested %d ports. Min: %d, Max: %d\n", seen, minPort, maxPort)
	return nil
}
