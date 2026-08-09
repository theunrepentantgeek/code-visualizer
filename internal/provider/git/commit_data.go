package git

import (
	"time"

	"github.com/go-git/go-git/v5/plumbing/object"
)

// commitData holds all per-file commit information collected in a single git log pass.
type commitData struct {
	oldest       time.Time
	newest       time.Time
	count        int64
	authors      map[string]bool
	linesAdded   int64
	linesRemoved int64
	hasLineStats bool
}

func (data *commitData) updateFrom(
	c *object.Commit,
	change *object.Change,
	needsLineStats bool,
) {
	when := c.Author.When

	if data.oldest.IsZero() || when.Before(data.oldest) {
		data.oldest = when
	}

	if data.newest.IsZero() || when.After(data.newest) {
		data.newest = when
	}

	data.authors[c.Author.Email] = true
	data.count++

	if !needsLineStats || change == nil || change.From.Name == "" {
		return
	}

	patch, err := object.Changes{change}.Patch()
	if err != nil {
		return
	}

	for _, stat := range patch.Stats() {
		data.linesAdded += int64(stat.Addition)
		data.linesRemoved += int64(stat.Deletion)
	}
}
