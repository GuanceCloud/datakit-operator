package admission

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInjectProfiler(t *testing.T) {
	profilerJavaImage = func() string { return "pubrepo.guance.com/datakit-operator/java-profiler-testing:v1.0.1" }
	profilerEnvs = func() []struct{ Key, Value string } {
		return []struct{ Key, Value string }{
			{"DK_AGENT_HOST", "datakit-service.datakit.svc"},
			{"DK_AGENT_PORT", "9529"},
			{"DK_PROFILE_DURATION", "240"},
			{"DK_PROFILE_ENV", "prod"},
			{"DK_PROFILE_SCHEDULE", "*/20 * * * *"},
			{"DK_PROFILE_VERSION", "1.2.333"},
		}
	}

	var (
		shareProcessNamespace = true
		fileOrCreate          = corev1.HostPathFileOrCreate
	)

	var testCases = []struct {
		in  corev1.PodTemplateSpec
		out corev1.PodTemplateSpec
	}{
		{
			in: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "testing-podTemplate",
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
			out: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "testing-podTemplate",
					Annotations: map[string]string{"admission.datakit/java-profiler.version": "latest"},
				},
				Spec: corev1.PodSpec{
					ShareProcessNamespace: &shareProcessNamespace,
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
							Command: []string{
								"bash",
								"-c",
								"mv -f /app/async-profiler/* /app/datakit-profiler/; ./profiling.sh --add-crontab; cron -f",
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
									Name:  "DK_PROFILE_DURATION",
									Value: "240",
								},
								{
									Name:  "DK_PROFILE_ENV",
									Value: "prod",
								},
								{
									Name:  "DK_PROFILE_SCHEDULE",
									Value: "*/20 * * * *",
								},
								{
									Name:  "DK_PROFILE_VERSION",
									Value: "1.2.333",
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
									Add: []corev1.Capability{"SYS_PTRACE"},
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
		err := injectProfilerToPodTemplate(testCases[idx].in.Name, &testCases[idx].in)
		assert.NoError(t, err)

		assert.Equal(t, &testCases[idx].out, &testCases[idx].in)

		// output, err := yaml.Marshal(&testCases[idx].in)
		// assert.NoError(t, err)
		// t.Logf("%s", output)
	}
}
