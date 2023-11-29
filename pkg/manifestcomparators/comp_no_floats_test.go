package manifestcomparators

import "testing"

func TestNoFloats(t *testing.T) {
	RunAllTestsInDirForComparator(t, NoFloats(), "testdata/no_floats")
}
