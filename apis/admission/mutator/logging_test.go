package mutator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMutateLoging(t *testing.T) {
	const configStr = `[{"disable":false,"type":"file","path":"/var/log/opt/**/*log","source":"logging-var"}, {"disable":false,"type":"file","path":"/tmp/opt/log","source":"logging-tmp"}]`

	loggingMatchNamespaceOrLabelsForConfig = func(_ string, _ map[string]string) string { return configStr }

	var testCases = []struct {
		in  corev1.Pod
		out corev1.Pod
	}{
		{
			in: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testing-pod",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.22",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "exist-mount",
									MountPath: "/var/log/opt",
									ReadOnly:  false,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "exist-mount",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
			out: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "testing-pod",
					Annotations: map[string]string{"datakit/logs": configStr},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.22",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "exist-mount",
									MountPath: "/var/log/opt",
									ReadOnly:  false,
								},
								// reuse exist-mount
								// {
								// 	Name:      "datakit-logs-volume-0",
								// 	MountPath: "/var/log/opt",
								// },
								{
									Name:      "datakit-logs-volume-1",
									MountPath: "/tmp/opt",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "exist-mount",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						// reuse exist-mount
						// {
						// 	Name: "datakit-logs-volume-0",
						// 	VolumeSource: corev1.VolumeSource{
						// 		EmptyDir: &corev1.EmptyDirVolumeSource{},
						// 	},
						// },
						{
							Name: "datakit-logs-volume-1",
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
		err := MutateLoggingToPod("", testCases[idx].in.Name, &testCases[idx].in)
		assert.NoError(t, err)

		assert.Equal(t, &testCases[idx].out, &testCases[idx].in)
	}
}
