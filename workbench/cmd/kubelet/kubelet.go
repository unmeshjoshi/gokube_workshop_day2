package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gokube/pkg/kubelet"
)

var (
	nodeName     string
	apiServerURL string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "kubelet",
		Short: "Start the gokube Kubelet",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runKubelet(); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.Flags().StringVar(&nodeName, "node-name", "", "The name of the node")
	rootCmd.Flags().StringVar(&apiServerURL, "api-server-url", "localhost:8080", "The URL of the API server")

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runKubelet() error {
	k, err := kubelet.NewKubelet(nodeName, apiServerURL)
	if err != nil {
		return fmt.Errorf("failed to create kubelet: %v", err)
	}

	if err := k.Start(); err != nil {
		return fmt.Errorf("failed to start kubelet: %v", err)
	}

	// Block forever
	select {}
}
