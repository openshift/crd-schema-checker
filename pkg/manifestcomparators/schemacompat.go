/*
Copyright 2021 The KCP Authors.
Modifications Copyright 2023 the OpenShift Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package manifestcomparators

import (
	"encoding/json"
	"fmt"
	"reflect"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateJSONSchemaCompatibility compares existing and new revisions of a JSON schema, making sure there are no
// backwards-incompatible changes. Some checks may not have been implemented yet; for these cases, an error is returned
// rather than allowing a potential incompatible change to go through undetected.
func ValidateJSONSchemaCompatibility(fldPath *field.Path, existing, new *apiextensionsv1.JSONSchemaProps) error {
	var newInternal, existingInternal apiextensions.JSONSchemaProps
	if err := apiextensionsv1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(existing, &existingInternal, nil); err != nil {
		return err
	}
	if err := apiextensionsv1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(new, &newInternal, nil); err != nil {
		return err
	}
	newStructural, err := schema.NewStructural(&newInternal)
	if err != nil {
		return err
	}

	existingStructural, err := schema.NewStructural(&existingInternal)
	if err != nil {
		return err
	}

	return validateStructuralSchemaCompatibility(fldPath, existingStructural, newStructural)
}

func validateStructuralSchemaCompatibility(fldPath *field.Path, existing, new *schema.Structural) error {
	if new == nil {
		return field.Invalid(fldPath, nil, "new schema doesn't allow anything")
	}
	if was, now := existing.XPreserveUnknownFields, new.XPreserveUnknownFields; was != now {
		return field.Invalid(fldPath.Child("x-kubernetes-preserve-unknown-fields"), new.XPreserveUnknownFields, fmt.Sprintf("x-kubernetes-preserve-unknown-fields value changed (was %t, now %t)", was, now))
	}

	switch existing.Type {
	case "number":
		return validateNumberCompatibility(fldPath, existing, new)
	case "integer":
		return validateIntegerCompatibility(fldPath, existing, new)
	case "string":
		return validateStringCompatibility(fldPath, existing, new)
	case "boolean":
		return validateBooleanCompatibility(fldPath, existing, new)
	case "array":
		return validateArrayCompatibility(fldPath, existing, new)
	case "object":
		return validateObjectCompatibility(fldPath, existing, new)
	case "":
		if existing.XIntOrString {
			return validateIntOrStringCompatibility(fldPath, existing, new)
		}
		if existing.XPreserveUnknownFields {
			return validatePreserveUnknownFieldsCompatibility(fldPath, existing, new)
		}
	}

	return field.Invalid(fldPath.Child("type"), existing.Type, "invalid/unsupported type")
}

func checkTypesAreTheSame(fldPath *field.Path, existing, new *schema.Structural) error {
	if new.Type != existing.Type {
		return field.Invalid(fldPath.Child("type"), new.Type, fmt.Sprintf("The type changed (was %q, now %q)", existing.Type, new.Type))
	}
	return nil
}

func errorIfPresent(fldPath *field.Path, existing, new interface{}, validationName, typeName string) error {
	if !reflect.ValueOf(existing).IsZero() || !reflect.ValueOf(new).IsZero() {
		return field.Forbidden(fldPath, fmt.Sprintf("%q does not support %q", typeName, validationName))
	}
	return nil
}

func floatPointersEqual(p1, p2 *float64) bool {
	if p1 == nil && p2 == nil {
		return true
	}
	if p1 != nil && p2 != nil {
		return *p1 == *p2
	}
	return false
}

func intPointersEqual(p1, p2 *int64) bool {
	if p1 == nil && p2 == nil {
		return true
	}
	if p1 != nil && p2 != nil {
		return *p1 == *p2
	}
	return false
}

func stringPointersEqual(p1, p2 *string) bool {
	if p1 == nil && p2 == nil {
		return true
	}
	if p1 != nil && p2 != nil {
		return *p1 == *p2
	}
	return false
}

func validateEnum(fldPath *field.Path, existing, new []schema.JSON) error {
	var existingValues sets.Set[string]
	for _, v := range existing {
		raw, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("error marshaling existing enum value '%v' to JSON: %w", v, err)
		}
		existingValues.Insert(string(raw))
	}

	var newValues sets.Set[string]
	for _, v := range new {
		raw, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("error marshaling new enum value '%v' to JSON: %w", v, err)
		}
		newValues.Insert(string(raw))
	}

	if !newValues.IsSuperset(existingValues) {
		return field.Invalid(fldPath, "TODO", "TODO enum value that was previously valid is no longer in the list")
	}

	return nil
}

func validateMultipleOf(fldPath *field.Path, existing, new *float64) error {
	// Unset on both
	if existing == nil && new == nil {
		return nil
	}

	// Unset on new; less restrictive
	if existing != nil && new == nil {
		return nil
	}

	// Only set on new; more restrictive
	if existing == nil && new != nil {
		return field.Invalid(fldPath, "TODO", "TODO more restrictive; might invalidate some existing data")
	}

	// TODO maybe allow decreasing the multiple to a factor
	if *existing != *new {
		return field.Invalid(fldPath, "TODO", "TODO can't change multiple value")
	}

	return nil
}

func validateMinimum(fldPath *field.Path, existing *float64, existingExclusive bool, new *float64, newExclusive bool) error {
	// Unset on both
	if existing == nil && new == nil {
		return nil
	}

	// Unset on new; less restrictive
	if existing != nil && new == nil {
		return nil
	}

	// Only set on new; more restrictive
	if existing == nil && new != nil {
		return field.Invalid(fldPath, "TODO", "TODO more restrictive; might invalidate some existing data")
	}

	// New lower min is ok
	if *new < *existing {
		return nil
	}

	if *new != *existing {
		return field.Invalid(fldPath, "TODO", "TODO more restrictive; might invalidate some existing data")
	}

	if !existingExclusive && newExclusive {
		return field.Invalid(fldPath, "TODO", "TODO more restrictive; might invalidate some existing data")
	}

	return nil
}

func validateMaximum(fldPath *field.Path, existing *float64, existingExclusive bool, new *float64, newExclusive bool) error {
	// Unset on both
	if existing == nil && new == nil {
		return nil
	}

	// Unset on new; less restrictive
	if existing != nil && new == nil {
		return nil
	}

	// Only set on new; more restrictive
	if existing == nil && new != nil {
		return field.Invalid(fldPath, "TODO", "TODO more restrictive; might invalidate some existing data")
	}

	// New higher max is ok
	if *new > *existing {
		return nil
	}

	if *new != *existing {
		return field.Invalid(fldPath, "TODO", "TODO more restrictive; might invalidate some existing data")
	}

	if !existingExclusive && newExclusive {
		return field.Invalid(fldPath, "TODO", "TODO more restrictive; might invalidate some existing data")
	}

	return nil
}

func validateNumberValueValidationCompatibility(fldPath *field.Path, existing, new *schema.ValueValidation, typeName string) error {
	var errors []error

	errors = append(errors,
		errorIfPresent(fldPath, existing.AllOf, new.AllOf, "allOf", typeName),
		errorIfPresent(fldPath, existing.AnyOf, new.AnyOf, "anyOf", typeName),
		errorIfPresent(fldPath, existing.OneOf, new.OneOf, "oneOf", typeName),
		errorIfPresent(fldPath, existing.Not, new.Not, "not", typeName),
		validateEnum(fldPath.Child("enum"), existing.Enum, new.Enum),
		validateMultipleOf(fldPath.Child("multipleOf"), existing.MultipleOf, new.MultipleOf),
		validateMinimum(fldPath.Child("minimum"), existing.Minimum, existing.ExclusiveMinimum, new.Minimum, new.ExclusiveMinimum),
		validateMinimum(fldPath.Child("maximum"), existing.Maximum, existing.ExclusiveMaximum, new.Maximum, new.ExclusiveMaximum),
	)

	return utilerrors.NewAggregate(errors)
}

func validateNumberCompatibility(fldPath *field.Path, existing, new *schema.Structural) error {
	if err := checkTypesAreTheSame(fldPath, existing, new); err != nil {
		return err
	}

	return validateNumberValueValidationCompatibility(fldPath, existing.ValueValidation, new.ValueValidation, "number")
}

func validateIntegerCompatibility(fldPath *field.Path, existing, new *schema.Structural) error {
	if new.Type == "number" {
		// new type is a superset of the existing type.
	} else if err := checkTypesAreTheSame(fldPath, existing, new); err != nil {
		return err
	}

	return validateNumberValueValidationCompatibility(fldPath, existing.ValueValidation, new.ValueValidation, "integer")
}

func validateStringValueValidationCompatibility(fldPath *field.Path, existing, new *schema.ValueValidation) error {
	var errors []error
	errors = append(errors, errorIfPresent(fldPath, existing.AllOf, new.AllOf, "allOf", "string"))

	if !intPointersEqual(new.MaxLength, existing.MaxLength) || !intPointersEqual(new.MinLength, existing.MinLength) {
		errors = append(errors,
			errorIfPresent(fldPath, existing.MaxLength, new.MaxLength, "maxLength", "string"),
			errorIfPresent(fldPath, existing.MinLength, new.MinLength, "minLength", "string"),
		)
	}

	if new.Pattern != existing.Pattern {
		errors = append(errors, errorIfPresent(fldPath, existing.Pattern, new.Pattern, "pattern", "string"))
	}

	toEnumSets := func(enum []schema.JSON) sets.Set[string] {
		enumSet := sets.New[string]()
		for _, val := range enum {
			strVal, isString := val.Object.(string)
			if !isString {
				errors = append(errors, field.Invalid(fldPath.Child("enum"), enum, fmt.Sprintf("enum value \"%v\" must be a string", val.Object)))
				continue
			}
			enumSet.Insert(strVal)
		}
		return enumSet
	}

	existingEnumValues := toEnumSets(existing.Enum)
	newEnumValues := toEnumSets(new.Enum)

	if !newEnumValues.IsSuperset(existingEnumValues) {
		errors = append(errors, field.Invalid(fldPath.Child("enum"), sets.List[string](newEnumValues.Difference(existingEnumValues)), "enum value has been changed in an incompatible way"))
	}

	if existing.Format != new.Format {
		errors = append(errors, field.Invalid(fldPath.Child("format"), new.Format, "format value has been changed in an incompatible way"))
	}

	return utilerrors.NewAggregate(errors)
}

func validateStringCompatibility(fldPath *field.Path, existing, new *schema.Structural) error {
	var errors []error

	errors = append(errors,
		checkTypesAreTheSame(fldPath, existing, new),
		validateStringValueValidationCompatibility(fldPath, existing.ValueValidation, new.ValueValidation),
	)

	return utilerrors.NewAggregate(errors)
}

func validateBooleanValueValidationCompatibility(fldPath *field.Path, existing, new *schema.ValueValidation) error {
	var errors []error

	errors = append(errors,
		errorIfPresent(fldPath, existing.AllOf, new.AllOf, "allOf", "boolean"),
		errorIfPresent(fldPath, existing.AllOf, new.AllOf, "anyOf", "boolean"),
		errorIfPresent(fldPath, existing.AllOf, new.AllOf, "oneOf", "boolean"),
		errorIfPresent(fldPath, existing.Enum, new.Enum, "enum", "boolean"),
	)

	return utilerrors.NewAggregate(errors)
}

func validateBooleanCompatibility(fldPath *field.Path, existing, new *schema.Structural) error {
	var errors []error

	errors = append(errors,
		checkTypesAreTheSame(fldPath, existing, new),
		validateBooleanValueValidationCompatibility(fldPath, existing.ValueValidation, new.ValueValidation),
	)

	return utilerrors.NewAggregate(errors)
}

func validateArrayValueValidationCompatibility(fldPath *field.Path, existing, new *schema.ValueValidation) error {
	var errors []error

	if !intPointersEqual(new.MaxItems, existing.MaxItems) ||
		!intPointersEqual(new.MinItems, existing.MinItems) {
		errors = append(errors,
			errorIfPresent(fldPath, existing.MaxLength, new.MaxLength, "maxItems", "array"),
			errorIfPresent(fldPath, existing.MinLength, new.MinLength, "minItems", "array"),
		)
	}

	if !existing.UniqueItems && new.UniqueItems {
		errors = append(errors, field.Invalid(fldPath.Child("uniqueItems"), new.UniqueItems, "uniqueItems value has been changed in an incompatible way"))
	}

	return utilerrors.NewAggregate(errors)
}

func validateArrayCompatibility(fldPath *field.Path, existing, new *schema.Structural) error {
	var errors []error

	errors = append(errors,
		checkTypesAreTheSame(fldPath, existing, new),
		validateArrayValueValidationCompatibility(fldPath, existing.ValueValidation, new.ValueValidation),
		validateStructuralSchemaCompatibility(fldPath.Child("Items"), existing.Items, new.Items),
	)

	if !stringPointersEqual(existing.Extensions.XListType, new.Extensions.XListType) {
		errors = append(errors, field.Invalid(fldPath.Child("x-kubernetes-list-type"), new.Extensions.XListType, "x-kubernetes-list-type value has been changed in an incompatible way"))
	}

	if !sets.New[string](existing.Extensions.XListMapKeys...).Equal(sets.New[string](new.Extensions.XListMapKeys...)) {
		errors = append(errors, field.Invalid(fldPath.Child("x-kubernetes-list-map-keys"), new.Extensions.XListType, "x-kubernetes-list-map-keys value has been changed in an incompatible way"))
	}

	return utilerrors.NewAggregate(errors)
}

func validateObjectValueValidationCompatibility(fldPath *field.Path, existing, new *schema.ValueValidation) error {
	var errors []error

	errors = append(errors,
		errorIfPresent(fldPath, existing.AllOf, new.AllOf, "allOf", "object"),
		errorIfPresent(fldPath, existing.AllOf, new.AllOf, "anyOf", "object"),
		errorIfPresent(fldPath, existing.AllOf, new.AllOf, "oneOf", "object"),
		errorIfPresent(fldPath, existing.Enum, new.Enum, "enum", "object"),
	)

	return utilerrors.NewAggregate(errors)
}

func validateObjectCompatibility(fldPath *field.Path, existing, new *schema.Structural) error {
	var errors []error

	errors = append(errors, checkTypesAreTheSame(fldPath, existing, new))

	if !stringPointersEqual(existing.Extensions.XMapType, new.Extensions.XMapType) {
		errors = append(errors, field.Invalid(fldPath.Child("x-kubernetes-map-type"), new.Extensions.XListType, "x-kubernetes-map-type value has been changed in an incompatible way"))
	}

	// Let's keep in mind that, in structural schemas, properties and additionalProperties are mutually exclusive,
	// which greatly simplifies the logic here.
	if len(existing.Properties) > 0 {
		if len(new.Properties) > 0 {
			existingProperties := sets.StringKeySet(existing.Properties)
			newProperties := sets.StringKeySet(new.Properties)
			if !newProperties.IsSuperset(existingProperties) {
				errors = append(errors, field.Invalid(fldPath.Child("properties"), existingProperties.Difference(newProperties).List(), "properties have been removed in an incompatible way"))
			}
		} else if new.AdditionalProperties != nil && new.AdditionalProperties.Structural != nil {
			for _, key := range sets.StringKeySet(existing.Properties).List() {
				existingPropertySchema := existing.Properties[key]
				errors = append(errors, validateStructuralSchemaCompatibility(fldPath.Child("properties").Key(key), &existingPropertySchema, new.AdditionalProperties.Structural))
			}
		} else if new.AdditionalProperties != nil && new.AdditionalProperties.Bool {
			// that allows named properties only.
			// => Keep the existing schemas as the lcd.
		} else {
			errors = append(errors, field.Invalid(fldPath.Child("properties"), sets.StringKeySet(existing.Properties).List(), "properties value has been completely cleared in an incompatible way"))
		}
	} else if existing.AdditionalProperties != nil {
		if existing.AdditionalProperties.Structural != nil {
			if new.AdditionalProperties.Structural != nil {
				errors = append(errors, validateStructuralSchemaCompatibility(fldPath.Child("additionalProperties"), existing.AdditionalProperties.Structural, new.AdditionalProperties.Structural))
			} else if existing.AdditionalProperties != nil && new.AdditionalProperties.Bool {
				// new schema allows any properties of any schema here => it is a superset of the existing schema
				// that allows any properties of a given schema.
				// => Keep the existing schemas as the lcd.
			} else {
				errors = append(errors, field.Invalid(fldPath.Child("additionalProperties"), new.AdditionalProperties.Bool, "additionalProperties value has been changed in an incompatible way"))
			}
		} else if existing.AdditionalProperties.Bool {
			if !new.AdditionalProperties.Bool {
				errors = append(errors, field.Invalid(fldPath.Child("additionalProperties"), new.AdditionalProperties.Bool, "additionalProperties value has been changed in an incompatible way"))
			}
		}
	}

	errors = append(errors, validateObjectValueValidationCompatibility(fldPath, existing.ValueValidation, new.ValueValidation))

	return utilerrors.NewAggregate(errors)
}

func validateIntOrStringCompatibility(fldPath *field.Path, existing, new *schema.Structural) error {
	var errors []error

	errors = append(errors, checkTypesAreTheSame(fldPath, existing, new))

	if !new.XIntOrString {
		errors = append(errors, field.Invalid(fldPath.Child("x-kubernetes-int-or-string"), new.XIntOrString, "x-kubernetes-int-or-string value has been changed in an incompatible way"))
	}

	errors = append(errors,
		validateStringValueValidationCompatibility(fldPath, existing.ValueValidation, new.ValueValidation),
		validateNumberValueValidationCompatibility(fldPath, existing.ValueValidation, new.ValueValidation, "x-kubernetes-int-or-string"),
	)

	return utilerrors.NewAggregate(errors)
}

func validatePreserveUnknownFieldsCompatibility(fldPath *field.Path, existing, new *schema.Structural) error {
	return checkTypesAreTheSame(fldPath, existing, new)
}
