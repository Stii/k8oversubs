package main

import (
	"flag"
	"k8oversubs/pkg/node"
	"k8oversubs/pkg/pod"
	"log"

	//"time"

	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/metrics/pkg/client/clientset/versioned"
)

func main() {
	config, nodeName, podCount, apply := parseFlags()

	clientset, metricsClient := createClientset(config)

	if nodeName == "getNodeName" {
		nodeName = node.ProcessNodes(clientset, metricsClient)
	}

	pod.ProcessPods(clientset, metricsClient, nodeName, podCount, apply)
}

func parseFlags() (config *rest.Config, nodeName string, podCount int, apply bool) {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	nodeNamePtr := flag.String("node", "getNodeName", "Name of the node")
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
