package admissionevaluator

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/openshift/crd-schema-checker/pkg/cmd/options"
	admissionv1 "k8s.io/api/admission/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

type AdmissionHook struct {
	ComparatorConfig *options.ComparatorConfig
}

// where to host it
func (a *AdmissionHook) ValidatingResource() (plural schema.GroupVersionResource, singular string) {
	return schema.GroupVersionResource{
		Group:    "admission.api.openshift.io",
		Version:  "v1",
		Resource: "crdextendedvalidations",
	}, "crdextendedvalidation"
}

// your business logic
func (a *AdmissionHook) Validate(admissionSpec *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	if admissionSpec.Operation != admissionv1.Create && admissionSpec.Operation != admissionv1.Update {
		return &admissionv1.AdmissionResponse{Allowed: true}
	}
	if len(admissionSpec.SubResource) > 0 {
		return &admissionv1.AdmissionResponse{Allowed: true}
	}
	if !(admissionSpec.Resource.Group == apiextensionsv1.GroupName) {
		return &admissionv1.AdmissionResponse{Allowed: true}
	}

	status := &admissionv1.AdmissionResponse{}

	newCRD := &apiextensionsv1.CustomResourceDefinition{}
	err := json.Unmarshal(admissionSpec.Object.Raw, newCRD)
	if err != nil {
		status.Allowed = false
		status.Result = &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
			Message: fmt.Sprintf("failed to unmarshal newCRD: %v", err.Error()),
		}
		return status
	}

	existingCRD := &apiextensionsv1.CustomResourceDefinition{}
	if len(admissionSpec.OldObject.Raw) > 0 {
		err := json.Unmarshal(admissionSpec.OldObject.Raw, existingCRD)
		if err != nil {
			status.Allowed = false
			status.Result = &metav1.Status{
				Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
				Message: fmt.Sprintf("failed to unmarshal oldCRD: %v", err.Error()),
			}
			return status
		}
	}

	comparisonResults, errs := a.ComparatorConfig.ComparatorRegistry.Compare(existingCRD, newCRD, a.ComparatorConfig.ComparatorNames...)
	if len(errs) > 0 {
		status.Allowed = false
		status.Result = &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
			Details: &metav1.StatusDetails{},
			Message: "individual errors in details",
		}
		for _, err := range errs {
			status.Result.Details.Causes = append(status.Result.Details.Causes,
				metav1.StatusCause{
					Type:    "EvaluationError",
					Message: err.Error(),
				})
		}
		return status
	}

	errorCauses := []metav1.StatusCause{}
	for _, comparisonResult := range comparisonResults {
		for _, msg := range comparisonResult.Errors {
			status.Allowed = false
			errorCauses = append(errorCauses,
				metav1.StatusCause{
					Type:    metav1.CauseType(comparisonResult.Name),
					Message: msg,
				})
		}
	}
	if len(errorCauses) > 0 {
		status.Result = &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
			Details: &metav1.StatusDetails{
				Causes: errorCauses,
			},
			Message: "individual errors in details",
		}
	}

	for _, comparisonResult := range comparisonResults {
		for _, msg := range comparisonResult.Warnings {
			status.Warnings = append(status.Warnings, fmt.Sprintf("%q: %v", comparisonResult.Name, msg))
		}
	}
	for _, comparisonResult := range comparisonResults {
		for _, msg := range comparisonResult.Infos {
			// better than nothing for info?
			status.Warnings = append(status.Warnings, fmt.Sprintf("fyi: %q: %v", comparisonResult.Name, msg))
		}
	}

	return status
}

// any special initialization goes here
func (a *AdmissionHook) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	if a.ComparatorConfig == nil {
		return fmt.Errorf("missing ComparatorConfig")
	}
	return nil
}
