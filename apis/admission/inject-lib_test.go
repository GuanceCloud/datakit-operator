package admission

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInjectLib(t *testing.T) {
	err := os.Setenv("ENV_DD_JAVA_AGENT_IMAGE", "pubrepo.jiagouyun.com/datakit-operator/java-lib-testing:v1.0.1")
	assert.NoError(t, err)
	err = os.Setenv("ENV_DD_AGENT_HOST", "datakit-service.datakit.svc")
	assert.NoError(t, err)
	err = os.Setenv("ENV_DD_TRACE_AGENT_PORT", "9529")
	assert.NoError(t, err)
	err = os.Setenv("ENV_DD_JMXFETCH_STATSD_HOST", "datakit-service.datakit.svc")
	assert.NoError(t, err)
	err = os.Setenv("ENV_DD_JMXFETCH_STATSD_PORT", "8125")
	assert.NoError(t, err)
	defer func() {
		os.Unsetenv("ENV_DD_JAVA_AGENT_IMAGE")
		os.Unsetenv("ENV_DD_AGENT_HOST")
		os.Unsetenv("ENV_DD_TRACE_AGENT_PORT")
		os.Unsetenv("ENV_DD_JMXFETCH_STATSD_HOST")
		os.Unsetenv("ENV_DD_JMXFETCH_STATSD_PORT")
	}()

	var testCases = []struct {
		in  corev1.PodTemplateSpec
		out corev1.PodTemplateSpec
	}{
		{
			in: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "testing-podTemplate",
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
			out: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "testing-podTemplate",
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
									Name:  "DD_JMXFETCH_STATSD_HOST",
									Value: "datakit-service.datakit.svc",
								},
								{
									Name:  "DD_JMXFETCH_STATSD_PORT",
									Value: "8125",
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:            "datakit-lib-init",
							Image:           "pubrepo.jiagouyun.com/datakit-operator/java-lib-testing:latest",
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
		err := injectLibToPodTemplate(testCases[idx].in.Name, &testCases[idx].in)
		assert.NoError(t, err)

		assert.Equal(t, &testCases[idx].out, &testCases[idx].in)
	}
}
