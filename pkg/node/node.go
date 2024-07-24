package node

import (
	"context"
	"log"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/metrics/pkg/client/clientset/versioned"
)

type NodeMetrics struct {
	Name     string
	CPUUsage int64
}

func ProcessNodes(clientset *kubernetes.Clientset, metricsClient *versioned.Clientset) string {
	// Get the nodes in the cluster
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Error listing nodes: %v", err)
	}

	// Get the CPU usage for each node
	var nodeMetricsList []NodeMetrics
	for _, node := range nodes.Items {
		metrics, err := metricsClient.MetricsV1beta1().NodeMetricses().Get(context.TODO(), node.Name, metav1.GetOptions{})
		if err != nil {
			// log.Printf("Error getting metrics for node %s: %v", node.Name, err)
			continue
		}

		cpuUsage := metrics.Usage.Cpu().MilliValue()
		nodeMetricsList = append(nodeMetricsList, NodeMetrics{Name: node.Name, CPUUsage: cpuUsage})
	}

	// Sort nodes by CPU usage in descending order
	sort.Slice(nodeMetricsList, func(i, j int) bool {
		return nodeMetricsList[i].CPUUsage > nodeMetricsList[j].CPUUsage
	})

	if len(nodeMetricsList) == 0 {
		log.Println("No nodes found with metrics available")
		return ""
	}

	// Get the node with the highest CPU usage
	topNode := nodeMetricsList[0]
	log.Printf("Node [%s] CPU [%d m]\n", topNode.Name, topNode.CPUUsage)

	return topNode.Name
}
