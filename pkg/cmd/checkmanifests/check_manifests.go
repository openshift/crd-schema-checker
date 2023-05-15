package checkmanifests

import (
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/openshift/crd-schema-compatibility-checker/pkg/defaultcomparators"
	"github.com/openshift/crd-schema-compatibility-checker/pkg/manifestcomparators"
	"github.com/openshift/crd-schema-compatibility-checker/pkg/resourceread"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
)

type CheckManifestOptions struct {
	ExistingCRDFile string
	NewCRDFile      string

	ComparatorRegistry        manifestcomparators.CRDComparatorRegistry
	KnownComparators          []string
	DefaultEnabledComparators []string
	EnabledComparators        []string
	DisabledComparators       []string

	IOStreams genericclioptions.IOStreams
}

func NewCheckManifestOptions() *CheckManifestOptions {
	o := &CheckManifestOptions{
		ComparatorRegistry: defaultcomparators.NewDefaultComparators(),
		IOStreams: genericclioptions.IOStreams{
			In:     os.Stdin,
			Out:    os.Stdout,
			ErrOut: os.Stderr,
		},
	}
	o.KnownComparators = o.ComparatorRegistry.KnownComparators()

	// TODO, we have the ability to change this default list at some point
	o.DefaultEnabledComparators = o.ComparatorRegistry.KnownComparators()

	return o
}

// NewRenderCommand creates a render command.
func NewCheckManifestsCommand() *cobra.Command {
	o := NewCheckManifestOptions()

	cmd := &cobra.Command{
		Use:   "check-manifests",
		Short: "Statically compare two manifests for incompatible schemas",
		Run: func(cmd *cobra.Command, args []string) {
			if err := o.Validate(); err != nil {
				klog.Fatal(err)
			}
			config, err := o.Complete()
			if err != nil {
				klog.Fatal(err)
			}
			if _, err := config.Run(); err != nil {
				klog.Fatal(err)
			}
		},
	}

	o.AddFlags(cmd.Flags())

	return cmd
}

func (o *CheckManifestOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.ExistingCRDFile, "existing-crd-filename", o.ExistingCRDFile, "file of existing CRD")
	fs.StringVar(&o.NewCRDFile, "new-crd-filename", o.NewCRDFile, "file of new CRD")
	fs.StringSliceVar(&o.DisabledComparators, "disabled-validators", o.DisabledComparators, "list of comparators that must be disabled")
	fs.StringSliceVar(&o.EnabledComparators, "enabled-validators", o.EnabledComparators, "list of comparators that must be enabled")
}

func (o *CheckManifestOptions) Validate() error {
	if len(o.NewCRDFile) == 0 {
		return fmt.Errorf("--new-crd-filename is required")
	}
	return nil
}

// Complete fills in missing values before command execution.
func (o *CheckManifestOptions) Complete() (*CheckManifestConfig, error) {
	ret := &CheckManifestConfig{
		ComparatorRegistry: o.ComparatorRegistry,
		IOStreams:          o.IOStreams,
	}

	if len(o.ExistingCRDFile) > 0 {
		content, err := os.ReadFile(o.ExistingCRDFile)
		if err != nil {
			return nil, fmt.Errorf("cannot read existing CRD manifest: %w", err)
		}
		crd, err := resourceread.ReadCustomResourceDefinitionV1(content)
		if err != nil {
			return nil, fmt.Errorf("cannot decode CRD manifest: %w", err)
		}
		ret.ExistingCRD = crd
	}

	content, err := os.ReadFile(o.NewCRDFile)
	if err != nil {
		return nil, fmt.Errorf("cannot read new CRD manifest: %w", err)
	}
	crd, err := resourceread.ReadCustomResourceDefinitionV1(content)
	if err != nil {
		return nil, fmt.Errorf("cannot decode CRD manifest: %w", err)
	}
	ret.NewCRD = crd

	knownComparators := sets.NewString(o.KnownComparators...)
	disabledComparators := sets.NewString(o.DisabledComparators...)
	enabledComparators := sets.NewString(o.EnabledComparators...)

	if diff := disabledComparators.Difference(knownComparators); len(diff) > 0 {
		return nil, fmt.Errorf("unknown comparators: %v", disabledComparators.List())
	}
	if diff := enabledComparators.Difference(knownComparators); len(diff) > 0 {
		return nil, fmt.Errorf("unknown comparators: %v", disabledComparators.List())
	}

	comparatorsToRun := sets.NewString(o.DefaultEnabledComparators...).Insert(o.EnabledComparators...).Delete(o.DisabledComparators...)
	ret.ComparatorNames = comparatorsToRun.List()

	return ret, nil
}

type CheckManifestConfig struct {
	ExistingCRD *apiextensionsv1.CustomResourceDefinition
	NewCRD      *apiextensionsv1.CustomResourceDefinition

	ComparatorRegistry manifestcomparators.CRDComparatorRegistry
	ComparatorNames    []string

	IOStreams genericclioptions.IOStreams
}

// Run contains the logic of the render command.
func (c *CheckManifestConfig) Run() ([]manifestcomparators.ComparisonResults, error) {

	comparisonResults, errs := c.ComparatorRegistry.Compare(c.ExistingCRD, c.NewCRD, c.ComparatorNames...)
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Fprintf(c.IOStreams.ErrOut, "Error during evalutions: %v\n", err)
		}
	}
	for _, comparisonResult := range comparisonResults {
		for _, msg := range comparisonResult.Errors {
			fmt.Fprintf(c.IOStreams.ErrOut, "ERROR: %q: %v\n", comparisonResult.Name, msg)
		}
	}
	for _, comparisonResult := range comparisonResults {
		for _, msg := range comparisonResult.Warnings {
			fmt.Fprintf(c.IOStreams.Out, "Warning: %q: %v\n", comparisonResult.Name, msg)
		}
	}
	for _, comparisonResult := range comparisonResults {
		for _, msg := range comparisonResult.Infos {
			fmt.Fprintf(c.IOStreams.Out, "info: %q: %v\n", comparisonResult.Name, msg)
		}
	}

	return comparisonResults, nil
}
