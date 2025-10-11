// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package constants defines global constants used throughout the KubeStack-AI application.
// This ensures consistency and avoids magic numbers or strings in the code.
package constants

import "time"

// --- Project Information ---

// AppName is the official name of the application.
const AppName = "KubeStack-AI"

// CliAppName is the name of the command-line executable.
const CliAppName = "ksa"

// --- Default Configurations ---

// DefaultTimeout is the default timeout for most operations.
const DefaultTimeout = 30 * time.Second

// DefaultRetryCount is the default number of retries for failed operations.
const DefaultRetryCount = 3

// DefaultCacheSize is the default size for in-memory caches.
const DefaultCacheSize = 1024

// --- File System Paths ---

// DefaultConfigPath is the default directory path for configuration files.
const DefaultConfigPath = "/etc/kubestack-ai/"

// DefaultPluginDir is the default directory for storing plugins.
const DefaultPluginDir = "/var/lib/kubestack-ai/plugins/"

// DefaultLogDir is the default directory for log files.
const DefaultLogDir = "/var/log/kubestack-ai/"

// DefaultDataDir is the default directory for application data, including knowledge base indexes.
const DefaultDataDir = "/var/lib/kubestack-ai/data/"

// DefaultReportDir is the default directory for storing diagnosis reports.
const DefaultReportDir = "/var/lib/kubestack-ai/reports/"

// --- Networking ---

// DefaultConnectTimeout is the default timeout for establishing network connections.
const DefaultConnectTimeout = 5 * time.Second

// DefaultReadTimeout is the default timeout for reading from a network connection.
const DefaultReadTimeout = 15 * time.Second

// DefaultWriteTimeout is the default timeout for writing to a network connection.
const DefaultWriteTimeout = 15 * time.Second

// --- Plugin System ---

// PluginFileExtension is the expected file extension for dynamically loaded Go plugins.
const PluginFileExtension = ".so"

// PluginConfigFormat is the default configuration file format for plugins.
const PluginConfigFormat = "yaml"

// --- Diagnosis Engine ---

// DefaultDiagnosisTimeout is the maximum duration for a single diagnosis run.
const DefaultDiagnosisTimeout = 5 * time.Minute

// DefaultConcurrentDiagnoses is the default number of diagnoses that can run in parallel.
const DefaultConcurrentDiagnoses = 5

// DefaultDiagnosisCacheTTL is the default time-to-live for diagnosis results in the cache.
const DefaultDiagnosisCacheTTL = 10 * time.Minute

// --- LLM Integration ---

// PromptTemplateDiagnosis is the identifier for the standard diagnosis prompt template.
const PromptTemplateDiagnosis = "diagnosis_v1"

// DefaultMaxTokens is the default maximum number of tokens to generate in an LLM response.
const DefaultMaxTokens = 4096

// DefaultLLMTimeout is the default timeout for API calls to the LLM service.
const DefaultLLMTimeout = 60 * time.Second

// --- Knowledge Base ---

// DefaultVectorDimension is the default dimension for text embedding vectors.
// This is a common dimension for models like Sentence-BERT.
const DefaultVectorDimension = 768

// DefaultRetrievalCount is the default number of documents to retrieve from the knowledge base for a query.
const DefaultRetrievalCount = 5

// DefaultSimilarityThreshold is the minimum similarity score for a document to be considered relevant.
const DefaultSimilarityThreshold = 0.75

// --- CLI ---

const (
	// CommandDiagnose is the name of the diagnose command.
	CommandDiagnose = "diagnose"
	// CommandAsk is the name of the ask command.
	CommandAsk = "ask"
	// CommandStatus is the name of the status command.
	CommandStatus = "status"
	// CommandPlugin is the name of the plugin command.
	CommandPlugin = "plugin"
	// CommandFix is the name of the fix command.
	CommandFix = "fix"
	// CommandConfig is the name of the config command.
	CommandConfig = "config"
)

// OutputFormatTable is the identifier for table-based CLI output.
const OutputFormatTable = "table"

// OutputFormatJSON is the identifier for JSON-based CLI output.
const OutputFormatJSON = "json"

// OutputFormatYAML is the identifier for YAML-based CLI output.
const OutputFormatYAML = "yaml"

//Personal.AI order the ending
