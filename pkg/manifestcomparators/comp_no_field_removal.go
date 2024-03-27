package manifestcomparators

import (
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

type noFieldRemoval struct{}

func NoFieldRemoval() CRDComparator {
	return noFieldRemoval{}
}

func (noFieldRemoval) Name() string {
	return "NoFieldRemoval"
}

func (noFieldRemoval) WhyItMatters() string {
	return "If fields are removed, then clients that rely on those fields will not be able to read them or write them."
}

func getFieldsAndEnums(version *apiextensionsv1.CustomResourceDefinitionVersion) (sets.String, map[string]sets.String) {
	fields := sets.NewString()
	enumsMap := make(map[string]sets.String)
	SchemaHas(version.Schema.OpenAPIV3Schema, field.NewPath("^"), field.NewPath("^"), nil,
		func(s *apiextensionsv1.JSONSchemaProps, fldPath, simpleLocation *field.Path, _ []*apiextensionsv1.JSONSchemaProps) bool {
			fields.Insert(simpleLocation.String())
			for _, enum := range s.Enum {
				_, exists := enumsMap[simpleLocation.String()]
				if !exists {
					enumsMap[simpleLocation.String()] = sets.NewString()
				}
				enumsMap[simpleLocation.String()].Insert(string(enum.Raw))
			}
			return false
		})

	return fields, enumsMap

}

func (b noFieldRemoval) Compare(existingCRD, newCRD *apiextensionsv1.CustomResourceDefinition) (ComparisonResults, error) {
	if existingCRD == nil {
		return ComparisonResults{
			Name:         b.Name(),
			WhyItMatters: b.WhyItMatters(),

			Errors:   nil,
			Warnings: nil,
			Infos:    nil,
		}, nil
	}
	errsToReport := []string{}

	for _, newVersion := range newCRD.Spec.Versions {

		existingVersion := GetVersionByName(existingCRD, newVersion.Name)
		if existingVersion == nil {
			continue
		}

		existingFields, existingEnumsMap := getFieldsAndEnums(existingVersion)
		newFields, newEnumsMap := getFieldsAndEnums(&newVersion)

		removedFields := existingFields.Difference(newFields)
		for _, removedField := range removedFields.List() {
			errsToReport = append(errsToReport, fmt.Sprintf("crd/%v version/%v field/%v may not be removed", newCRD.Name, newVersion.Name, removedField))
		}

		for field, existingEnums := range existingEnumsMap {
			newEnums, exists := newEnumsMap[field]
			if exists {
				removedEnums := existingEnums.Difference(newEnums)
				for _, removedEnum := range removedEnums.List() {
					errsToReport = append(errsToReport, fmt.Sprintf("crd/%v version/%v enum/%v may not be removed for field/%v", newCRD.Name, newVersion.Name, removedEnum, field))
				}
			}
		}

	}

	return ComparisonResults{
		Name:         b.Name(),
		WhyItMatters: b.WhyItMatters(),

		Errors:   errsToReport,
		Warnings: nil,
		Infos:    nil,
	}, nil
}
