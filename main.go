package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sort"

	//"time"

	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/metrics/pkg/client/clientset/versioned"
)

type PodMetrics struct {
	Name      string
	CPU       int64
	Namespace string
}

func main() {
	config, nodeName, podCount, apply := parseFlags()

	clientset, metricsClient := createClientset(config)

	processPods(clientset, metricsClient, nodeName, podCount, apply)
}

func parseFlags() (config *rest.Config, nodeName string, podCount int, apply bool) {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	nodeNamePtr := flag.String("node", "", "Name of the node")
	contextNamePtr := flag.String("context", "", "Name of the kube context")
	podCountPtr := flag.Int("podcount", 10, "Number of pods to delete")
	applyPtr := flag.Bool("apply", false, "Apply the deletion")

	flag.Parse()

	if *nodeNamePtr == "" {
		log.Fatalf("Node name must be provided")
	}

	// Build the configuration from the kubeconfig file
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %v", err)
	}

	if *contextNamePtr != "" {
		configOverrides := &clientcmd.ConfigOverrides{CurrentContext: *contextNamePtr}
		config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: *kubeconfig},
			configOverrides,
		).ClientConfig()
		if err != nil {
			log.Fatalf("Error setting context: %v", err)
		}
	}

	return config, *nodeNamePtr, *podCountPtr, *applyPtr
}

func createClientset(config *rest.Config) (*kubernetes.Clientset, *versioned.Clientset) {
	// Create the clientset for core APIs
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating clientset: %v", err)
	}

	// Create the clientset for metrics APIs
	metricsClient, err := versioned.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating metrics client: %v", err)
	}

	return clientset, metricsClient
}

func processPods(clientset *kubernetes.Clientset, metricsClient *versioned.Clientset, nodeName string, podCount int, apply bool) {
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

		if pod.Namespace == "default" || pod.Namespace == "consumers" {
			podMetricsList = append(podMetricsList, PodMetrics{Name: pod.Name, CPU: totalCPU, Namespace: pod.Namespace})
		}
	}

	// Sort pods by CPU usage in descending order
	sort.Slice(podMetricsList, func(i, j int) bool {
		return podMetricsList[i].CPU > podMetricsList[j].CPU
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
		log.Printf("[%s] pod [%s] CPU [%d m]\n", podToDelete.Namespace, podToDelete.Name, podToDelete.CPU)
		if apply == true {
			// Delete the pod
			// err = clientset.CoreV1().Pods(podToDelete.Namespace).Delete(context.TODO(), podToDelete.Name, metav1.DeleteOptions{})
			// if err != nil {
			//     log.Fatalf("Error deleting pod %s: %v", podToDelete.Name, err)
			// }

			fmt.Printf("Pod %s in namespace %s with CPU usage %d m deleted\n", podToDelete.Name, podToDelete.Namespace, podToDelete.CPU)
		}
	}
}
