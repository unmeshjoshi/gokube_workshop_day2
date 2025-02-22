package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gokube/pkg/registry"
	"gokube/pkg/scheduler"
	"gokube/pkg/storage"

	"github.com/spf13/cobra"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	etcdPort       int
	schedulingRate time.Duration
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "scheduler",
		Short: "Start the gokube scheduler",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runScheduler(); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.Flags().IntVar(&etcdPort, "etcd-port", 2379, "Port of the etcd server")
	rootCmd.Flags().DurationVar(&schedulingRate, "scheduling-rate", 10*time.Second, "How often to run the scheduling loop")

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runScheduler() error {
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)

	// Create etcd client
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
	podRegistry := registry.NewPodRegistry(store)
	nodeRegistry := registry.NewNodeRegistry(store)

	// Create and start the scheduler
	sched := scheduler.NewScheduler(podRegistry, nodeRegistry, schedulingRate)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go sched.Start(ctx)

	fmt.Printf("Scheduler started successfully\n")
	fmt.Printf("Connected to etcd at localhost:%d\n", etcdPort)
	fmt.Printf("Scheduling rate: %v\n", schedulingRate)

	<-stopCh
	fmt.Println("\nReceived shutdown signal. Stopping scheduler...")
	return nil
}
