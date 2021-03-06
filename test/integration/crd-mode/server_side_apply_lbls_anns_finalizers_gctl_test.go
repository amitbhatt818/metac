/*
Copyright 2020 The MayaData Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package crdmode

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/klog"

	"openebs.io/metac/controller/generic"
	"openebs.io/metac/test/integration/framework"
	"openebs.io/metac/third_party/kubernetes"
)

func TestServerSideApplyOfLblsAnnsFinalizersViaGctl(t *testing.T) {
	f := framework.NewIntegrationTester(t)
	defer f.TearDown()

	watchName := "watch-ssaolafvg"
	namespaceName := "ns-ssaolafvg"
	attachmentName := "attachment-ssaolafvg"

	// define "reconcile logic" in this hook
	syncHook := f.ServeWebhook(func(body []byte) ([]byte, error) {
		req := generic.SyncHookRequest{}
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, err
		}

		child := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"kind":       "ServerSideApplyAttachment",
				"apiVersion": "integration.test.io/v1alpha1",
				"metadata": map[string]interface{}{
					"name":      attachmentName,
					"namespace": namespaceName,
					"labels": map[string]interface{}{
						"app": "metac",
					},
				},
			},
		}
		resp := generic.SyncHookResponse{
			Attachments: []*unstructured.Unstructured{child},
			// we set the resync to a very short time here to test
			// if server side apply is working as desired in whatever
			// limited test time we have got
			ResyncAfterSeconds: 1,
		}
		return json.Marshal(resp)
	})

	// Run the testcase here
	//
	// NOTE:
	// 	TestSteps are executed in their defined order
	result, err := f.Test(
		[]framework.TestStep{
			framework.TestStep{
				Name: "create-testcase-namespace",
				Apply: framework.Apply{
					State: &unstructured.Unstructured{
						Object: map[string]interface{}{
							"kind":       "Namespace",
							"apiVersion": "v1",
							"metadata": map[string]interface{}{
								"name": namespaceName,
							},
						},
					},
				},
			},
			framework.TestStep{
				Name: "create-watch-crd",
				Apply: framework.Apply{
					State: &unstructured.Unstructured{
						Object: map[string]interface{}{
							"apiVersion": "apiextensions.k8s.io/v1beta1",
							"kind":       "CustomResourceDefinition",
							"metadata": map[string]interface{}{
								"name": "serversideapplywatches.integration.test.io",
							},
							"spec": map[string]interface{}{
								"version": "v1alpha1",
								"group":   "integration.test.io",
								"scope":   "Namespaced",
								"names": map[string]interface{}{
									"kind":     "ServerSideApplyWatch",
									"listKind": "ServerSideApplyWatchList",
									"singular": "serversideapplywatch",
									"plural":   "serversideapplywatches",
									"shortNames": []interface{}{
										"ssawatch",
									},
								},
								"versions": []interface{}{
									map[string]interface{}{
										"name":    "v1alpha1",
										"served":  true,
										"storage": true,
									},
								},
							},
						},
					},
				},
			},
			framework.TestStep{
				Name: "create-attachment-crd",
				Apply: framework.Apply{
					State: &unstructured.Unstructured{
						Object: map[string]interface{}{
							"apiVersion": "apiextensions.k8s.io/v1beta1",
							"kind":       "CustomResourceDefinition",
							"metadata": map[string]interface{}{
								"name": "serversideapplyattachments.integration.test.io",
							},
							"spec": map[string]interface{}{
								"version": "v1alpha1",
								"group":   "integration.test.io",
								"scope":   "Namespaced",
								"names": map[string]interface{}{
									"kind":     "ServerSideApplyAttachment",
									"listKind": "ServerSideApplyAttachmentList",
									"singular": "serversideapplyattachment",
									"plural":   "serversideapplyattachments",
									"shortNames": []interface{}{
										"ssaattachment",
									},
								},
								"versions": []interface{}{
									map[string]interface{}{
										"name":    "v1alpha1",
										"served":  true,
										"storage": true,
									},
								},
							},
						},
					},
				},
			},
			framework.TestStep{
				Name: "create-watch-resource",
				Apply: framework.Apply{
					State: &unstructured.Unstructured{
						Object: map[string]interface{}{
							"kind":       "ServerSideApplyWatch",
							"apiVersion": "integration.test.io/v1alpha1",
							"metadata": map[string]interface{}{
								"name":      watchName,
								"namespace": namespaceName,
							},
						},
					},
				},
			},
			framework.TestStep{
				Name: "create-generic-controller",
				Apply: framework.Apply{
					State: &unstructured.Unstructured{
						Object: map[string]interface{}{
							"kind":       "GenericController",
							"apiVersion": "metac.openebs.io/v1alpha1",
							"metadata": map[string]interface{}{
								"name":      "server-side-apply-lbls-anns-finalizers-gctl",
								"namespace": namespaceName,
							},
							"spec": map[string]interface{}{
								"watch": map[string]interface{}{
									"apiVersion": "integration.test.io/v1alpha1",
									"resource":   "serversideapplywatches",
								},
								"attachments": []interface{}{
									map[string]interface{}{
										"apiVersion": "integration.test.io/v1alpha1",
										"resource":   "serversideapplyattachments",
									},
								},
								"hooks": map[string]interface{}{
									"sync": map[string]interface{}{
										"webhook": map[string]interface{}{
											"url": kubernetes.StringPtr(syncHook.URL),
										},
									},
								},
							},
						},
					},
				},
			},
			framework.TestStep{
				Name: "assert-presence-of-attachment",
				Assert: &framework.Assert{
					State: &unstructured.Unstructured{
						Object: map[string]interface{}{
							"kind":       "ServerSideApplyAttachment",
							"apiVersion": "integration.test.io/v1alpha1",
							"metadata": map[string]interface{}{
								"name":      attachmentName,
								"namespace": namespaceName,
								"labels": map[string]interface{}{
									"app": "metac",
								},
							},
						},
					},
				},
			},
			framework.TestStep{
				Name: "set-attachment-lbls-anns-finalizers-externally",
				Apply: framework.Apply{
					State: &unstructured.Unstructured{
						Object: map[string]interface{}{
							"kind":       "ServerSideApplyAttachment",
							"apiVersion": "integration.test.io/v1alpha1",
							"metadata": map[string]interface{}{
								"name":      attachmentName,
								"namespace": namespaceName,
								"labels": map[string]interface{}{
									"ext-lbl-1": "test-1",
									// overrides controller's desired state
									// since there is no update strategy
									"app": "test",
								},
								"annotations": map[string]interface{}{
									"ext-ann-1": "test-1",
									"app":       "metac",
								},
								"finalizers": []interface{}{
									"test-protect-1",
									"test-protect-2",
								},
							},
						},
					},
				},
			},
			framework.TestStep{
				Name: "assert-server-side-apply-of-attachment-part-1",
				Assert: &framework.Assert{
					State: &unstructured.Unstructured{
						Object: map[string]interface{}{
							"kind":       "ServerSideApplyAttachment",
							"apiVersion": "integration.test.io/v1alpha1",
							"metadata": map[string]interface{}{
								"name":      attachmentName,
								"namespace": namespaceName,
								"labels": map[string]interface{}{
									"ext-lbl-1": "test-1",
									"app":       "test",
								},
								"annotations": map[string]interface{}{
									"ext-ann-1": "test-1",
									"app":       "metac",
								},
								"finalizers": []interface{}{
									"test-protect-1",
									"test-protect-2",
								},
							},
						},
					},
				},
			},
			framework.TestStep{
				Name: "unset-attachment-lbls-anns-finalizers-externally",
				Apply: framework.Apply{
					State: &unstructured.Unstructured{
						Object: map[string]interface{}{
							"kind":       "ServerSideApplyAttachment",
							"apiVersion": "integration.test.io/v1alpha1",
							"metadata": map[string]interface{}{
								"name":      attachmentName,
								"namespace": namespaceName,
								"labels": map[string]interface{}{
									"ext-lbl-1": nil,
									"app":       nil,
								},
								"annotations": map[string]interface{}{
									"ext-ann-1": nil,
									"app":       nil,
								},
								"finalizers": []interface{}{},
							},
						},
					},
				},
			},
			framework.TestStep{
				Name: "assert-server-side-apply-of-attachment-part-2",
				Assert: &framework.Assert{
					State: &unstructured.Unstructured{
						Object: map[string]interface{}{
							"kind":       "ServerSideApplyAttachment",
							"apiVersion": "integration.test.io/v1alpha1",
							"metadata": map[string]interface{}{
								"name":      attachmentName,
								"namespace": namespaceName,
								"labels": map[string]interface{}{
									"ext-lbl-1": "",
									"app":       "",
								},
								"annotations": map[string]interface{}{
									"ext-ann-1": "",
									"app":       "",
								},
							},
						},
					},
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("Test failed: %+v", err)
	}
	if result.Phase == framework.TestStepResultFailed {
		t.Fatalf("Test failed:\n%s", result)
	}
	klog.Infof("Test passed:\n%s", result)
}
