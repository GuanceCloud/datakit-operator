package injector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInjectLogfwd(t *testing.T) {
	logfwdReuseExistVolume = func() bool { return true }
	logfwdImage = func() string { return "pubrepo.guance.com/datakit-operator/logfwd-testing:v1.0.1" }
	logfwdEnvs = func() []struct{ Key, Value string } {
		return []struct{ Key, Value string }{
			{"LOGFWD_POD_NAME", "{fieldRef:metadata.name}"},
			{"LOGFWD_POD_NAMESPACE", "{fieldRef:metadata.namespace}"},
			{"LOGFWD_GLOBAL_SERVICE", "{fieldRef:metadata.labels['app']}"},
		}
	}

	var instances = `[
    {
        "datakit_addr": "datakit-service.datakit.svc:9533",
        "loggings": [
            {
                "logfiles": ["/var/log/nginx/success/*.log"],
                "source": "nginx-success",
                "tags": {
                    "key01": "value01"
                }
            },
            {
                "logfiles": ["/var/log/nginx/error/*.log"],
                "source": "nginx-error",
                "pipeline": "nginx-error.p"
            }
        ]
    }
]
`
	var instancesCompact = `[{"datakit_addr":"datakit-service.datakit.svc:9533","loggings":[{"logfiles":["/var/log/nginx/success/*.log"],"source":"nginx-success","tags":{"key01":"value01"}},{"logfiles":["/var/log/nginx/error/*.log"],"source":"nginx-error","pipeline":"nginx-error.p"}]}]`

	var testCases = []struct {
		in  corev1.Pod
		out corev1.Pod
	}{
		{
			in: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "testing-pod",
					Annotations: map[string]string{"admission.datakit/logfwd.instances": instances},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.22",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "exist-mount",
									MountPath: "/var/log/nginx/success",
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
					Annotations: map[string]string{"admission.datakit/logfwd.instances": instances},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.22",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "exist-mount",
									MountPath: "/var/log/nginx/success",
									ReadOnly:  false,
								},
								// {
								// 	// reuse exist-mount
								// 	Name:      "datakit-logfwd-volume-0",
								// 	MountPath: "/var/log/nginx/success",
								// },
								{
									Name:      "datakit-logfwd-volume-1",
									MountPath: "/var/log/nginx/error",
								},
							},
						},
						{
							Name:            "datakit-logfwd",
							Image:           "pubrepo.guance.com/datakit-operator/logfwd-testing:v1.0.1",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Env: []corev1.EnvVar{
								{
									Name: "LOGFWD_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "LOGFWD_POD_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
								{
									Name: "LOGFWD_GLOBAL_SERVICE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.labels['app']",
										},
									},
								},
								{
									Name:  "LOGFWD_JSON_CONFIG",
									Value: instancesCompact,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									// reuse exist-mount
									// Name: "datakit-logfwd-volume-0",
									Name:      "exist-mount",
									MountPath: "/var/log/nginx/success",
									ReadOnly:  true,
								},
								{
									Name:      "datakit-logfwd-volume-1",
									MountPath: "/var/log/nginx/error",
									ReadOnly:  true,
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("64Mi"),
								},
								Limits: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
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
						// {
						// 	Name: "datakit-logfwd-volume-0",
						// 	VolumeSource: corev1.VolumeSource{
						// 		EmptyDir: &corev1.EmptyDirVolumeSource{},
						// 	},
						// },
						{
							Name: "datakit-logfwd-volume-1",
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
		err := InjectLogfwdToPod(testCases[idx].in.Name, &testCases[idx].in)
		assert.NoError(t, err)

		assert.Equal(t, &testCases[idx].out, &testCases[idx].in)
	}
}
