package catalog

import "testing"

func TestLifecycleTransitions(t *testing.T) {
	for _, tc := range []struct {
		from, to Status
		want     bool
	}{{Created, Uploaded, true}, {Created, Deleting, true}, {Uploaded, Indexing, true}, {Indexing, Ready, true}, {Ready, Reindexing, true}, {Reindexing, Ready, true}, {Ready, Deleting, true}, {Deleting, Deleted, true}, {Created, Ready, false}, {Deleted, Ready, false}} {
		if got := CanTransition(tc.from, tc.to); got != tc.want {
			t.Errorf("%s -> %s = %v", tc.from, tc.to, got)
		}
	}
}
