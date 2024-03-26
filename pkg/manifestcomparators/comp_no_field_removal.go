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

		existingFields := sets.NewString()
		existingEnumsMap := make(map[string]sets.String)
		SchemaHas(existingVersion.Schema.OpenAPIV3Schema, field.NewPath("^"), field.NewPath("^"), nil,
			func(s *apiextensionsv1.JSONSchemaProps, fldPath, simpleLocation *field.Path, _ []*apiextensionsv1.JSONSchemaProps) bool {
				existingFields.Insert(simpleLocation.String())
				for _, enum := range s.Enum {
					_, exists := existingEnumsMap[simpleLocation.String()]
					if !exists {
						existingEnumsMap[simpleLocation.String()] = sets.NewString()
					}
					existingEnumsMap[simpleLocation.String()].Insert(string(enum.Raw))
				}
				return false
			})

		newFields := sets.NewString()
		newEnumsMap := make(map[string]sets.String)
		SchemaHas(newVersion.Schema.OpenAPIV3Schema, field.NewPath("^"), field.NewPath("^"), nil,
			func(s *apiextensionsv1.JSONSchemaProps, fldPath, simpleLocation *field.Path, _ []*apiextensionsv1.JSONSchemaProps) bool {
				newFields.Insert(simpleLocation.String())
				for _, enum := range s.Enum {
					_, exists := newEnumsMap[simpleLocation.String()]
					if !exists {
						newEnumsMap[simpleLocation.String()] = sets.NewString()
					}
					newEnumsMap[simpleLocation.String()].Insert(string(enum.Raw))
				}
				return false
			})

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
