package checkmanifests

import (
	"fmt"
	"os"

	"github.com/openshift/crd-schema-checker/pkg/cmd/options"

	"github.com/openshift/crd-schema-checker/pkg/manifestcomparators"
	"github.com/openshift/crd-schema-checker/pkg/resourceread"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
)

type CheckManifestOptions struct {
	ExistingCRDFile string
	NewCRDFile      string

	ComparatorOptions *options.ComparatorOptions

	IOStreams genericclioptions.IOStreams
}

func NewCheckManifestOptions(streams genericclioptions.IOStreams) *CheckManifestOptions {
	o := &CheckManifestOptions{
		ComparatorOptions: options.NewComparatorOptions(),
		IOStreams:         streams,
	}

	return o
}

// NewRenderCommand creates a render command.
func NewCheckManifestsCommand(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewCheckManifestOptions(streams)

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
			if _, failed, _ := config.Run(); failed {
				// errors are reported by .Run so we just need to exit non-zero
				os.Exit(1)
			}
		},
	}

	o.AddFlags(cmd.Flags())

	return cmd
}

func (o *CheckManifestOptions) AddFlags(fs *pflag.FlagSet) {
	o.ComparatorOptions.AddFlags(fs)
	fs.StringVar(&o.ExistingCRDFile, "existing-crd-filename", o.ExistingCRDFile, "file of existing CRD")
	fs.StringVar(&o.NewCRDFile, "new-crd-filename", o.NewCRDFile, "file of new CRD")
}

func (o *CheckManifestOptions) Validate() error {
	if len(o.NewCRDFile) == 0 {
		return fmt.Errorf("--new-crd-filename is required")
	}
	if err := o.ComparatorOptions.Validate(); err != nil {
		return err
	}
	return nil
}

// Complete fills in missing values before command execution.
func (o *CheckManifestOptions) Complete() (*CheckManifestConfig, error) {
	ret := &CheckManifestConfig{
		IOStreams: o.IOStreams,
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

	comparatorConfig, err := o.ComparatorOptions.Complete()
	if err != nil {
		return nil, err
	}
	ret.ComparatorConfig = comparatorConfig

	return ret, nil
}

type CheckManifestConfig struct {
	ExistingCRD *apiextensionsv1.CustomResourceDefinition
	NewCRD      *apiextensionsv1.CustomResourceDefinition

	ComparatorConfig *options.ComparatorConfig

	IOStreams genericclioptions.IOStreams
}

// Run contains the logic of the render command.
func (c *CheckManifestConfig) Run() ([]manifestcomparators.ComparisonResults, bool, error) {
	failed := false

	comparisonResults, errs := c.ComparatorConfig.ComparatorRegistry.Compare(c.ExistingCRD, c.NewCRD, c.ComparatorConfig.ComparatorNames...)
	if len(errs) > 0 {
		failed = true
		for _, err := range errs {
			fmt.Fprintf(c.IOStreams.ErrOut, "Error during evalutions: %v\n", err)
		}
	}
	for _, comparisonResult := range comparisonResults {
		for _, msg := range comparisonResult.Errors {
			failed = true
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

	return comparisonResults, failed, utilerrors.NewAggregate(errs)
}
