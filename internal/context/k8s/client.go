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

// Package k8s provides components for interacting with a Kubernetes cluster.
package k8s

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Client is a wrapper around the official Kubernetes client-go library.
// It simplifies common operations like loading configuration and fetching resources.
type Client struct {
	log       logger.Logger
	clientset kubernetes.Interface
	config    *rest.Config
}

// NewClient creates a new Kubernetes client. It automatically handles loading the
// kubeconfig from standard locations (~/.kube/config) or from an in-cluster service account
// if the application is running inside a pod.
func NewClient() (*Client, error) {
	log := logger.NewLogger("k8s-client")

	// First, try to load configuration from within the cluster. This is the
	// standard way for applications running as pods to talk to the API server.
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Debugf("Could not load in-cluster config: %v. Falling back to kubeconfig file.", err)

		// If in-cluster config fails, fall back to loading from a kubeconfig file.
		var kubeconfigPath string
		if home := homedir.HomeDir(); home != "" {
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		} else {
			return nil, fmt.Errorf("home directory not found, cannot locate kubeconfig")
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("could not load kubeconfig from '%s': %w", kubeconfigPath, err)
		}
		log.Infof("Successfully loaded kubeconfig from %s", kubeconfigPath)
	} else {
		log.Info("Successfully loaded in-cluster Kubernetes config.")
	}

	// The client-go library handles connection pooling, retries, and performance optimization internally.
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	return &Client{
		log:       log,
		clientset: clientset,
		config:    config,
	}, nil
}

// --- Resource Accessor Methods ---

// GetPod fetches a specific Pod resource by name and namespace.
func (c *Client) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	return c.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
}

// ListPods lists all Pods in a namespace, optionally filtered by a label selector.
func (c *Client) ListPods(ctx context.Context, namespace, labelSelector string) (*corev1.PodList, error) {
	return c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
}

// GetDeployment fetches a specific Deployment resource by name and namespace.
func (c *Client) GetDeployment(ctx context.Context, namespace, name string) (*appsv1.Deployment, error) {
	return c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
}

// GetService fetches a specific Service resource by name and namespace.
func (c *Client) GetService(ctx context.Context, namespace, name string) (*corev1.Service, error) {
	return c.clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
}

// TODO: Implement methods for other common resource types (StatefulSets, ConfigMaps, etc.).
// TODO: Implement methods for Custom Resources (CRDs) using a dynamic client (`dynamic.NewForConfig`).
// TODO: Implement RBAC checks using `authorizationv1.SelfSubjectAccessReview`.
// TODO: Implement event listening by creating and using an informer from `tools/cache`.

//Personal.AI order the ending
