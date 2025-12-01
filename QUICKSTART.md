# Quickstart

This guide provides the essential steps to build and run KubeStack-AI.

## Prerequisites

- Go 1.18 or later
- Git

## 1. Clone the Repository

Clone the project from GitHub:

```sh
git clone https://github.com/kubestack-ai/kubestack-ai.git
cd kubestack-ai
```

## 2. Build the Binary

Compile the `ksa` command-line tool from the source:

```sh
go build -o ksa cmd/ksa/main.go
```

This will create a binary named `ksa` in the root of the project directory.

## 3. Start the Server

Run the KubeStack-AI server. This will start the API and web console on `http://localhost:8080`.

```sh
./ksa server start
```

You should see log output indicating that the server has started successfully.

## 4. Run a Diagnosis

In a separate terminal, you can use the `ksa` CLI to run a diagnosis on a middleware component.

For example, to diagnose a Redis instance running on `localhost:6379`:

```sh
./ksa diagnose redis --instance localhost:6379
```

This will trigger a diagnosis and display the results in your terminal.
