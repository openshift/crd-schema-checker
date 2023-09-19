package manifestcomparators

import "testing"

func TestNoNewRequiredFields(t *testing.T) {
	RunAllTestsInDirForComparator(t, NoNewRequiredFields(), "testdata/no_new_required_fields")
}
