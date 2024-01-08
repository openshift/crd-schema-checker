package manifestcomparators

import (
	"fmt"
	"log"
	"sort"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/version"
)

type noFieldTypeChange struct{}

func NoFieldTypeChange() CRDComparator {
	return noFieldTypeChange{}
}

func (noFieldTypeChange) Name() string {
	return "NoFieldTypeChange"
}

func (noFieldTypeChange) WhyItMatters() string {
	return "If a field's type is changed, then clients that rely on that field will likely encounter errors."
}

func (b noFieldTypeChange) noError() ComparisonResults {
	return ComparisonResults{
		Name:         b.Name(),
		WhyItMatters: b.WhyItMatters(),

		Errors:   nil,
		Warnings: nil,
		Infos:    nil,
	}
}
func (b noFieldTypeChange) Compare(existingCRD, newCRD *apiextensionsv1.CustomResourceDefinition) (ComparisonResults, error) {
	if existingCRD == nil {
		return b.noError(), nil
	}
	if len(existingCRD.Spec.Versions) == 0 {
		return b.noError(), nil
	}

	existingVersions := make([]string, len(existingCRD.Spec.Versions))
	for i, existingVersion := range existingCRD.Spec.Versions {
		existingVersions[i] = existingVersion.Name
	}

	sort.SliceStable(existingVersions, func(i, j int) bool {
		return version.CompareKubeAwareVersionStrings(existingVersions[i], existingVersions[j]) < 0
	})

	fields := make(map[string]string)
	for _, existingVersion := range existingCRD.Spec.Versions {
		log.Printf("Parsing existing version %s\n", existingVersion.Name)
		SchemaHas(existingVersion.Schema.OpenAPIV3Schema, field.NewPath("^"), field.NewPath("^"), nil,
			func(s *apiextensionsv1.JSONSchemaProps, fldPath, simpleLocation *field.Path, _ []*apiextensionsv1.JSONSchemaProps) bool {
				if strings.HasSuffix(simpleLocation.String(), "[*]") {
					log.Printf("Skipping array intermediate %v\n", simpleLocation)
					return false
				}
				log.Printf("Checking for existence of path %v\n", simpleLocation)
				_, found := fields[simpleLocation.String()]
				if !found {
					log.Printf("Path %v not found; adding\n", simpleLocation)
					fields[simpleLocation.String()] = existingVersion.Name
					return false
				}
				return false
			},
		)
	}

	var errors []string

	for _, newVersion := range newCRD.Spec.Versions {
		log.Printf("Parsing new version %s\n", newVersion.Name)
		SchemaHas(newVersion.Schema.OpenAPIV3Schema, field.NewPath("^"), field.NewPath("^"), nil,
			func(s *apiextensionsv1.JSONSchemaProps, fldPath, simpleLocation *field.Path, _ []*apiextensionsv1.JSONSchemaProps) bool {
				if strings.HasSuffix(simpleLocation.String(), "[*]") {
					log.Printf("Skipping array intermediate %v\n", simpleLocation)
					return false
				}

				log.Printf("Checking for existence of path %v\n", simpleLocation)
				initialVersion, found := fields[simpleLocation.String()]
				if !found {
					log.Printf("Path %v not found; adding\n", simpleLocation)
					fields[simpleLocation.String()] = newVersion.Name
					return false
				}

				existingVersion := GetVersionByName(existingCRD, initialVersion)
				existingRootSchema := existingVersion.Schema.OpenAPIV3Schema
				field := simpleLocation.String()
				fields := strings.Split(field, ".")
				fieldSchema, err := getSchemaForField(existingRootSchema, fields[1:]...)
				if err != nil {
					panic(err)
				}

				log.Println("Checking for type change")
				if err := ValidateJSONSchemaCompatibility(simpleLocation, fieldSchema, s); err != nil {
					log.Println("Got an error")
					errors = append(errors, err.Error())
				}

				return false
			},
		)
	}

	return ComparisonResults{
		Name:         b.Name(),
		WhyItMatters: b.WhyItMatters(),

		Errors:   errors,
		Warnings: nil,
		Infos:    nil,
	}, nil
}

// FIXME this came from kcp
func getSchemaForField(s *apiextensionsv1.JSONSchemaProps, fields ...string) (*apiextensionsv1.JSONSchemaProps, error) {
	// Cursor keeps track of the current schema subtree as we navigate down through each field segment. We start at the
	// root of the object (which has apiVersion, metadata, spec, status, etc.).
	cursor := s

	// Keep track of which fields we've already visited, so we can be specific in our errors
	visited := make([]string, 0, len(fields))

	// Starting with the initial field (e.g. "spec"), try to resolve the next segment (e.g. "name") until we get
	// to the desired field (e.g. "first").
	for _, f := range fields {
		// Verify that each intermediate field is an object or array.
		switch cursor.Type {
		case "object", "array":
		default:
			return nil, fmt.Errorf("expected field %q to be an object or arry", strings.Join(visited, "."))
		}

		visited = append(visited, f)

		if strings.HasSuffix(f, "[*]") {
			arrayFieldName := strings.TrimSuffix(f, "[*]")
			property := getField(cursor, arrayFieldName)
			if property == nil {
				return nil, fmt.Errorf("field %q doesn't exist", strings.Join(visited, "."))
			}
			cursor = property.Items.Schema
			continue
		}

		if property := getField(cursor, f); property != nil {
			cursor = property
			continue
		}

		// The field didn't exist in either properties or additional properties
		return nil, fmt.Errorf("field %q doesn't exist", strings.Join(visited, "."))
	}

	// Cursor is now set to the schema representing the desired field.
	return cursor, nil
}

func getField(s *apiextensionsv1.JSONSchemaProps, f string) *apiextensionsv1.JSONSchemaProps {
	// First, check properties
	if property, exists := s.Properties[f]; exists {
		return &property
	}

	// Second, check additional properties
	if s.AdditionalProperties != nil && s.AdditionalProperties.Schema != nil {
		if property, exists := s.AdditionalProperties.Schema.Properties[f]; exists {
			return &property
		}
	}

	// The field didn't exist in either properties or additional properties
	return nil
}
