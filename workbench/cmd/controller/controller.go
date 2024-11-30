package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"gokube/pkg/controller"
	"gokube/pkg/registry"
	"gokube/pkg/storage"

	"github.com/spf13/cobra"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	apiServerURL string
	etcdPort     int
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "controller",
		Short: "Start the gokube controller",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runController(); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.Flags().StringVar(&apiServerURL, "api-server", "localhost:8080", "URL of the API server")
	rootCmd.Flags().IntVar(&etcdPort, "etcd-port", 2379, "Port of the etcd server")

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runController() error {
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)

	cli, err := clientv3.New(clientv3.Config{
		Endpoints: []string{fmt.Sprintf("localhost:%d", etcdPort)},
	})
	if err != nil {
		return fmt.Errorf("failed to create etcd client: %v", err)
	}
	defer cli.Close()

	// Create etcd storage instance
	store := storage.NewEtcdStorage(cli)

	// Initialize registries with the etcd storage
	rsRegistry := registry.NewReplicaSetRegistry(store)
	podRegistry := registry.NewPodRegistry(store)

	rsController := controller.NewReplicaSetController(rsRegistry, podRegistry)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go rsController.Start(ctx)

	fmt.Println("Controller started successfully")

	<-stopCh
	fmt.Println("\nReceived shutdown signal. Stopping controller...")
	return nil
}
