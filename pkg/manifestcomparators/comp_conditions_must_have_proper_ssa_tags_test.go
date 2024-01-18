package manifestcomparators

import "testing"

func TestConditionsMustHaveProperSSATags(t *testing.T) {
	RunAllTestsInDirForComparators(t, []CRDComparator{ConditionsMustHaveProperSSATags(), ListsMustHaveSSATags()}, "testdata/conditions_must_have_proper_ssa_tags")
}
