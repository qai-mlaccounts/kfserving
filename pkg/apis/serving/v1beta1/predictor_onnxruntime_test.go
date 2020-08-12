/*
Copyright 2020 kubeflow.org.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"github.com/golang/protobuf/proto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	"github.com/kubeflow/kfserving/pkg/constants"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestOnnxRuntimeValidation(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	scenarios := map[string]struct {
		spec    PredictorSpec
		matcher types.GomegaMatcher
	}{
		"AcceptGoodRuntimeVersion": {
			spec: PredictorSpec{
				ONNX: &ONNXRuntimeSpec{
					PredictorExtensionSpec: PredictorExtensionSpec{
						RuntimeVersion: proto.String("latest"),
					},
				},
			},
			matcher: gomega.BeNil(),
		},
		"ValidStorageUri": {
			spec: PredictorSpec{
				ONNX: &ONNXRuntimeSpec{
					PredictorExtensionSpec: PredictorExtensionSpec{
						StorageURI: proto.String("s3://modelzoo"),
					},
				},
			},
			matcher: gomega.BeNil(),
		},
		"InvalidStorageUri": {
			spec: PredictorSpec{
				ONNX: &ONNXRuntimeSpec{
					PredictorExtensionSpec: PredictorExtensionSpec{
						StorageURI: proto.String("hdfs://modelzoo"),
					},
				},
			},
			matcher: gomega.Not(gomega.BeNil()),
		},
		"InvalidReplica": {
			spec: PredictorSpec{
				ComponentExtensionSpec: ComponentExtensionSpec{
					MinReplicas: GetIntReference(3),
					MaxReplicas: 2,
				},
				ONNX: &ONNXRuntimeSpec{
					PredictorExtensionSpec: PredictorExtensionSpec{
						StorageURI: proto.String("hdfs://modelzoo"),
					},
				},
			},
			matcher: gomega.Not(gomega.BeNil()),
		},
		"InvalidContainerConcurrency": {
			spec: PredictorSpec{
				ComponentExtensionSpec: ComponentExtensionSpec{
					MinReplicas:          GetIntReference(3),
					ContainerConcurrency: proto.Int64(-1),
				},
				ONNX: &ONNXRuntimeSpec{
					PredictorExtensionSpec: PredictorExtensionSpec{
						StorageURI: proto.String("hdfs://modelzoo"),
					},
				},
			},
			matcher: gomega.Not(gomega.BeNil()),
		},
	}

	for name, scenario := range scenarios {
		t.Run(name, func(t *testing.T) {
			res := scenario.spec.Validate()
			if !g.Expect(res).To(scenario.matcher) {
				t.Errorf("got %q, want %q", res, scenario.matcher)
			}
		})
	}
}

func TestONNXRuntimeDefaulter(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	config := InferenceServicesConfig{
		Predictors: PredictorsConfig{
			ONNX: PredictorConfig{
				ContainerImage:      "onnxruntime",
				DefaultImageVersion: "v1.0.0",
			},
		},
	}
	defaultResource = v1.ResourceList{
		v1.ResourceCPU:    resource.MustParse("1"),
		v1.ResourceMemory: resource.MustParse("2Gi"),
	}
	scenarios := map[string]struct {
		spec     PredictorSpec
		expected PredictorSpec
	}{
		"DefaultRuntimeVersion": {
			spec: PredictorSpec{
				ONNX: &ONNXRuntimeSpec{
					PredictorExtensionSpec: PredictorExtensionSpec{},
				},
			},
			expected: PredictorSpec{
				ONNX: &ONNXRuntimeSpec{
					PredictorExtensionSpec: PredictorExtensionSpec{
						RuntimeVersion: proto.String("v1.0.0"),
						Container: v1.Container{
							Name: constants.InferenceServiceContainerName,
							Resources: v1.ResourceRequirements{
								Requests: defaultResource,
								Limits:   defaultResource,
							},
						},
					},
				},
			},
		},
		"DefaultResources": {
			spec: PredictorSpec{
				ONNX: &ONNXRuntimeSpec{
					PredictorExtensionSpec: PredictorExtensionSpec{
						RuntimeVersion: proto.String("v1.0.0"),
					},
				},
			},
			expected: PredictorSpec{
				ONNX: &ONNXRuntimeSpec{
					PredictorExtensionSpec: PredictorExtensionSpec{
						RuntimeVersion: proto.String("v1.0.0"),
						Container: v1.Container{
							Name: constants.InferenceServiceContainerName,
							Resources: v1.ResourceRequirements{
								Requests: defaultResource,
								Limits:   defaultResource,
							},
						},
					},
				},
			},
		},
	}

	for name, scenario := range scenarios {
		t.Run(name, func(t *testing.T) {
			scenario.spec.ONNX.Default(&config)
			if !g.Expect(scenario.spec).To(gomega.Equal(scenario.expected)) {
				t.Errorf("got %q, want %q", scenario.spec, scenario.expected)
			}
		})
	}
}

func TestCreateONNXRuntimeContainer(t *testing.T) {

	var requestedResource = v1.ResourceRequirements{
		Limits: v1.ResourceList{
			"cpu": resource.Quantity{
				Format: "100",
			},
		},
		Requests: v1.ResourceList{
			"cpu": resource.Quantity{
				Format: "90",
			},
		},
	}
	var config = InferenceServicesConfig{
		Predictors: PredictorsConfig{
			ONNX: PredictorConfig{
				ContainerImage:      "mcr.microsoft.com/onnxruntime/server",
				DefaultImageVersion: "v1.0.0",
			},
		},
	}
	g := gomega.NewGomegaWithT(t)
	scenarios := map[string]struct {
		isvc                  InferenceService
		expectedContainerSpec *v1.Container
	}{
		"ContainerSpecWithDefaultImage": {
			isvc: InferenceService{
				ObjectMeta: metav1.ObjectMeta{
					Name: "onnx",
				},
				Spec: InferenceServiceSpec{
					Predictor: PredictorSpec{
						ONNX: &ONNXRuntimeSpec{
							PredictorExtensionSpec: PredictorExtensionSpec{
								StorageURI:     proto.String("gs://someUri"),
								RuntimeVersion: proto.String("v1.0.0"),
								Container: v1.Container{
									Resources: requestedResource,
								},
							},
						},
					},
				},
			},
			expectedContainerSpec: &v1.Container{
				Image:     "mcr.microsoft.com/onnxruntime/server:v1.0.0",
				Name:      constants.InferenceServiceContainerName,
				Resources: requestedResource,
				Args: []string{
					"--model_path=/mnt/models/model.onnx",
					"--http_port=8080",
					"--grpc_port=9000",
				},
			},
		},
		"ContainerSpecWithCustomImage": {
			isvc: InferenceService{
				ObjectMeta: metav1.ObjectMeta{
					Name: "onnx",
				},
				Spec: InferenceServiceSpec{
					Predictor: PredictorSpec{
						ONNX: &ONNXRuntimeSpec{
							PredictorExtensionSpec: PredictorExtensionSpec{
								StorageURI: proto.String("gs://someUri"),
								Container: v1.Container{
									Image:     "mcr.microsoft.com/onnxruntime/server:v0.5.0",
									Resources: requestedResource,
								},
							},
						},
					},
				},
			},
			expectedContainerSpec: &v1.Container{
				Image:     "mcr.microsoft.com/onnxruntime/server:v0.5.0",
				Name:      constants.InferenceServiceContainerName,
				Resources: requestedResource,
				Args: []string{
					"--model_path=/mnt/models/model.onnx",
					"--http_port=8080",
					"--grpc_port=9000",
				},
			},
		},
		"ContainerSpecWithContainerConcurrency": {
			isvc: InferenceService{
				ObjectMeta: metav1.ObjectMeta{
					Name: "onnx",
				},
				Spec: InferenceServiceSpec{
					Predictor: PredictorSpec{
						ComponentExtensionSpec: ComponentExtensionSpec{
							ContainerConcurrency: proto.Int64(1),
						},
						ONNX: &ONNXRuntimeSpec{
							PredictorExtensionSpec: PredictorExtensionSpec{
								StorageURI:     proto.String("gs://someUri"),
								RuntimeVersion: proto.String("v1.0.0"),
								Container: v1.Container{
									Resources: requestedResource,
								},
							},
						},
					},
				},
			},
			expectedContainerSpec: &v1.Container{
				Image:     "mcr.microsoft.com/onnxruntime/server:v1.0.0",
				Name:      constants.InferenceServiceContainerName,
				Resources: requestedResource,
				Args: []string{
					"--model_path=/mnt/models/model.onnx",
					"--http_port=8080",
					"--grpc_port=9000",
				},
			},
		},
	}
	for name, scenario := range scenarios {
		t.Run(name, func(t *testing.T) {
			predictor := scenario.isvc.Spec.Predictor.GetPredictor()[0]
			res := predictor.GetContainer(metav1.ObjectMeta{Name: "someName"}, scenario.isvc.Spec.Predictor.ContainerConcurrency, &config)
			if !g.Expect(res).To(gomega.Equal(scenario.expectedContainerSpec)) {
				t.Errorf("got %q, want %q", res, scenario.expectedContainerSpec)
			}
		})
	}
}