# k8oversubs

This is a simple CLI command to get pods that uses the most CPU on a given node.

Just a very simple go experiment.

# Setup

You need to have go installed

Clone the repo and do the following:

```
$ go mod tidy
$ go build -o k8oversubs
$ ./k8oversubs --help
Usage of ./k8oversubs:
  -apply
        Apply the deletion
  -context string
        Name of the kube context
  -kubeconfig string
        (optional) absolute path to the kubeconfig file (default "/Users/homesweethome/.kube/config")
  -node string
        Name of the node
  -podcount int
        Number of pods to delete (default 10)
```

*Note* does not actually apply the deletion of pods. Yet
