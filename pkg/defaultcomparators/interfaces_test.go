package defaultcomparators

import (
	"testing"

	"github.com/openshift/crd-schema-checker/pkg/manifestcomparators"
)

func TestRegistry(t *testing.T) {
	manifestcomparators.RunAllTestsInDirForRegistry(t, NewDefaultComparators(), "../manifestcomparators/testdata")
}
