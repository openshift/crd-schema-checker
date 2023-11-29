package checkadmission

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/openshift/crd-schema-checker/pkg/resourceread"

	"k8s.io/apimachinery/pkg/types"

	v1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/google/uuid"

	"github.com/openshift/crd-schema-checker/pkg/manifestcomparators"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/client-go/rest"
)

type admissionComparatorTest struct {
	restClient     rest.Interface
	ComparatorTest manifestcomparators.ComparatorTest
}

func (tc *admissionComparatorTest) Test(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	admissionReview := &admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Request: &admissionv1.AdmissionRequest{
			UID: types.UID(uuid.New().String()),
			Kind: metav1.GroupVersionKind{
				Group:   apiextensionsv1.SchemeGroupVersion.Group,
				Version: apiextensionsv1.SchemeGroupVersion.Version,
				Kind:    "CustomResourceDefinition",
			},
			Resource: metav1.GroupVersionResource{
				Group:    apiextensionsv1.SchemeGroupVersion.Group,
				Version:  apiextensionsv1.SchemeGroupVersion.Version,
				Resource: "customresourcedefinitions",
			},
			SubResource:        "",
			RequestSubResource: "",
			UserInfo:           v1.UserInfo{},
			Object: runtime.RawExtension{
				Raw: []byte(resourceread.WriteCustomResourceDefinitionV1OrDie(tc.ComparatorTest.NewCRD)),
			},
			DryRun:  nil,
			Options: runtime.RawExtension{},
		},
	}
	if tc.ComparatorTest.ExistingCRD == nil {
		admissionReview.Request.Operation = admissionv1.Create
	} else {
		admissionReview.Request.Operation = admissionv1.Update
		admissionReview.Request.OldObject = runtime.RawExtension{
			Raw: []byte(resourceread.WriteCustomResourceDefinitionV1OrDie(tc.ComparatorTest.ExistingCRD)),
		}
		admissionReview.Request.Name = tc.ComparatorTest.ExistingCRD.Name
	}

	admissionBytes, err := json.Marshal(admissionReview)
	if err != nil {
		t.Fatal(err)
	}
	actual := &admissionv1.AdmissionReview{}
	result := tc.restClient.Post().AbsPath("/apis/admission.api.openshift.io/v1/crdextendedvalidations").Body(admissionBytes).Do(ctx)
	if body, err := result.Raw(); err != nil {
		t.Log(string(body))
		t.Fatal(err)
	}
	err = result.Into(actual)
	if err != nil {
		t.Fatal(err)
	}
	actual.Request = nil

	actualResults := []manifestcomparators.ComparisonResults{}
	actualErrors := []error{}
	comparatorNameToResults := map[string]manifestcomparators.ComparisonResults{}

	if actual.Response.UID != admissionReview.Request.UID {
		t.Fatalf("mismatch of UID: sent %v, got %v", admissionReview.Request.UID, actual.Response.UID)
	}

	t.Logf("request.allowed == %v", actual.Response.Allowed)
	if !actual.Response.Allowed {
		responseJSON, err := json.MarshalIndent(actual.Response, "", "    ")
		if err != nil {
			t.Log(err)
		}
		t.Log(string(responseJSON))
	}

	if actual.Response.Result != nil && actual.Response.Result.Details != nil {
		for _, cause := range actual.Response.Result.Details.Causes {
			t.Log(cause.Message)
			name := string(cause.Type)
			if name == "EvaluationError" {
				actualErrors = append(actualErrors, fmt.Errorf(cause.Message))
				continue
			}
			results := comparatorNameToResults[name]
			results.Name = name
			results.Errors = append(results.Errors, cause.Message)
			comparatorNameToResults[name] = results
		}
	} else {
		switch {
		case actual.Response.Result == nil:
			t.Logf("got no response.result")
		case actual.Response.Result.Details == nil:
			t.Logf("got no response.result.details, but result: %v", actual.Response.Result)
		}

	}

	t.Logf("got %d warnings", len(actual.Response.Warnings))
	for _, warning := range actual.Response.Warnings {
		t.Log(warning)
		if strings.HasPrefix(warning, "fyi: ") {
			parts := strings.SplitN(warning, ":", 3)
			name := parts[1]
			results := comparatorNameToResults[name]
			results.Name = name
			results.Infos = append(results.Infos, parts[2])
			comparatorNameToResults[name] = results
			continue
		}
		parts := strings.SplitN(warning, ":", 2)
		name := parts[0]
		results := comparatorNameToResults[name]
		results.Name = name
		results.Warnings = append(results.Warnings, parts[1])
		comparatorNameToResults[name] = results
	}

	for name := range comparatorNameToResults {
		results := comparatorNameToResults[name]
		actualResults = append(actualResults, results)
	}

	tc.ComparatorTest.Test(t, actualResults, actualErrors)
}
