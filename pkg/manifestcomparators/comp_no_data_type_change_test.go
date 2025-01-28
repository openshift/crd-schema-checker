package manifestcomparators

import "testing"

func TestNoDataTypeChange(t *testing.T) {
	RunAllTestsInDirForComparator(t, NoDataTypeChange(), "testdata/no_data_type_change")
}
