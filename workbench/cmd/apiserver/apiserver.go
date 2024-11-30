package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gokube/pkg/api/server"
	"gokube/pkg/storage"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/spf13/cobra"
)

var (
	address        string
	etcdPeerPort   int
	etcdClientPort int
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "apiserver",
		Short: "Start the gokube API server",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runAPIServer(); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.Flags().StringVar(&address, "address", ":8080", `The address to serve on (default ":8080")`)
	rootCmd.Flags().IntVar(&etcdPeerPort, "etcd-peer-port", 0, `The port to start etcd peer on (default random port)`)
	rootCmd.Flags().IntVar(&etcdClientPort, "etcd-client-port", 2379, `The port to start etcd client on (default 2379)`)

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runAPIServer() error {
	// Create a channel to handle shutdown signals
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)

	// Start embedded etcd
	etcdServer, port, err := storage.StartEmbeddedEtcdWithPort(etcdPeerPort, etcdClientPort)
	if err != nil {
		return fmt.Errorf("failed to start etcd: %v", err)
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{fmt.Sprintf("http://localhost:%d", port)},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to create etcd client: %v", err)
	}
	defer cli.Close()

	store := storage.NewEtcdStorage(cli)
	apiServer := server.NewAPIServer(store)

	fmt.Printf("Starting API server on %s\n", address)

	// Start the API server in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- apiServer.Start(address)
	}()

	// Wait for either an error or shutdown signal
	select {
	case err := <-errCh:
		storage.StopEmbeddedEtcd(etcdServer)
		return err
	case <-stopCh:
		fmt.Println("\nReceived shutdown signal. Stopping services...")
		storage.StopEmbeddedEtcd(etcdServer)
		return nil
	}
}
