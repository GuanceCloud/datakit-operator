package injector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInjectDDTrace(t *testing.T) {
	ddtraceJavaAgentImage = func() string { return "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1" }
	ddtraceEnvs = func() []struct{ Key, Value string } {
		return []struct{ Key, Value string }{
			{"DD_AGENT_HOST", "datakit-service.datakit.svc"},
			{"DD_TRACE_AGENT_PORT", "9529"},

			{"POD_NAME", "{fieldRef:metadata.name}"},
			{"SERVICE", "{fieldRef:metadata.annotations['service']}"},

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
									Name:  "JAVA_TOOL_OPTIONS",
									Value: " -javaagent:/datadog-lib/dd-java-agent.jar",
								},
								{
									Name:  "DD_AGENT_HOST",
									Value: "datakit-service.datakit.svc",
								},
								{
									Name:  "DD_TRACE_AGENT_PORT",
									Value: "9529",
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
									Name: "SERVICE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.annotations['service']",
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
							Resources: corev1.ResourceRequirements{
								Requests: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    resource.MustParse("200m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
								Limits: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    resource.MustParse("1000m"),
									corev1.ResourceMemory: resource.MustParse("1Gi"),
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
