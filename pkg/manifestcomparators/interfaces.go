package manifestcomparators

import apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

type ComparisonResults struct {
	Name         string `yaml:"name"`
	WhyItMatters string `yaml:"whyItMatters"`

	Errors   []string `yaml:"errors"`
	Warnings []string `yaml:"warnings"`
	Infos    []string `yaml:"infos"`
}

type CRDComparator interface {
	Name() string
	Labels() []string
	WhyItMatters() string
	Compare(existingCRD, newCRD *apiextensionsv1.CustomResourceDefinition) (ComparisonResults, error)
}

type SingleCRDValidator interface {
	Validate(crd *apiextensionsv1.CustomResourceDefinition) (ComparisonResults, error)
}

type CRDComparatorRegistry interface {
	AddComparator(comparator CRDComparator) error
	GetComparator(name string) (CRDComparator, error)

	KnownComparators() []string
	ComparatorsMatchingLabels(labels []string) []string
	AllComparators() []CRDComparator

	Compare(existingCRD, newCRD *apiextensionsv1.CustomResourceDefinition, names ...string) ([]ComparisonResults, []error)
}

type LabelEnum int

const (
	BackwardsCompatibility LabelEnum = iota
	DataType
	Style
)

func (l LabelEnum) String() string {
	switch l {
	case BackwardsCompatibility:
		return "BackwardsCompatibility"
	case DataType:
		return "DataType"
	case Style:
		return "Style"
	default:
		return "Unknown"
	}
}
