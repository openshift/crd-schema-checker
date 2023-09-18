package manifestcomparators

import "testing"

func TestListsMustHaveSSATags(t *testing.T) {
	RunAllTestsInDirForComparator(t, ListsMustHaveSSATags(), "testdata/lists_must_have_ssa_tags")
}
