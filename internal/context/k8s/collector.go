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

package k8s

import (
	"context"
	"fmt"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Collector defines the interface for components responsible for gathering
// contextual information about a workload running in a Kubernetes cluster.
type Collector interface {
	// Collect gathers high-level information about Kubernetes resources (such as
	// Pods, Services, and Deployments) that are related to a specific workload,
	// identified by its namespace and a label selector.
	//
	// Parameters:
	//   ctx (context.Context): The context for the API requests.
	//   namespace (string): The namespace of the target workload.
	//   labelSelector (string): The label selector to identify the workload's resources.
	//
	// Returns:
	//   *models.KubernetesContext: A struct containing the collected resource information.
	//   error: An error if the collection process fails.
	Collect(ctx context.Context, namespace, labelSelector string) (*models.KubernetesContext, error)
}

// collector is the concrete implementation of the Collector interface.
type collector struct {
	log    logger.Logger
	client *Client
}

// NewCollector creates a new Kubernetes context collector that uses the provided
// client to interact with the Kubernetes API.
//
// Parameters:
//   client (*Client): An initialized Kubernetes client.
//
// Returns:
//   Collector: A new instance of the Kubernetes collector.
//   error: An error if the provided client is nil.
func NewCollector(client *Client) (Collector, error) {
	if client == nil {
		return nil, fmt.Errorf("k8s client cannot be nil")
	}
	return &collector{
		log:    logger.NewLogger("k8s-collector"),
		client: client,
	}, nil
}

// Collect implements the Collector interface. It gathers information about key
// Kubernetes resources (currently Pods, Services, and Deployments) that match the
// provided namespace and label selector. It aggregates this information into a
// `KubernetesContext` model for use by the diagnosis engine.
//
// Parameters:
//   ctx (context.Context): The context for the Kubernetes API requests.
//   namespace (string): The namespace to search for resources in.
//   labelSelector (string): A label selector string to identify the relevant resources.
//
// Returns:
//   *models.KubernetesContext: A pointer to a struct containing the collected resource information.
//   error: An error if there is a failure in listing the core resources (like Pods).
func (c *collector) Collect(ctx context.Context, namespace, labelSelector string) (*models.KubernetesContext, error) {
	c.log.Infof("Collecting Kubernetes context for namespace '%s' with selector '%s'", namespace, labelSelector)

	k8sCtx := &models.KubernetesContext{
		Namespace: namespace,
		Resources: make([]*models.K8sResource, 0),
	}

	// Collect Pods
	pods, err := c.client.ListPods(ctx, namespace, labelSelector)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}
	for _, pod := range pods.Items {
		k8sCtx.Resources = append(k8sCtx.Resources, &models.K8sResource{
			Kind: "Pod",
			Name: pod.Name,
			UID:  string(pod.UID),
		})
		// TODO: Collect recent events related to the pod.
		// `c.client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{FieldSelector: "involvedObject.uid=" + string(pod.UID)})`
		// TODO: Collect logs from the pod's containers.
		// `c.client.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{...})`
	}

	// Collect Services
	// This assumes the service has the same labels as the pods, which is a common pattern.
	services, err := c.client.clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		c.log.Warnf("Failed to list services: %v", err)
	} else {
		for _, svc := range services.Items {
			k8sCtx.Resources = append(k8sCtx.Resources, &models.K8sResource{
				Kind: "Service",
				Name: svc.Name,
				UID:  string(svc.UID),
			})
		}
	}

	// Collect Deployments
	deployments, err := c.client.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		c.log.Warnf("Failed to list deployments: %v", err)
	} else {
		for _, dep := range deployments.Items {
			k8sCtx.Resources = append(k8sCtx.Resources, &models.K8sResource{
				Kind: "Deployment",
				Name: dep.Name,
				UID:  string(dep.UID),
			})
		}
	}

	// TODO: Collect other important resources like StatefulSets, ConfigMaps, Secrets, and PersistentVolumeClaims.
	// TODO: Collect resource usage metrics (CPU, memory) if the Kubernetes metrics-server is available.
	// This would require a `metrics.k8s.io/v1beta1` client.

	c.log.Infof("Collected context for %d Kubernetes resources.", len(k8sCtx.Resources))
	return k8sCtx, nil
}

//Personal.AI order the ending
