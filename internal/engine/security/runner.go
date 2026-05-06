package security

import (
	"sort"
	"sync"

	"github.com/hamza-hafeez82/cortex/pkg/detector"
	"github.com/hamza-hafeez82/cortex/pkg/detector/detectors"
)

// Runner executes all security detectors concurrently and collects results.
type Runner struct {
	detectors []detector.Detector
}

// NewRunner creates a security Runner pre-loaded with all built-in detectors.
func NewRunner() *Runner {
	return &Runner{
		detectors: []detector.Detector{
			&detectors.HardcodedSecretsDetector{},
			&detectors.SQLInjectionDetector{},
			&detectors.CommandInjectionDetector{},
			&detectors.PathTraversalDetector{},
			&detectors.InsecureRandomDetector{},
			&detectors.JWTMisconfigDetector{},
			&detectors.CORSMisconfigDetector{},
			&detectors.SensitiveDataInLogsDetector{},
			&detectors.VulnerableDepsDetector{},
		},
	}
}

// Run executes all detectors in parallel and returns merged, sorted results.
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
		si := severityOrder(all[i].Severity)
		sj := severityOrder(all[j].Severity)
		if si != sj {
			return si < sj
		}
		if all[i].File != all[j].File {
			return all[i].File < all[j].File
		}
		return all[i].Line < all[j].Line
	})

	return all
}

func severityOrder(s string) int {
	switch s {
	case detector.SeverityCritical:
		return 0
	case detector.SeverityHigh:
		return 1
	case detector.SeverityMedium:
		return 2
	case detector.SeverityLow:
		return 3
	default:
		return 4
	}
}
