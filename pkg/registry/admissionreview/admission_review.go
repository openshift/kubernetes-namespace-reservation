package admissionreview

import (
	"fmt"

	"net/http"

	"encoding/json"

	admissionv1alpha1 "k8s.io/api/admission/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
)

type REST struct {
}

var _ rest.Creater = &REST{}

func NewREST() *REST {
	return &REST{}
}

func (r *REST) New() runtime.Object {
	return &admissionv1alpha1.AdmissionReview{}
}

func (r *REST) Create(ctx apirequest.Context, obj runtime.Object, _ bool) (runtime.Object, error) {
	fmt.Printf("#### got %#v\n", obj)

	admissionReview := obj.(*admissionv1alpha1.AdmissionReview)
	if admissionReview.Spec.Resource.Group != "project.openshift.io" ||
		admissionReview.Spec.Resource.Resource != "projectrequests" ||
		len(admissionReview.Spec.SubResource) != 0 ||
		admissionReview.Spec.Operation != admissionv1alpha1.Create {

		admissionReview.Status.Allowed = true
		return admissionReview, nil
	}

	admittingObjectName := &NamedThing{}
	err := json.Unmarshal(admissionReview.Spec.Object.Raw, admittingObjectName)
	if err != nil {
		return nil, errors.NewBadRequest(err.Error())
	}

	if len(admittingObjectName.Name) == 0 {
		admissionReview.Status.Allowed = false
		admissionReview.Status.Result = &metav1.Status{
			Status:  metav1.StatusFailure,
			Code:    http.StatusForbidden,
			Reason:  metav1.StatusReasonForbidden,
			Message: "name is required",
		}
		return admissionReview, nil
	}

	if admittingObjectName.Name == "fail-me" {
		admissionReview.Status.Allowed = false
		admissionReview.Status.Result = &metav1.Status{
			Status:  metav1.StatusFailure,
			Code:    http.StatusForbidden,
			Reason:  metav1.StatusReasonForbidden,
			Message: fmt.Sprintf("%q is reserved", admittingObjectName.Name),
		}
		return admissionReview, nil
	}

	admissionReview.Status.Allowed = true
	return admissionReview, nil
}

type NamedThing struct {
	Name string `json:name`
}
