package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	admissionv1alpha1 "k8s.io/api/admission/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	genericapiserver "k8s.io/apiserver/pkg/server"

	"github.com/openshift/kubernetes-namespace-reservation/pkg/genericadmissionserver/cmd"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func main() {
	cmd.RunAdmission(&admissionHook{})
}

type admissionHook struct {
}

func (a *admissionHook) Resource() (plural schema.GroupVersionResource, singular string) {
	return schema.GroupVersionResource{
			Group:    "admission.online.openshift.io",
			Version:  "v1alpha1",
			Resource: "namespacereservations",
		},
		"namespacereservation"
}

func (a *admissionHook) Admit(admissionSpec admissionv1alpha1.AdmissionReviewSpec) admissionv1alpha1.AdmissionReviewStatus {
	status := admissionv1alpha1.AdmissionReviewStatus{}

	if admissionSpec.Resource.Group != "project.openshift.io" ||
		admissionSpec.Resource.Resource != "projectrequests" ||
		len(admissionSpec.SubResource) != 0 ||
		admissionSpec.Operation != admissionv1alpha1.Create {

		status.Allowed = true
		return status
	}

	admittingObjectName := &NamedThing{}
	err := json.Unmarshal(admissionSpec.Object.Raw, admittingObjectName)
	if err != nil {
		status.Allowed = false
		status.Result = &metav1.Status{
			Status:  metav1.StatusFailure,
			Code:    http.StatusBadRequest,
			Reason:  metav1.StatusReasonBadRequest,
			Message: err.Error(),
		}
		return status
	}

	if len(admittingObjectName.Name) == 0 {
		status.Allowed = false
		status.Result = &metav1.Status{
			Status:  metav1.StatusFailure,
			Code:    http.StatusForbidden,
			Reason:  metav1.StatusReasonForbidden,
			Message: "name is required",
		}
		return status
	}

	if admittingObjectName.Name == "fail-me" {
		status.Allowed = false
		status.Result = &metav1.Status{
			Status:  metav1.StatusFailure,
			Code:    http.StatusForbidden,
			Reason:  metav1.StatusReasonForbidden,
			Message: fmt.Sprintf("%q is reserved", admittingObjectName.Name),
		}
		return status
	}

	status.Allowed = true
	return status
}

func (a *admissionHook) Initialize(context genericapiserver.PostStartHookContext) error {
	return nil
}

type NamedThing struct {
	Name string `json:name`
}
