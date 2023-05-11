package manifestcomparators

import "testing"

func TestNoFieldRemoval(t *testing.T) {
	RunAllTestsInDirForComparator(t, NoFieldRemoval(), "testdata/no_field_removal")
}
