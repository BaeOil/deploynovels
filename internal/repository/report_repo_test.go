package repository

import "testing"

func TestReportResolutionSuspendsOnlyPublishedNovel(t *testing.T) {
	if !shouldSuspendNovelForReport("published", true) {
		t.Fatal("expected published novel to be suspended on first resolved report")
	}
	if shouldSuspendNovelForReport("suspended", false) {
		t.Fatal("did not expect already suspended novel to be suspended again")
	}
	if !canResolvePendingReport("suspended", false) {
		t.Fatal("expected pending report on suspended novel to remain resolvable")
	}
	if canResolvePendingReport("draft", false) {
		t.Fatal("did not expect pending report on draft novel to be approved")
	}
}

func TestMultipleReportsOnSameNovelRemainResolvableAfterSuspension(t *testing.T) {
	// Report A resolves first from a published novel; that transition is allowed.
	if !shouldSuspendNovelForReport("published", true) {
		t.Fatal("expected first report on published novel to trigger suspension")
	}

	// After suspension, a later pending report B should not be blocked by the novel being suspended.
	if shouldSuspendNovelForReport("suspended", false) {
		t.Fatal("did not expect second report to suspend an already suspended novel again")
	}
	if !canResolvePendingReport("suspended", false) {
		t.Fatal("expected remaining pending report on the same novel to stay resolvable after suspension")
	}
}
