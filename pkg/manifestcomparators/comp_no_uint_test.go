package manifestcomparators

import "testing"

func TestNoUints(t *testing.T) {
	RunAllTestsInDirForComparator(t, NoUints(), "testdata/no_uints")
}
