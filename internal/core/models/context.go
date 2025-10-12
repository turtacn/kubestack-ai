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

package models

import "context"

// ExecutionContext is the top-level data container for a single operation. It
// aggregates various specific contexts (user, system, Kubernetes, etc.) and is
// passed through the system to provide state and information to all components
// involved in processing a request.
type ExecutionContext struct {
	// Session holds information about the current user session.
	Session *SessionContext `json:"session" yaml:"session"`
	// User holds information about the user executing the command.
	User *UserContext `json:"user" yaml:"user"`
	// System holds information about the host system being targeted.
	System *SystemContext `json:"system,omitempty" yaml:"system,omitempty"`
	// Kubernetes holds information about the Kubernetes cluster being targeted.
	Kubernetes *KubernetesContext `json:"kubernetes,omitempty" yaml:"kubernetes,omitempty"`
	// Middleware holds context specific to the middleware instance being diagnosed.
	Middleware *MiddlewareContext `json:"middleware,omitempty" yaml:"middleware,omitempty"`

	// GoContext is the standard Go context for handling cancellation, deadlines, and
	// passing request-scoped values. It is intentionally not serialized.
	GoContext context.Context `json:"-" yaml:"-"`
}

// SessionContext holds information about the current user session, which is
// useful for tracking and auditing.
type SessionContext struct {
	// SessionID is a unique identifier for the user's session.
	SessionID string `json:"sessionId" yaml:"sessionId"`
	// StartTime is the Unix timestamp when the session began.
	StartTime int64 `json:"startTime" yaml:"startTime"`
}

// UserContext holds information about the user executing the command, including
// their identity and permissions.
type UserContext struct {
	// Username is the identifier for the user.
	Username string `json:"username" yaml:"username"`
	// Permissions is a list of permissions the user has, which can be used for RBAC checks.
	Permissions []string `json:"permissions" yaml:"permissions"`
	// Preferences holds any user-specific preferences that might affect command behavior.
	Preferences map[string]string `json:"preferences,omitempty" yaml:"preferences,omitempty"`
}

// SystemContext holds information about the host operating system where the
// command is running or targeting.
type SystemContext struct {
	// OS is the name of the operating system (e.g., "linux", "darwin").
	OS string `json:"os" yaml:"os"`
	// Arch is the system's architecture (e.g., "amd64", "arm64").
	Arch string `json:"arch" yaml:"arch"`
	// Hostname is the hostname of the system.
	Hostname string `json:"hostname" yaml:"hostname"`
}

// KubernetesContext holds information about the Kubernetes cluster being targeted,
// including connection details and a list of relevant discovered resources.
type KubernetesContext struct {
	// ClusterName is the name of the Kubernetes cluster.
	ClusterName string `json:"clusterName" yaml:"clusterName"`
	// Kubeconfig is the path to the kubeconfig file. It is excluded from serialization for security.
	Kubeconfig string `json:"-" yaml:"-"`
	// Namespace is the specific namespace being targeted within the cluster.
	Namespace string `json:"namespace" yaml:"namespace"`
	// Resources is a list of Kubernetes resources relevant to the current operation.
	Resources []*K8sResource `json:"resources,omitempty" yaml:"resources,omitempty"`
}

// K8sResource represents a single discovered Kubernetes resource, providing
// enough information to uniquely identify it within its namespace.
type K8sResource struct {
	// Kind is the type of the resource (e.g., "Pod", "Service", "Deployment").
	Kind string `json:"kind" yaml:"kind"`
	// Name is the name of the resource instance.
	Name string `json:"name" yaml:"name"`
	// UID is the unique identifier for the resource provided by Kubernetes, which is
	// guaranteed to be unique across time and namespaces.
	UID string `json:"uid" yaml:"uid"`
}

// MiddlewareContext holds context specific to the middleware instance being
// diagnosed, such as its type and version.
type MiddlewareContext struct {
	// Type is the type of the middleware (e.g., "Redis", "MySQL").
	Type string `json:"type" yaml:"type"`
	// Version is the detected version of the middleware instance.
	Version string `json:"version" yaml:"version"`
}

// Merge combines another ExecutionContext into the current one. Non-nil fields
// from the `other` context will overwrite the corresponding fields in the receiver.
// This is useful for progressively building up the context as more information is
// discovered during an operation.
//
// Parameters:
//   other (*ExecutionContext): The context to merge into the current one.
func (ec *ExecutionContext) Merge(other *ExecutionContext) {
	if other.Session != nil {
		ec.Session = other.Session
	}
	if other.User != nil {
		ec.User = other.User
	}
	if other.System != nil {
		ec.System = other.System
	}
	if other.Kubernetes != nil {
		ec.Kubernetes = other.Kubernetes
	}
	if other.Middleware != nil {
		ec.Middleware = other.Middleware
	}
	if other.GoContext != nil {
		ec.GoContext = other.GoContext
	}
}

// Sanitize returns a deep copy of the context with sensitive information, such
// as the Kubeconfig, removed or redacted. This is a crucial utility for logging
// or exporting context data without leaking secrets.
//
// Returns:
//   *ExecutionContext: A new, sanitized instance of the ExecutionContext.
func (ec *ExecutionContext) Sanitize() *ExecutionContext {
	// Create a shallow copy first
	sanitized := *ec

	// Deep copy and sanitize Kubernetes context if it exists
	if ec.Kubernetes != nil {
		k8sCtxCopy := *ec.Kubernetes
		k8sCtxCopy.Kubeconfig = "[REDACTED]"
		sanitized.Kubernetes = &k8sCtxCopy
	}

	// Add other sanitization logic here for other sensitive fields in the future.

	return &sanitized
}

//Personal.AI order the ending
