package admission

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInjectLogfwd(t *testing.T) {
	err := os.Setenv("ENV_LOGFWD_IMAGE", "pubrepo.jiagouyun.com/datakit-operator/logfwd-testing:v1.0.1")
	assert.NoError(t, err)
	defer os.Unsetenv("ENV_LOGFWD_IMAGE")

	var instances = `
[
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
		in  corev1.PodTemplateSpec
		out corev1.PodTemplateSpec
	}{
		{
			in: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "testing-podTemplate",
					Annotations: map[string]string{"admission.datakit/logfwd.instances": instances},
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
					Annotations: map[string]string{"admission.datakit/logfwd.instances": instances},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.22",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "datakit-logfwd-volume-0",
									MountPath: "/var/log/nginx/success",
								},
								{
									Name:      "datakit-logfwd-volume-1",
									MountPath: "/var/log/nginx/error",
								},
							},
						},
						{
							Name:            "datakit-logfwd",
							Image:           "pubrepo.jiagouyun.com/datakit-operator/logfwd-testing:v1.0.1",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Env: []corev1.EnvVar{
								{
									Name:  "LOGFWD_JSON_CONFIG",
									Value: instancesCompact,
								},
								{
									Name: "LOGFWD_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											APIVersion: "v1",
											FieldPath:  "metadata.name",
										},
									},
								},
								{
									Name: "LOGFWD_POD_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											APIVersion: "v1",
											FieldPath:  "metadata.namespace",
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "datakit-logfwd-volume-0",
									MountPath: "/var/log/nginx/success",
									ReadOnly:  true,
								},
								{
									Name:      "datakit-logfwd-volume-1",
									MountPath: "/var/log/nginx/error",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "datakit-logfwd-volume-0",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
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
		err := injectLogfwdToPodTemplate(&testCases[idx].in)
		assert.NoError(t, err)

		assert.Equal(t, &testCases[idx].out, &testCases[idx].in)
	}
}
