package injector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInjectProfiler(t *testing.T) {
	profilerJavaImage = func() string { return "pubrepo.guance.com/datakit-operator/java-profiler-testing:v1.0.1" }
	profilerEnvs = func() []struct{ Key, Value string } {
		return []struct{ Key, Value string }{
			{"DK_AGENT_HOST", "datakit-service.datakit.svc"},
			{"DK_AGENT_PORT", "9529"},

			{"POD_NAME", "{fieldRef:metadata.name}"},
		}
	}

	var (
		shareProcessNamespace = true
		fileOrCreate          = corev1.HostPathFileOrCreate
	)

	var testCases = []struct {
		in  corev1.Pod
		out corev1.Pod
	}{
		{
			in: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "testing-pod",
					Annotations: map[string]string{"admission.datakit/java-profiler.version": "latest"},
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
					Annotations: map[string]string{"admission.datakit/java-profiler.version": "latest"},
				},
				Spec: corev1.PodSpec{
					ShareProcessNamespace: &shareProcessNamespace,
					RestartPolicy:         corev1.RestartPolicyAlways,
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.22",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "datakit-profiler-volume",
									MountPath: "/app/datakit-profiler",
								},
								{
									Name:      "tmp",
									MountPath: "/tmp",
								},
								{
									Name:      "timezone",
									MountPath: "/etc/localtime",
								},
							},
						},
						{
							Name:            "datakit-profiler",
							Image:           "pubrepo.guance.com/datakit-operator/java-profiler-testing:latest",
							ImagePullPolicy: corev1.PullIfNotPresent,
							WorkingDir:      "/app/datakit-profiler",
							Command: []string{
								"bash",
								"cmd.sh",
							},
							Env: []corev1.EnvVar{
								{
									Name:  "DK_AGENT_HOST",
									Value: "datakit-service.datakit.svc",
								},
								{
									Name:  "DK_AGENT_PORT",
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
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "datakit-profiler-volume",
									MountPath: "/app/datakit-profiler",
								},
								{
									Name:      "tmp",
									MountPath: "/tmp",
								},
								{
									Name:      "timezone",
									MountPath: "/etc/localtime",
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{"SYS_PTRACE", "SYS_ADMIN"},
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
							Name: "datakit-profiler-volume",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "tmp",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "timezone",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc/localtime",
									Type: &fileOrCreate,
								},
							},
						},
					},
				},
			},
		},
	}

	for idx := range testCases {
		err := InjectProfilerToPod(testCases[idx].in.Name, &testCases[idx].in)
		assert.NoError(t, err)

		assert.Equal(t, &testCases[idx].out, &testCases[idx].in)

		// output, err := yaml.Marshal(&testCases[idx].in)
		// assert.NoError(t, err)
		// t.Logf("%s", output)
	}
}
