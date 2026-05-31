package git

import (
	"errors"
	"time"
)

var errUntracked = errors.New("file has no git history")

const monthHours = 24 * 30.44

func (s *repoService) quantityMetric(
	relPath string,
	tracked func(*commitData) bool,
	value func(*commitData) int64,
) (int64, error) {
	data, err := s.getCommitData(relPath)
	if err != nil {
		return 0, err
	}

	if !tracked(data) {
		return 0, errUntracked
	}

	return value(data), nil
}

func (s *repoService) measureMetric(
	relPath string,
	tracked func(*commitData) bool,
	value func(*commitData) float64,
) (float64, error) {
	data, err := s.getCommitData(relPath)
	if err != nil {
		return 0, err
	}

	if !tracked(data) {
		return 0, errUntracked
	}

	return value(data), nil
}

func (s *repoService) fileAge(relPath string) (int64, error) {
	return s.quantityMetric(relPath, func(data *commitData) bool {
		return !data.oldest.IsZero()
	}, func(data *commitData) int64 {
		age := time.Since(data.oldest)

		return int64(age.Hours() / 24)
	})
}

func (s *repoService) fileFreshness(relPath string) (int64, error) {
	return s.quantityMetric(relPath, func(data *commitData) bool {
		return !data.newest.IsZero()
	}, func(data *commitData) int64 {
		freshness := time.Since(data.newest)

		return int64(freshness.Hours() / 24)
	})
}

func (s *repoService) authorCount(relPath string) (int64, error) {
	return s.quantityMetric(relPath, func(data *commitData) bool {
		return len(data.authors) > 0
	}, func(data *commitData) int64 {
		return int64(len(data.authors))
	})
}

func (s *repoService) commitCount(relPath string) (int64, error) {
	return s.quantityMetric(relPath, func(data *commitData) bool {
		return data.count > 0
	}, func(data *commitData) int64 {
		return data.count
	})
}

func (s *repoService) totalLinesAdded(relPath string) (int64, error) {
	return s.quantityMetric(relPath, func(data *commitData) bool {
		return data.count > 0
	}, func(data *commitData) int64 {
		return data.linesAdded
	})
}

func (s *repoService) totalLinesRemoved(relPath string) (int64, error) {
	return s.quantityMetric(relPath, func(data *commitData) bool {
		return data.count > 0
	}, func(data *commitData) int64 {
		return data.linesRemoved
	})
}

func (s *repoService) commitDensity(relPath string) (float64, error) {
	return s.measureMetric(relPath, func(data *commitData) bool {
		return data.count > 0
	}, func(data *commitData) float64 {
		fileAgeMonths := time.Since(data.oldest).Hours() / monthHours
		if fileAgeMonths < 1 {
			fileAgeMonths = 1
		}

		return float64(data.count) / fileAgeMonths
	})
}
