package architecture

import (
	"sort"
	"sync"

	"github.com/hamza-hafeez82/cortex/pkg/detector"
	"github.com/hamza-hafeez82/cortex/pkg/detector/detectors"
)

// Runner executes all architecture detectors concurrently.
type Runner struct {
	detectors []detector.Detector
}

// NewRunner creates an architecture Runner with all built-in detectors.
func NewRunner() *Runner {
	return &Runner{
		detectors: []detector.Detector{
			&detectors.GodFileDetector{},
			&detectors.CircularDepsDetector{},
			&detectors.DeepNestingDetector{},
			&detectors.MissingErrorHandlingDetector{},
			&detectors.MagicNumbersDetector{},
			&detectors.DeadCodeDetector{},
			&detectors.MissingTestsDetector{},
			&detectors.NamingConventionsDetector{},
		},
	}
}

// Run executes all architecture detectors in parallel and returns sorted results.
func (r *Runner) Run(ctx *detector.ScanContext) []detector.Issue {
	results := make(chan []detector.Issue, len(r.detectors))
	var wg sync.WaitGroup

	for _, d := range r.detectors {
		wg.Add(1)
		go func(d detector.Detector) {
			defer wg.Done()
			results <- d.Run(ctx)
		}(d)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var all []detector.Issue
	for issues := range results {
		all = append(all, issues...)
	}

	sort.Slice(all, func(i, j int) bool {
		if all[i].File != all[j].File {
			return all[i].File < all[j].File
		}
		return all[i].Line < all[j].Line
	})

	return all
}
