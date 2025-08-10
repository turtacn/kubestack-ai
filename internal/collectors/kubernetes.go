package collectors

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/turtacn/kubestack-ai/internal/errors"
	"github.com/turtacn/kubestack-ai/internal/logging"
	"github.com/turtacn/kubestack-ai/internal/models"
)

// k8sCollector Kubernetes采集实现。k8sCollector implements Collector for Kubernetes.
type k8sCollector struct {
	clientset     *kubernetes.Clientset
	namespace     string
	labelSelector string
}

// NewK8sCollector 创建Kubernetes采集器。NewK8sCollector creates Kubernetes collector.
func NewK8sCollector(cs *kubernetes.Clientset, namespace string, labelSelector string) Collector {
	return &k8sCollector{
		clientset:     cs,
		namespace:     namespace,
		labelSelector: labelSelector,
	}
}

// GetInstanceStatus 获取Pod状态。Get pod status.
func (c *k8sCollector) GetInstanceStatus(ctx context.Context) (string, error) {
	logging.Logger.Debugf("Getting pod status in namespace %s with labels %s", c.namespace, c.labelSelector)

	pods, err := c.clientset.CoreV1().Pods(c.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: c.labelSelector,
	})
	if err != nil {
		logging.Logger.Errorf("Failed to list pods: %v", err)
		return "", errors.ErrDataCollectionFailed
	}

	if len(pods.Items) == 0 {
		return "no pods found", nil
	}

	// 汇总所有Pod的状态。Aggregate status from all pods.
	statusCounts := make(map[v1.PodPhase]int)
	for _, pod := range pods.Items {
		statusCounts[pod.Status.Phase]++
	}

	// 确定整体状态。Determine overall status.
	if statusCounts[v1.PodRunning] == len(pods.Items) {
		return "running", nil
	} else if statusCounts[v1.PodPending] > 0 {
		return "pending", nil
	} else if statusCounts[v1.PodFailed] > 0 {
		return "failed", nil
	}

	return string(pods.Items[0].Status.Phase), nil
}

// GetResourceUsage 获取资源使用。Get resource usage.
func (c *k8sCollector) GetResourceUsage(ctx context.Context) (models.Metrics, error) {
	logging.Logger.Debugf("Getting resource usage in namespace %s", c.namespace)

	pods, err := c.clientset.CoreV1().Pods(c.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: c.labelSelector,
	})
	if err != nil {
		logging.Logger.Errorf("Failed to list pods for resource usage: %v", err)
		return nil, errors.ErrDataCollectionFailed
	}

	metrics := models.Metrics{
		"pod_count": len(pods.Items),
		"cpu": map[string]interface{}{
			"requested": 0,
			"used":      0,
		},
		"memory": map[string]interface{}{
			"requested": 0,
			"used":      0,
		},
	}

	// 计算请求的资源。Calculate requested resources.
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			if cpu, ok := container.Resources.Requests[v1.ResourceCPU]; ok {
				metrics["cpu"].(map[string]interface{})["requested"] =
					metrics["cpu"].(map[string]interface{})["requested"].(int64) +
						cpu.MilliValue()
			}
			if memory, ok := container.Resources.Requests[v1.ResourceMemory]; ok {
				metrics["memory"].(map[string]interface{})["requested"] =
					metrics["memory"].(map[string]interface{})["requested"].(int64) +
						memory.Value()
			}
		}
	}

	// 这里应该从metrics-server获取实际使用的资源。Here we should get actual usage from metrics-server.
	// 简化示例：假设使用了请求资源的75%。Simplified example: assume 75% of requested resources used.
	metrics["cpu"].(map[string]interface{})["used"] =
		metrics["cpu"].(map[string]interface{})["requested"].(int64) * 75 / 100
	metrics["memory"].(map[string]interface{})["used"] =
		metrics["memory"].(map[string]interface{})["requested"].(int64) * 75 / 100

	return metrics, nil
}

// GetLogs 获取日志。Get logs.
func (c *k8sCollector) GetLogs(ctx context.Context, since time.Duration) (models.Logs, error) {
	logging.Logger.Debugf("Getting logs from namespace %s", c.namespace)

	pods, err := c.clientset.CoreV1().Pods(c.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: c.labelSelector,
	})
	if err != nil {
		logging.Logger.Errorf("Failed to list pods for logs: %v", err)
		return nil, errors.ErrDataCollectionFailed
	}

	logs := models.Logs{}
	for _, pod := range pods.Items {
		// 获取第一个容器的日志。Get logs from first container.
		if len(pod.Spec.Containers) == 0 {
			continue
		}

		logReq := c.clientset.CoreV1().Pods(c.namespace).GetLogs(pod.Name, &v1.PodLogOptions{
			SinceSeconds: int64(since.Seconds()),
		})

		logStream, err := logReq.Stream(ctx)
		if err != nil {
			logging.Logger.Warnf("Failed to get logs from pod %s: %v", pod.Name, err)
			continue
		}
		defer logStream.Close()

		// 简化示例：实际应读取logStream内容。Simplified example: should read logStream in real implementation.
		logs = append(logs, models.LogEntry{
			Timestamp: time.Now(),
			Level:     "info",
			Message:   "Sample log from pod " + pod.Name,
		})
	}

	return logs, nil
}

// GetNetworkInfo 获取网络信息。Get network information.
func (c *k8sCollector) GetNetworkInfo(ctx context.Context) (models.Metrics, error) {
	// 简化实现：实际应获取网络相关指标。Simplified implementation.
	return models.Metrics{
		"dns_policy": "ClusterFirst",
		"service": map[string]interface{}{
			"name": "middleware-service",
			"type": "ClusterIP",
			"ports": []map[string]interface{}{
				{"name": "tcp", "port": 80, "target_port": 8080},
			},
		},
	}, nil
}

// GetEvents 获取事件。Get events.
func (c *k8sCollector) GetEvents(ctx context.Context, since time.Duration) ([]string, error) {
	// 简化实现：实际应从Kubernetes API获取事件。Simplified implementation.
	return []string{
		"Event: Pod scheduled",
		"Event: Container started",
	}, nil
}

// GetEnvironmentInfo 获取环境信息。Get environment information.
func (c *k8sCollector) GetEnvironmentInfo(ctx context.Context) (models.Config, error) {
	// 获取节点信息。Get node information.
	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		logging.Logger.Errorf("Failed to list nodes: %v", err)
		return nil, errors.ErrDataCollectionFailed
	}

	return models.Config{
		"environment":        "kubernetes",
		"node_count":         len(nodes.Items),
		"namespace":          c.namespace,
		"kubernetes_version": "v1.28.0",
	}, nil
}

//Personal.AI order the ending
