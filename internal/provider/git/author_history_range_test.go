package git

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestBulkAuthorHistoryInHistoryRange_UsesTagSelection(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)

	result, err := BulkAuthorHistoryInHistoryRange(
		fixture.dir,
		map[string]bool{"main.go": true, "feature.go": true},
		false,
		HistoryRange{FromTag: "v1.0", UntilTag: "v2.0"},
		nil,
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.ByFile).To(HaveKey("main.go"))
	g.Expect(result.ByFile).To(HaveKey("feature.go"))
	g.Expect(result.HeadDate.IsZero()).To(BeFalse())
}
