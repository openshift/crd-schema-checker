package checkadmission

import (
	"github.com/openshift/crd-schema-checker/pkg/admissionevaluator"
	"github.com/openshift/crd-schema-checker/pkg/cmd/options"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/openshift/generic-admission-server/pkg/cmd/server"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type AdmissionCheckOptions struct {
	AdmissionServerOptions *server.AdmissionServerOptions

	ComparatorOptions *options.ComparatorOptions

	admissionHook *admissionevaluator.AdmissionHook
}

func NewAdmissionCheckOptions(streams genericclioptions.IOStreams) *AdmissionCheckOptions {
	o := &AdmissionCheckOptions{
		ComparatorOptions: options.NewComparatorOptions(),
		admissionHook:     &admissionevaluator.AdmissionHook{},
	}
	o.AdmissionServerOptions = server.NewAdmissionServerOptions(streams.Out, streams.ErrOut, o.admissionHook)

	return o
}

// NewCommandStartMaster provides a CLI handler for 'start master' command
func NewCommandStartAdmissionServer(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewAdmissionCheckOptions(streams)

	// TODO fix.
	forever := make(chan struct{})

	cmd := &cobra.Command{
		Use:   "admission-check",
		Short: "Check CRDs for compatibility and potentially other things.",
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(); err != nil {
				return err
			}
			if err := o.Validate(args); err != nil {
				return err
			}
			if err := o.RunAdmissionServer(forever); err != nil {
				return err
			}
			return nil
		},
	}

	flags := cmd.Flags()
	o.AddFlags(flags)

	return cmd
}

func (o *AdmissionCheckOptions) AddFlags(fs *pflag.FlagSet) {
	o.AdmissionServerOptions.RecommendedOptions.AddFlags(fs)
	o.ComparatorOptions.AddFlags(fs)
}

func (o *AdmissionCheckOptions) Complete() error {
	comparatorConfig, err := o.ComparatorOptions.Complete()
	if err != nil {
		return err
	}
	o.admissionHook.ComparatorConfig = comparatorConfig

	return o.AdmissionServerOptions.Complete()
}

func (o *AdmissionCheckOptions) Validate(args []string) error {
	if err := o.ComparatorOptions.Validate(); err != nil {
		return err
	}

	return o.AdmissionServerOptions.Validate(args)
}

func (o AdmissionCheckOptions) RunAdmissionServer(stopCh <-chan struct{}) error {
	if err := o.AdmissionServerOptions.RunAdmissionServer(stopCh); err != nil {
		return err
	}

	return nil
}
