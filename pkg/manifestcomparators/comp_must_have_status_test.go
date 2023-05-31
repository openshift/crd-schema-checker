package manifestcomparators

import "testing"

func TestMustHaveStatus(t *testing.T) {
	RunAllTestsInDirForComparator(t, MustHaveStatus(), "testdata/must_have_status")
}
