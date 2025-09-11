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

// ExecutionContext is the top-level context container for a single operation.
// It aggregates various specific contexts and is passed through the system to provide
// state and information to all components involved in processing a request.
type ExecutionContext struct {
	Session    *SessionContext    `json:"session" yaml:"session"`
	User       *UserContext       `json:"user" yaml:"user"`
	System     *SystemContext     `json:"system,omitempty" yaml:"system,omitempty"`
	Kubernetes *KubernetesContext `json:"kubernetes,omitempty" yaml:"kubernetes,omitempty"`
	Middleware *MiddlewareContext `json:"middleware,omitempty" yaml:"middleware,omitempty"`

	// GoContext is the standard Go context for handling cancellation, deadlines, and passing request-scoped values.
	// It is not serialized.
	GoContext context.Context `json:"-" yaml:"-"`
}

// SessionContext holds information about the current user session.
type SessionContext struct {
	SessionID string `json:"sessionId" yaml:"sessionId"`
	StartTime int64  `json:"startTime" yaml:"startTime"`
}

// UserContext holds information about the user executing the command.
type UserContext struct {
	Username    string            `json:"username" yaml:"username"`
	Permissions []string          `json:"permissions" yaml:"permissions"`
	Preferences map[string]string `json:"preferences,omitempty" yaml:"preferences,omitempty"`
}

// SystemContext holds information about the host system where the command is running or targeting.
type SystemContext struct {
	OS       string `json:"os" yaml:"os"`
	Arch     string `json:"arch" yaml:"arch"`
	Hostname string `json:"hostname" yaml:"hostname"`
	// Additional hardware and network info can be added here.
}

// KubernetesContext holds information about the Kubernetes cluster being targeted.
type KubernetesContext struct {
	ClusterName string         `json:"clusterName" yaml:"clusterName"`
	Kubeconfig  string         `json:"-" yaml:"-"` // Sensitive data, excluded from serialization.
	Namespace   string         `json:"namespace" yaml:"namespace"`
	Resources   []*K8sResource `json:"resources,omitempty" yaml:"resources,omitempty"`
}

// K8sResource represents a single discovered Kubernetes resource relevant to the context.
type K8sResource struct {
	Kind string `json:"kind" yaml:"kind"`
	Name string `json:"name" yaml:"name"`
	UID  string `json:"uid" yaml:"uid"`
}

// MiddlewareContext holds context specific to the middleware instance being diagnosed.
type MiddlewareContext struct {
	Type    string `json:"type" yaml:"type"`
	Version string `json:"version" yaml:"version"`
	// Other middleware-specific contextual information can be added here.
}

// Merge combines another ExecutionContext into the current one.
// Non-nil fields from the 'other' context will overwrite the corresponding fields in the receiver.
// This is useful for building up context as more information is discovered.
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

// Sanitize returns a deep copy of the context with sensitive information removed or redacted.
// This is useful for logging or exporting context data without leaking secrets.
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
