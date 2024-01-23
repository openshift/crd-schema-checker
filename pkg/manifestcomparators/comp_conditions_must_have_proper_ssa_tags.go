package manifestcomparators

import (
	"fmt"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

type conditionsMustHaveProperSSATags struct{}

func ConditionsMustHaveProperSSATags() CRDComparator {
	return conditionsMustHaveProperSSATags{}
}

func (conditionsMustHaveProperSSATags) Name() string {
	return "ConditionsMustHaveProperSSATags"
}

func (conditionsMustHaveProperSSATags) WhyItMatters() string {
	return "Collection of conditions should be treated as a map with a key of type."
}

func (c conditionsMustHaveProperSSATags) Validate(crd *apiextensionsv1.CustomResourceDefinition) (ComparisonResults, error) {
	errsToReport := []string{}

	for _, newVersion := range crd.Spec.Versions {
		conditionsWithoutMapListType := []string{}
		conditionsWithoutListMapKeysType := []string{}
		SchemaHas(newVersion.Schema.OpenAPIV3Schema, field.NewPath("^"), field.NewPath("^"), nil,
			func(s *apiextensionsv1.JSONSchemaProps, fldPath, simpleLocation *field.Path, _ []*apiextensionsv1.JSONSchemaProps) bool {
				if s.Type != "array" {
					return false
				}

				if !strings.Contains(simpleLocation.String(), ".conditions") {
					return false
				}

				if !areConditionPropertiesPresent(s.Items.Schema.Properties) {
					return false
				}

				if s.XListType == nil || *s.XListType != "map" {
					conditionsWithoutMapListType = append(conditionsWithoutMapListType, simpleLocation.String())
				}

				if len(s.XListMapKeys) == 0 || !containsString(s.XListMapKeys, "type") {
					conditionsWithoutListMapKeysType = append(conditionsWithoutListMapKeysType, simpleLocation.String())
				}

				return false
			})

		for _, affectedField := range conditionsWithoutMapListType {
			errStr := fmt.Sprintf("crd/%v version/%v field/%v must set x-kubernetes-list-type with value \"map\"", crd.Name, newVersion.Name, affectedField)
			errsToReport = append(errsToReport, errStr)
		}
		for _, affectedField := range conditionsWithoutListMapKeysType {
			errStr := fmt.Sprintf("crd/%v version/%v field/%v must set x-kubernetes-list-map-keys containing value \"type\"", crd.Name, newVersion.Name, affectedField)
			errsToReport = append(errsToReport, errStr)
		}
	}

	return ComparisonResults{
		Name:         c.Name(),
		WhyItMatters: c.WhyItMatters(),

		Errors:   errsToReport,
		Warnings: nil,
		Infos:    nil,
	}, nil
}

func (b conditionsMustHaveProperSSATags) Compare(existingCRD, newCRD *apiextensionsv1.CustomResourceDefinition) (ComparisonResults, error) {
	return RatchetCompare(b, existingCRD, newCRD)
}

func containsString(a []string, s string) bool {
	for _, v := range a {
		if v == s {
			return true
		}
	}
	return false
}

func areConditionPropertiesPresent(properties map[string]apiextensionsv1.JSONSchemaProps) bool {
	expectedConditionProperties := []string{"type", "reason", "status", "observedGeneration", "lastTransitionTime"}

	for _, p := range expectedConditionProperties {
		_, ok := properties[p]
		if !ok {
			return false
		}
	}
	return true
}
