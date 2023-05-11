package manifestcomparators

import "testing"

func TestNoBools(t *testing.T) {
	RunAllTestsInDirForComparator(t, NoBools(), "testdata/no_bools")
}
