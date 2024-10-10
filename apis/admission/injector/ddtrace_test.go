package injector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInjectDDTrace(t *testing.T) {
	ddtraceJavaAgentImage = func() string { return "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1" }
	ddtraceEnvs = func() []struct{ Key, Value string } {
		return []struct{ Key, Value string }{
			{"DD_AGENT_HOST", "datakit-service.datakit.svc"},
			{"DD_TAGS", "host:node-02,system:linux"},

			{"POD_NAME", "{fieldRef:metadata.name}"},
			// invalid annotation key
			{"SERVICE_NOT", "{fieldRef:metadata.annotations['hello-$$$']}"},
		}
	}

	var testCases = []struct {
		in  corev1.Pod
		out corev1.Pod
	}{
		{
			in: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "testing-pod",
					Annotations: map[string]string{"admission.datakit/java-lib.version": "latest"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.22",
							Env: []corev1.EnvVar{
								{
									Name:  "DD_TAGS",
									Value: "host:node-01",
								},
							},
						},
						{
							Name:  "nginx-2",
							Image: "nginx:1.22",
							Env:   []corev1.EnvVar{},
						},
					},
				},
			},
			out: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "testing-pod",
					Annotations: map[string]string{"admission.datakit/java-lib.version": "latest"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.22",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "datakit-auto-instrument",
									MountPath: "/datadog-lib",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "DD_TAGS",
									Value: "host:node-01,system:linux",
								},
								{
									Name:  "JAVA_TOOL_OPTIONS",
									Value: " -javaagent:/datadog-lib/dd-java-agent.jar",
								},
								{
									Name:  "DD_AGENT_HOST",
									Value: "datakit-service.datakit.svc",
								},
								{
									Name: "POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name:  "SERVICE_NOT",
									Value: "{fieldRef:metadata.annotations['hello-$$$']}",
								},
							},
						},
						{
							Name:  "nginx-2",
							Image: "nginx:1.22",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "datakit-auto-instrument",
									MountPath: "/datadog-lib",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "JAVA_TOOL_OPTIONS",
									Value: " -javaagent:/datadog-lib/dd-java-agent.jar",
								},
								{
									Name:  "DD_AGENT_HOST",
									Value: "datakit-service.datakit.svc",
								},
								{
									Name:  "DD_TAGS",
									Value: "host:node-02,system:linux",
								},
								{
									Name: "POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name:  "SERVICE_NOT",
									Value: "{fieldRef:metadata.annotations['hello-$$$']}",
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:            "datakit-lib-init",
							Image:           "pubrepo.guance.com/datakit-operator/java-lib-testing:latest",
							Command:         []string{"sh", "copy-lib.sh", "/datadog-lib"},
							ImagePullPolicy: corev1.PullIfNotPresent,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "datakit-auto-instrument",
									MountPath: "/datadog-lib",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "datakit-auto-instrument",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	for idx := range testCases {
		err := InjectDDTraceToPod(testCases[idx].in.Name, &testCases[idx].in)
		assert.NoError(t, err)

		assert.Equal(t, &testCases[idx].out, &testCases[idx].in)
	}
}

func TestInjectDDTraceForNamespaces(t *testing.T) {
	ddtraceEnabledNamespaces = func(_ string) string { return "java" }
	ddtraceJavaAgentImage = func() string { return "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1" }

	ddtraceEnvs = func() []struct{ Key, Value string } {
		return []struct{ Key, Value string }{
			{"DD_AGENT_HOST", "datakit-service.datakit.svc"},
		}
	}

	var testCases = []struct {
		in  corev1.Pod
		out corev1.Pod
	}{
		{
			in: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testing-pod",
					Namespace: "testing-namespace",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.22",
						},
					},
				},
			},
			out: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testing-pod",
					Namespace: "testing-namespace",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.22",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "datakit-auto-instrument",
									MountPath: "/datadog-lib",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "JAVA_TOOL_OPTIONS",
									Value: " -javaagent:/datadog-lib/dd-java-agent.jar",
								},
								{
									Name:  "DD_AGENT_HOST",
									Value: "datakit-service.datakit.svc",
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:            "datakit-lib-init",
							Image:           "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1",
							Command:         []string{"sh", "copy-lib.sh", "/datadog-lib"},
							ImagePullPolicy: corev1.PullIfNotPresent,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "datakit-auto-instrument",
									MountPath: "/datadog-lib",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "datakit-auto-instrument",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	for idx := range testCases {
		err := InjectDDTraceToPod(testCases[idx].in.Name, &testCases[idx].in)
		assert.NoError(t, err)

		assert.Equal(t, &testCases[idx].out, &testCases[idx].in)
	}
}

func TestInjectDDTraceForLabelSelectors(t *testing.T) {
	ddtraceEnabledLabelSelectors = func(_ map[string]string) string { return "java" }
	// not used namespaces
	ddtraceEnabledNamespaces = func(_ string) string { return "java" }
	ddtraceJavaAgentImage = func() string { return "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1" }

	ddtraceEnvs = func() []struct{ Key, Value string } {
		return []struct{ Key, Value string }{
			{"DD_AGENT_HOST", "datakit-service.datakit.svc"},
		}
	}

	var testCases = []struct {
		in  corev1.Pod
		out corev1.Pod
	}{
		{
			in: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testing-pod",
					Namespace: "testing-namespace",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.22",
						},
					},
				},
			},
			out: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testing-pod",
					Namespace: "testing-namespace",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.22",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "datakit-auto-instrument",
									MountPath: "/datadog-lib",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "JAVA_TOOL_OPTIONS",
									Value: " -javaagent:/datadog-lib/dd-java-agent.jar",
								},
								{
									Name:  "DD_AGENT_HOST",
									Value: "datakit-service.datakit.svc",
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:            "datakit-lib-init",
							Image:           "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1",
							Command:         []string{"sh", "copy-lib.sh", "/datadog-lib"},
							ImagePullPolicy: corev1.PullIfNotPresent,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "datakit-auto-instrument",
									MountPath: "/datadog-lib",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "datakit-auto-instrument",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	for idx := range testCases {
		err := InjectDDTraceToPod(testCases[idx].in.Name, &testCases[idx].in)
		assert.NoError(t, err)

		assert.Equal(t, &testCases[idx].out, &testCases[idx].in)
	}
}
