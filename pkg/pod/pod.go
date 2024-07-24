package pod

import (
	"context"
	"fmt"
	"log"
	"sort"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/metrics/pkg/client/clientset/versioned"
)

type PodMetrics struct {
	Name       string
	CPUUsage   int64
	CPURequest int64
	Namespace  string
}

func ProcessPods(clientset *kubernetes.Clientset, metricsClient *versioned.Clientset, nodeName string, podCount int, apply bool) {
	// Get the pods on the specified node
	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + nodeName,
	})
	if err != nil {
		log.Fatalf("Error listing pods: %v", err)
	}

	var podMetricsList []PodMetrics

	// Get metrics for each pod
	for _, pod := range pods.Items {
		metrics, err := metricsClient.MetricsV1beta1().PodMetricses(pod.Namespace).Get(context.TODO(), pod.Name, metav1.GetOptions{})
		if err != nil {
			// log.Printf("Error getting metrics for pod %s: %v", pod.Name, err)
			continue
		}

		var totalCPU int64
		for _, container := range metrics.Containers {
			cpuUsage := container.Usage.Cpu().MilliValue()
			totalCPU += cpuUsage
		}

		var totalCPURequest int64
		for _, container := range pod.Spec.Containers {
			if cpuRequest, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
				totalCPURequest += cpuRequest.MilliValue()
			}
		}

		if approvedNamespace(pod.Namespace) {
			podMetricsList = append(podMetricsList, PodMetrics{Name: pod.Name, CPUUsage: totalCPU, CPURequest: totalCPURequest, Namespace: pod.Namespace})
		}
	}

	// Sort pods by CPU usage in descending order
	sort.Slice(podMetricsList, func(i, j int) bool {
		return podMetricsList[i].CPUUsage > podMetricsList[j].CPUUsage
	})

	if len(podMetricsList) == 0 {
		log.Println("No pods found with metrics available")
		return
	}

	// Get the top 10 pods with the highest CPU usage
	topPods := podMetricsList
	if len(podMetricsList) > podCount {
		topPods = podMetricsList[:podCount]
	}

	for _, podToDelete := range topPods {
		log.Printf("[%s] pod [%s] CPU [%d m] Request [%d m]\n", podToDelete.Namespace, podToDelete.Name, podToDelete.CPUUsage, podToDelete.CPURequest)
		if apply {
			// Delete the pod
			// err = clientset.CoreV1().Pods(podToDelete.Namespace).Delete(context.TODO(), podToDelete.Name, metav1.DeleteOptions{})
			// if err != nil {
			//     log.Fatalf("Error deleting pod %s: %v", podToDelete.Name, err)
			// }

			fmt.Printf("Pod %s in namespace %s with CPU usage %d m deleted\n", podToDelete.Name, podToDelete.Namespace, podToDelete.CPUUsage)
		}
	}
}

func approvedNamespace(namespace string) bool {
	excluded := []string{
		"kube-system",
		"buildsystems",
		"cattle-fleet-system",
		"cert-manager",
		"cattle-impersonation-system",
		"cattle-system",
		"keda",
		"kube-node-lease",
		"kube-public",
		"kubecost",
		"lacework",
		"orca-security",
		"percona-monitoring",
		"platform",
		"vault-data",
		"vault-key-value",
		"velero",
		"yopass-sellers",
	}

	// Check if namespace is not in excluded
	namespaceOk := true
	for _, v := range excluded {
		if v == namespace {
			namespaceOk = false
			break
		}
	}
	return namespaceOk
}
