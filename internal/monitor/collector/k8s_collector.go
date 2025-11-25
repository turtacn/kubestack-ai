package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/monitor/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	metricsv1beta1 "k8s.io/metrics/pkg/client/clientset/versioned"
)

type K8sCollector struct {
	clientset     *kubernetes.Clientset
	metricsClient *metricsv1beta1.Clientset
	namespace     string // Empty means all namespaces
}

// NewK8sCollector creates a new Kubernetes collector
func NewK8sCollector(kubeconfigPath string, namespace string) (*K8sCollector, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	metricsClient, err := metricsv1beta1.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics client: %w", err)
	}

	return &K8sCollector{
		clientset:     clientset,
		metricsClient: metricsClient,
		namespace:     namespace,
	}, nil
}

// Collect implements MetricsCollector interface
func (c *K8sCollector) Collect(ctx context.Context) ([]*model.MetricPoint, error) {
	points := make([]*model.MetricPoint, 0)

	// 1. Collect Node Metrics
	nodeMetrics, err := c.metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list node metrics: %w", err)
	}

	for _, node := range nodeMetrics.Items {
		// CPU usage (millicores)
		cpuUsage := float64(node.Usage.Cpu().MilliValue()) / 1000.0
		points = append(points, &model.MetricPoint{
			Name:      "k8s_node_cpu_usage_cores",
			Value:     cpuUsage,
			Timestamp: time.Now(),
			Labels: map[string]string{
				"node": node.Name,
				"type": "node",
			},
		})

		// Memory usage (bytes)
		memUsage := float64(node.Usage.Memory().Value())
		points = append(points, &model.MetricPoint{
			Name:      "k8s_node_memory_usage_bytes",
			Value:     memUsage,
			Timestamp: time.Now(),
			Labels: map[string]string{
				"node": node.Name,
				"type": "node",
			},
		})
	}

	// 2. Collect Pod Metrics
	podMetrics, err := c.metricsClient.MetricsV1beta1().PodMetricses(c.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pod metrics: %w", err)
	}

	for _, pod := range podMetrics.Items {
		for _, container := range pod.Containers {
			cpuUsage := float64(container.Usage.Cpu().MilliValue()) / 1000.0
			points = append(points, &model.MetricPoint{
				Name:      "k8s_pod_cpu_usage_cores",
				Value:     cpuUsage,
				Timestamp: time.Now(),
				Labels: map[string]string{
					"namespace": pod.Namespace,
					"pod":       pod.Name,
					"container": container.Name,
				},
			})

			memUsage := float64(container.Usage.Memory().Value())
			points = append(points, &model.MetricPoint{
				Name:      "k8s_pod_memory_usage_bytes",
				Value:     memUsage,
				Timestamp: time.Now(),
				Labels: map[string]string{
					"namespace": pod.Namespace,
					"pod":       pod.Name,
					"container": container.Name,
				},
			})
		}
	}

	return points, nil
}

func (c *K8sCollector) Name() string {
	return "kubernetes"
}

func (c *K8sCollector) Interval() time.Duration {
	return 30 * time.Second // Could be configurable
}
