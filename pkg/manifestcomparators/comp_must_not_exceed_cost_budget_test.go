package manifestcomparators

import "testing"

func TestMustNotExceedCostBudget(t *testing.T) {
	RunAllTestsInDirForComparator(t, MustNotExceedCostBudget(), "testdata/must_not_exceed_cost_budget")
}
