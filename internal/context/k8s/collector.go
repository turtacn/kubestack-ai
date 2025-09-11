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

// Collector is the interface for gathering contextual information from a Kubernetes cluster.
type Collector interface {
	// Collect gathers information about Kubernetes resources related to a given workload,
	// identified by its namespace and a label selector.
	Collect(ctx context.Context, namespace, labelSelector string) (*models.KubernetesContext, error)
}

// collector is the concrete implementation of the Collector interface.
type collector struct {
	log    logger.Logger
	client *Client
}

// NewCollector creates a new Kubernetes context collector.
func NewCollector(client *Client) (Collector, error) {
	if client == nil {
		return nil, fmt.Errorf("k8s client cannot be nil")
	}
	return &collector{
		log:    logger.NewLogger("k8s-collector"),
		client: client,
	}, nil
}

// Collect gathers information about Kubernetes resources (Pods, Services, Deployments, etc.)
// that match the provided namespace and label selector.
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
