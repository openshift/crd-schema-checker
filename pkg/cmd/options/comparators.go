package options

import (
	"fmt"
	"path"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/openshift/crd-schema-checker/pkg/defaultcomparators"
	"github.com/openshift/crd-schema-checker/pkg/manifestcomparators"
	"github.com/spf13/pflag"
)

type ComparatorOptions struct {
	ComparatorRegistry          manifestcomparators.CRDComparatorRegistry
	KnownComparators            []string
	DefaultEnabledComparators   []string
	EnabledComparators          []string
	DisabledComparators         []string
	DisabledComparatorWildcards []string
	EnabledComparatorWildcards  []string
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
	fs.StringSliceVar(&o.DisabledComparatorWildcards, "disabled-validators", o.DisabledComparatorWildcards, "list of comparators that must be disabled. May contain wildcards")
	fs.StringSliceVar(&o.EnabledComparatorWildcards, "enabled-validators", o.EnabledComparatorWildcards, "list of comparators that must be enabled. May contain wildcards")
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

func (o *ComparatorOptions) comparatorWildcardsToComparators(comparatorWildcards []string) []string {
	matchingComparators := make([]string, 0)

	for _, known := range o.KnownComparators {
		for _, wildcard := range comparatorWildcards {
			if known == wildcard {
				matchingComparators = append(matchingComparators, known)
				break
			}
			if matched, _ := path.Match(wildcard, known); matched {
				matchingComparators = append(matchingComparators, known)
				break
			}
		}
	}

	return matchingComparators
}

func containsWildcards(comparatorWildcards []string) bool {
	for _, wildcard := range comparatorWildcards {
		if strings.ContainsAny(wildcard, "*") {
			return true
		}
	}
	return false
}

// Complete fills in missing values before command execution.
func (o *ComparatorOptions) Complete() (*ComparatorConfig, error) {
	ret := &ComparatorConfig{
		ComparatorRegistry: o.ComparatorRegistry,
	}

	o.DisabledComparators = o.comparatorWildcardsToComparators(o.DisabledComparatorWildcards)
	o.EnabledComparators = o.comparatorWildcardsToComparators(o.EnabledComparatorWildcards)

	knownComparators := sets.NewString(o.KnownComparators...)
	disabledComparators := sets.NewString(o.DisabledComparators...)
	enabledComparators := sets.NewString(o.EnabledComparators...)

	if diff := disabledComparators.Difference(knownComparators); len(diff) > 0 {
		return nil, fmt.Errorf("unknown comparators: %v", disabledComparators.List())
	}
	if diff := enabledComparators.Difference(knownComparators); len(diff) > 0 {
		return nil, fmt.Errorf("unknown comparators: %v", disabledComparators.List())
	}

	disabledContainsWildcards := containsWildcards(o.DisabledComparatorWildcards)
	enabledContainsWildcards := containsWildcards(o.EnabledComparatorWildcards)

	comparatorsToRun := sets.NewString()

	if disabledContainsWildcards {
		comparatorsToRun = sets.NewString(o.DefaultEnabledComparators...).Delete(o.DisabledComparators...).Insert(o.EnabledComparators...)
	} else if !disabledContainsWildcards && enabledContainsWildcards {
		comparatorsToRun = sets.NewString(o.DefaultEnabledComparators...).Insert(o.EnabledComparators...).Delete(o.DisabledComparators...)
	} else if len(o.EnabledComparators) == 0 && len(o.DisabledComparators) != 0 {
		comparatorsToRun = sets.NewString(o.DefaultEnabledComparators...).Delete(o.DisabledComparators...)
	} else if len(o.EnabledComparators) == 0 && len(o.DisabledComparators) == 0 {
		comparatorsToRun = sets.NewString(o.DefaultEnabledComparators...)
	} else if len(o.EnabledComparators) != 0 && len(o.DisabledComparators) == 0 {
		return nil, fmt.Errorf("Enabling comparators without disabling comparators has no effect. Consider using a wildcard for enabled comparators")
	} else if !disabledContainsWildcards && !enabledContainsWildcards {
		return nil, fmt.Errorf("Cannot both disable and enable comparators if neither of them has a wildcard")
	}

	ret.ComparatorNames = comparatorsToRun.List()

	return ret, nil
}

type ComparatorConfig struct {
	ComparatorRegistry manifestcomparators.CRDComparatorRegistry
	ComparatorNames    []string
}
