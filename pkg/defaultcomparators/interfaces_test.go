package defaultcomparators

import (
	"testing"

	"github.com/deads2k/crd-schema-compatibility-checker/pkg/manifestcomparators"
)

func TestRegistry(t *testing.T) {
	manifestcomparators.RunAllTestsInDirForRegistry(t, NewDefaultComparators(), "../manifestcomparators/testdata")
}
