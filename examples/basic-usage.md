# KubeStack-AI Usage Examples

This guide provides examples and best practices for using the KubeStack-AI command-line tool (`ksa`).

## 1. Quick Start

### Installation

First, clone the repository. Then, use the provided build script to compile the `ksa` binary for your system.

```bash
git clone https://github.com/kubestack-ai/kubestack-ai.git
cd kubestack-ai
./scripts/build.sh
```

This will create the `ksa` binary in the `./bin` directory. You can move this binary to a location in your `PATH` (e.g., `/usr/local/bin`) for easy access.

### Configuration

KubeStack-AI requires an LLM API key to function. The **recommended** way to provide this is via an environment variable, as it avoids storing secrets in plain text files.

**For OpenAI:**
```bash
export KSA_LLM_OPENAI_APIKEY="sk-..."
```

**For Gemini:**
```bash
export KSA_LLM_GEMINI_APIKEY="..."
```

Alternatively, you can place these keys in a configuration file located at `/etc/kubestack-ai/config.yaml` or `$HOME/.ksa.yaml`.

## 2. Common Commands

### Running a Diagnosis

The `diagnose` command is the core of KubeStack-AI. It runs a comprehensive analysis on a specified middleware instance.

```bash
# Diagnose a Redis instance named 'my-redis' running in the 'default' Kubernetes namespace
ksa diagnose redis my-redis -n default
```

**Example Output:**

The tool will show real-time progress and then display a final report.

```
 diagnóstico > Initialization     | Plugin loaded successfully.
 diagnóstico > Data Collection    | Data collection finished.
 diagnóstico > Analysis           | Analysis finished, found 2 issues.

--- Diagnosis Report ---
Status: Critical
Summary: Redis diagnosis complete. Found 2 potential issues.

Found 2 issues:
+----------+--------------------------------+-------------------------------------------------------------+
| SEVERITY |             TITLE              |                       RECOMMENDATION                        |
+----------+--------------------------------+-------------------------------------------------------------+
| Critical | Password Protection Disabled   | Set a strong password via the 'requirepass' configuration   |
|          |                                | option to secure the instance.                              |
+----------+--------------------------------+-------------------------------------------------------------+
| Warning  | High Memory Fragmentation      | A high fragmentation ratio can often be resolved by         |
|          |                                | restarting the Redis server. Ensure persistence is enabled  |
|          |                                | to avoid data loss.                                         |
+----------+--------------------------------+-------------------------------------------------------------+
```

### Asking the AI Assistant

Use the `ask` command for natural language queries. You can ask general questions or ask for clarification on a diagnosis.

```bash
ksa ask "What are the pros and cons of AOF persistence in Redis?"
```

**Example Streaming Output:**

```
🤖 KubeStack-AI: AOF (Append-Only File) persistence in Redis logs every write operation received by the server. This provides a higher level of durability compared to RDB snapshots.

**Pros:**
*   **Durability:** You can configure different `fsync` policies, allowing you to lose at most one second of data in the default configuration.
*   **Robustness:** The AOF file is an append-only log, making it less prone to corruption.

**Cons:**
*   **File Size:** The AOF file is typically larger than the equivalent RDB file.
*   **Performance:** Depending on the `fsync` policy, AOF can be slower than RDB, as it involves a write to disk on every operation.
```

### Applying an Automated Fix

The `fix` command allows you to apply automated fixes based on a diagnosis report. It is designed to be safe, requiring user review and confirmation.

```bash
# Suppose a diagnosis returned a report with ID 'diag-redis-12345'
# Use this ID to generate and review a fix plan:
ksa fix diag-redis-12345
```

**Example Interaction:**

The tool will first show you what it plans to do and ask for confirmation.

```
Fetching recommendations for diagnosis ID: diag-redis-12345
Generating execution plan...

--- [Execution Plan Review] ---
 Risk Level: High
 Description: Plan contains high-risk operations that will modify the running configuration.
 Steps to be executed:
  1. Set Redis Password
     └─ Command: `redis-cli CONFIG SET requirepass "a-new-strong-password"`
  2. Restart Redis Service
     └─ Command: `systemctl restart redis`

Do you want to execute this plan? [y/N]: y

Executing plan...
# The execution engine will now run, potentially asking for more confirmations on a per-step basis.
...
```

## 3. Troubleshooting

*   **Connection Errors:** If a diagnosis fails with a connection error, ensure the middleware instance is reachable from where you are running `ksa` and that any credentials in your configuration are correct.
*   **API Key Errors:** If `ask` commands fail, double-check that your `KSA_LLM_..._APIKEY` environment variable is set correctly and has been `export`ed in your shell session.

<!-- Personal.AI order the ending -->
