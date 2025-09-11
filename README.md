# KubeStack-AI

[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)](#)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](./LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubestack-ai/kubestack-ai)](https://goreportcard.com/report/github.com/kubestack-ai/kubestack-ai)

**KubeStack-AI is an intelligent SRE assistant designed to help you diagnose, analyze, and fix issues with your middleware infrastructure using the power of Large Language Models.**

Whether your Redis is slow, your Kafka cluster is unbalanced, or your MySQL database is throwing errors, the `ksa` command-line tool provides a unified, AI-powered interface to quickly get to the root of the problem and find a solution.

## ✨ Features

*   **AI-Driven Diagnosis**: Use natural language to ask "why is my database slow?" and get an analysis backed by real-time data and a comprehensive knowledge base.
*   **Unified CLI**: A single, consistent command-line tool (`ksa`) to manage and diagnose all your supported middleware (Redis, MySQL, Kafka, Elasticsearch, and more).
*   **Pluggable Architecture**: Easily extend KubeStack-AI by creating your own plugins for internal tools or currently unsupported middleware.
*   **Automated Fixes**: Generate execution plans for automated fixes and review them before applying them safely and interactively from your terminal.
*   **Multi-Environment**: Works seamlessly across Kubernetes, Docker, and bare-metal environments.

## 🚀 Quick Start

1.  **Install:**
    ```bash
    # Clone the repository and build the binary
    git clone https://github.com/kubestack-ai/kubestack-ai.git
    cd kubestack-ai
    ./scripts/build.sh
    ```
    This will create the `ksa` binary in the `./bin` directory for your platform.

2.  **Configure:**
    Set your LLM provider's API key. This is required for the `ask` command and AI-enhanced diagnostics.
    ```bash
    # For OpenAI
    export KSA_LLM_OPENAI_APIKEY="sk-..."
    ```

3.  **Run a Diagnosis:**
    ```bash
    # Run a diagnosis on a Redis instance
    # Note: This example assumes a locally running Redis server.
    ./bin/ksa-linux-amd64 diagnose redis localhost
    ```

4.  **Ask a Question:**
    ```bash
    # Ask the AI assistant for help
    ./bin/ksa-linux-amd64 ask "what causes high memory fragmentation in redis?"
    ```

For more detailed examples and use cases, see the [**Basic Usage Guide**](./examples/basic-usage.md).

## 🏛️ Architecture

KubeStack-AI is built with a modular, layered architecture designed for extensibility and maintainability. For a detailed overview of the components and data flow, please see the [**Architecture Document**](./docs/architecture.md).

## 🤝 Contributing

We welcome contributions of all kinds! Please see our [**Contributing Guide**](./CONTRIBUTING.md) for more details on how to get started with setting up your development environment, our coding standards, and the pull request process.

## 📄 License

This project is licensed under the Apache 2.0 License. See the [LICENSE](./LICENSE) file for details.

<!-- Personal.AI order the ending -->
