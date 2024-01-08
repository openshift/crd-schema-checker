package manifestcomparators

import "testing"

func TestNoFieldTypeChange(t *testing.T) {
	RunAllTestsInDirForComparator(t, NoFieldTypeChange(), "testdata/no_field_type_change")
}
