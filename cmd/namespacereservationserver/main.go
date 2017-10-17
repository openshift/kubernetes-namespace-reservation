package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	admissionv1alpha1 "k8s.io/api/admission/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	"sync"

	"github.com/openshift/generic-admission-server/pkg/cmd"
)

func main() {
	cmd.RunAdmission(&admissionHook{})
}

type admissionHook struct {
	reservationClient dynamic.ResourceInterface

	lock        sync.RWMutex
	initialized bool
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
			Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
			Message: err.Error(),
		}
		return status
	}
	if len(admittingObjectName.Name) == 0 {
		status.Allowed = false
		status.Result = &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
			Message: "name is required",
		}
		return status
	}

	a.lock.RLock()
	defer a.lock.RUnlock()
	if !a.initialized {
		status.Allowed = false
		status.Result = &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusInternalServerError, Reason: metav1.StatusReasonInternalError,
			Message: "not initialized",
		}
		return status
	}

	_, err = a.reservationClient.Get(admittingObjectName.Name, metav1.GetOptions{})
	if err == nil {
		status.Allowed = false
		status.Result = &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
			Message: fmt.Sprintf("%q is reserved", admittingObjectName.Name),
		}
		return status
	}
	if apierrors.IsNotFound(err) {
		status.Allowed = true
		return status
	}

	status.Allowed = false
	status.Result = &metav1.Status{
		Status: metav1.StatusFailure, Code: http.StatusInternalServerError, Reason: metav1.StatusReasonInternalError,
		Message: err.Error(),
	}
	return status
}

func (a *admissionHook) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.initialized = true

	shallowClientConfigCopy := *kubeClientConfig
	shallowClientConfigCopy.GroupVersion = &schema.GroupVersion{
		Group:   "online.openshift.io",
		Version: "v1alpha1",
	}
	shallowClientConfigCopy.APIPath = "/apis"
	dynamicClient, err := dynamic.NewClient(&shallowClientConfigCopy)
	if err != nil {
		return err
	}
	a.reservationClient = dynamicClient.Resource(
		&metav1.APIResource{
			Name:       "namespacereservations",
			Namespaced: false,
			Group:      "online.openshift.io",
			Version:    "v1alpha1",
			// kind is the kind for the resource (e.g. 'Foo' is the kind for a resource 'foo')
			Kind: "NamespaceReservation",
		},
		"",
	)

	return nil
}

type NamedThing struct {
	Name string `json:name`
}
