package options

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/openshift/crd-schema-checker/pkg/defaultcomparators"
	"github.com/openshift/crd-schema-checker/pkg/manifestcomparators"
	"github.com/spf13/pflag"
)

type ComparatorOptions struct {
	ComparatorRegistry        manifestcomparators.CRDComparatorRegistry
	KnownComparators          []string
	DefaultEnabledComparators []string
	EnabledComparators        []string
	DisabledComparators       []string
	ComparatorLabels          []string
	ComparatorsMatchingLabels []string
}

func NewComparatorOptions() *ComparatorOptions {
	o := &ComparatorOptions{
		ComparatorRegistry: defaultcomparators.NewDefaultComparators(),
	}
	o.KnownComparators = o.ComparatorRegistry.KnownComparators()

	// TODO, we have the ability to change this default list at some point
	o.DefaultEnabledComparators = o.ComparatorRegistry.KnownComparators()

	return o
}

func (o *ComparatorOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringSliceVar(&o.DisabledComparators, "disabled-validators", o.DisabledComparators, "list of comparators that must be disabled")
	fs.StringSliceVar(&o.EnabledComparators, "enabled-validators", o.EnabledComparators, "list of comparators that must be enabled")
	fs.StringSliceVar(&o.ComparatorLabels, "labels", o.ComparatorLabels, "comparators with matching labels will be enabled")
}

func (o *ComparatorOptions) Validate() error {
	knownComparators := sets.NewString(o.KnownComparators...)
	disabledComparators := sets.NewString(o.DisabledComparators...)
	enabledComparators := sets.NewString(o.EnabledComparators...)

	if diff := disabledComparators.Difference(knownComparators); len(diff) > 0 {
		return fmt.Errorf("unknown comparators: %v", disabledComparators.List())
	}
	if diff := enabledComparators.Difference(knownComparators); len(diff) > 0 {
		return fmt.Errorf("unknown comparators: %v", disabledComparators.List())
	}

	return nil
}

// Complete fills in missing values before command execution.
func (o *ComparatorOptions) Complete() (*ComparatorConfig, error) {
	ret := &ComparatorConfig{
		ComparatorRegistry: o.ComparatorRegistry,
	}

	knownComparators := sets.NewString(o.KnownComparators...)
	disabledComparators := sets.NewString(o.DisabledComparators...)
	enabledComparators := sets.NewString(o.EnabledComparators...)

	if diff := disabledComparators.Difference(knownComparators); len(diff) > 0 {
		return nil, fmt.Errorf("unknown comparators: %v", disabledComparators.List())
	}
	if diff := enabledComparators.Difference(knownComparators); len(diff) > 0 {
		return nil, fmt.Errorf("unknown comparators: %v", disabledComparators.List())
	}

	o.ComparatorsMatchingLabels = o.ComparatorRegistry.ComparatorsMatchingLabels(o.ComparatorLabels)
	var baseComparators sets.String
	if len(o.ComparatorLabels) == 0 {
		baseComparators = sets.NewString(o.DefaultEnabledComparators...)
	} else {
		baseComparators = sets.NewString(o.ComparatorsMatchingLabels...)
	}
	comparatorsToRun := baseComparators.Insert(o.EnabledComparators...).Delete(o.DisabledComparators...)
	ret.ComparatorNames = comparatorsToRun.List()

	return ret, nil
}

type ComparatorConfig struct {
	ComparatorRegistry manifestcomparators.CRDComparatorRegistry
	ComparatorNames    []string
}
