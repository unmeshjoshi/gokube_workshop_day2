# GoKube: A Miniature Kubernetes-like Container Orchestrator
GoKube is an educational project that implements a simplified version of a container orchestrator, inspired by Kubernetes. This project is designed to teach the concepts of distributed system design using a Kubernetes-like system as an example.

## Project Overview

GoKube is built in Go and aims to demonstrate key concepts of container orchestration such as:

- Container scheduling
- Service discovery
- Load balancing
- State management
- Scaling

By implementing a miniature version of Kubernetes, this project provides hands-on experience with the fundamental principles of distributed systems and container orchestration.

## Prerequisites

- Docker installed on your system
- Basic understanding of Go programming language
- Familiarity with container concepts

## Building the Docker Image

To build the Docker image for development and testing, run the following command from the root directory of this project:

```bash
docker build --no-cache -t gokube-builder .
```
## Running the Development Environment

To run a container from the built image, use the following command:

```bash
docker run --ulimit nofile=65536:65536 --privileged --rm -it -v $(pwd):/app gokube-builder:latest
```

This command does the following:

- `--ulimit nofile=65536:65536`: Sets the ulimit for open files to 65536.
- `--privileged`: Gives extended privileges to this container, necessary for running Docker inside Docker.
- `--rm`: Automatically removes the container when it exits.
- `-it`: Runs the container interactively and allocates a pseudo-TTY.
- `-v $(pwd):/app`: Mounts the current directory to `/app` in the container.
- `gokube-builder:latest`: Specifies the image to use.

Once the container is running, you can build the project using the following command:

```bash
make ci
```
This command will run the continuous integration build process.

### Running Go Commands

Inside the container, you can run all standard Go commands. For example, to run tests for the kubelet package:
```bash
go test -v ./pkg/kubelet
```

You can use similar commands to run tests for other packages, build the project, or perform any other Go-related tasks.

## Project Structure

The GoKube project is organized into several key directories:

```
gokube/
├── cmd/
│   ├── apiserver/
│   ├── controller-manager/
│   ├── kubelet/
│   └── scheduler/
├── pkg/
│   ├── api/
│   ├── controller/
│   ├── kubelet/
│   ├── scheduler/
│   └── util/
├── internal/
│   └── ...
├── test/
│   └── ...
├── docs/
│   └── ...
├── Dockerfile
├── go.mod
├── go.sum
└── README.md

- `pkg/`: Contains the core packages used throughout the project.
  - `api/`: Defines the API objects and clients.
  - `controller/`: Implements the controllers for managing the system state.
  - `kubelet/`: Implements the kubelet functionality.
  - `scheduler/`: Implements the scheduling of pods onto nodes.
  - `util/`: Contains utility functions used across the project.

- `internal/`: Houses internal packages not intended for use outside the project.

- `test/`: Contains integration and end-to-end tests.

- `docs/`: Project documentation.

This structure mimics Kubernetes' organization, providing a familiar layout for those acquainted with the Kubernetes codebase while simplifying it for educational purposes.


## Components

- API Server: Handles API requests and manages the system's state
- Kubelet: Manages containers on individual nodes
- Etcd: Distributed key-value store for system state (embedded)

## Current Features

- Basic container management (create, start, stop)
- Simple pod creation and management
- Rudimentary node management

## Learning Objectives

By working with this project, you will gain insights into:

1. The architecture of container orchestration systems
2. Distributed system design principles
3. Container lifecycle management
4. Network management in containerized environments
5. Challenges in distributed state management
6. Scaling and load balancing in distributed systems

## Acknowledgments

- Kubernetes project for inspiration
- Patterns Of Distributed Systems for design principles
```

## Setting Up the Development Environment

To set up the development environment, you have two options: using Devbox or installing the necessary tools individually.

### Option 1: Using Devbox

1. Install Devbox by following the instructions on the [Devbox GitHub page](https://github.com/jetify-com/devbox).
2. Once Devbox is installed, navigate to the root directory of this project and run:

  ```bash
  devbox shell
  ```

This will automatically install the required packages (`goreleaser` and `lima`) and set up the environment.

### Option 2: Installing Tools Individually

If you prefer not to use Devbox, you can install the required tools using Homebrew:

1. Install `goreleaser`:

  ```bash
  brew install goreleaser
  ```

2. Install `lima`:

  ```bash
  brew install lima
  ```

After installing these tools, you can proceed with the rest of the setup instructions.

## Managing the VM

This setup uses the `workbench/debian-12.yaml` configuration and assumes you are running it on an M series MacBook. If you are using a non-M series MacBook, please ask the instructor to provide the necessary instructions.

When the VM is started, it will have all the necessary tools installed, including Docker and etcd. Additionally, the path to the GoKube binary is set, allowing you to run the apiserver, controller, and kubelet directly from the VM shell.

The Makefile includes commands to manage a Lima VM for running GoKube. Here are the instructions to start, stop, delete, and access the VM shell.

### Starting the VM

To start the VM, run the following command:

```bash
make start/vm
```

This command will start a Lima instance named `gokube` using the configuration specified in `workbench/debian-12.yaml`.

### Stopping the VM

To stop the VM, run:

```bash
make stop/vm
```

This command will stop the `gokube` Lima instance.

### Deleting the VM

To delete the VM, use:

```bash
make delete/vm
```

This command will delete the `gokube` Lima instance.

### Accessing the VM Shell

To access the shell of the running VM, execute:

```bash
make shell/vm
```

This command will open a shell in the `gokube` Lima instance, allowing you to interact with the VM directly.