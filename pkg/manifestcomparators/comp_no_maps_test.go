package manifestcomparators

import "testing"

func TestNoMaps(t *testing.T) {
	RunAllTestsInDirForComparator(t, NoMaps(), "testdata/no_maps")
}
