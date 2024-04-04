package manifestcomparators

import "testing"

func TestNoEnumRemoval(t *testing.T) {
	RunAllTestsInDirForComparator(t, NoEnumRemoval(), "testdata/no_enum_removal")
}
