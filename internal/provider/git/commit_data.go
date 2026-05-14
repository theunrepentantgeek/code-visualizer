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
}

func (data *commitData) updateFrom(
	c *object.Commit,
	relPath string,
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

	if c.NumParents() > 0 {
		added, removed := computeFileDiffStats(c, relPath)
		data.linesAdded += added
		data.linesRemoved += removed
	}
}
