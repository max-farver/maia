package utils

func ListActivePods() []string {
	return []string{"test-pod-1", "test-pod-2"}
}

func ListContainersForPod(podName string) []string {
	return []string{"container-1", "container-2"}
}
