package app

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
)

func (a *App) Optimize(ctx context.Context, in OptimizeInput) (OptimizeOutput, error) {
	if a == nil || a.provider == nil {
		return OptimizeOutput{}, fmt.Errorf("%w: provider is not configured", ErrProvider)
	}
	if len(in.Zones) == 0 {
		return OptimizeOutput{}, fmt.Errorf("%w: zones is required", ErrInput)
	}
	if in.Duration <= 0 {
		return OptimizeOutput{}, fmt.Errorf("%w: duration must be > 0", ErrInput)
	}
	if in.Lookahead <= 0 {
		return OptimizeOutput{}, fmt.Errorf("%w: lookahead must be > 0", ErrInput)
	}
	if in.Duration > in.Lookahead*3600 {
		return OptimizeOutput{}, fmt.Errorf("%w: duration %ds exceeds forecast coverage %ds", ErrInput, in.Duration, in.Lookahead*3600)
	}
	if in.Timeout <= 0 {
		return OptimizeOutput{}, fmt.Errorf("%w: timeout must be > 0", ErrInput)
	}
	model, err := normalizeModel(in.Model)
	if err != nil {
		return OptimizeOutput{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, in.Timeout)
	defer cancel()

	type zoneOutcome struct {
		result ZoneResult
		zone   string
		err    error
	}

	outcomeCh := make(chan zoneOutcome, len(in.Zones))
	var wg sync.WaitGroup

	for _, zone := range in.Zones {
		zone := zone
		wg.Add(1)
		go func() {
			defer wg.Done()

			analysis, err := a.AnalyzeBestWindow(ctx, zone, in.Duration, in.Lookahead, model)
			if err != nil {
				outcomeCh <- zoneOutcome{zone: zone, err: err}
				return
			}

			outcomeCh <- zoneOutcome{
				zone: zone,
				result: ZoneResult{
					Zone:      zone,
					Emission:  analysis.BestEmission,
					BestStart: analysis.BestStart.UTC(),
					BestEnd:   analysis.BestEnd.UTC(),
				},
			}
		}()
	}

	wg.Wait()
	close(outcomeCh)

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return OptimizeOutput{}, fmt.Errorf("%w: operation timed out", ErrTimeout)
	}

	results := make([]ZoneResult, 0, len(in.Zones))
	failures := make(map[string]string)
	hadTimeout := false
	hadNoValidWindow := false
	hadProviderError := false
	for outcome := range outcomeCh {
		if outcome.err != nil {
			failures[outcome.zone] = outcome.err.Error()
			switch {
			case errors.Is(outcome.err, ErrTimeout), errors.Is(outcome.err, context.DeadlineExceeded):
				hadTimeout = true
			case errors.Is(outcome.err, ErrNoValidWindow):
				hadNoValidWindow = true
			default:
				hadProviderError = true
			}
			continue
		}
		results = append(results, outcome.result)
	}

	if len(results) == 0 {
		switch {
		case hadTimeout:
			return OptimizeOutput{}, fmt.Errorf("%w: operation timed out", ErrTimeout)
		case hadProviderError:
			return OptimizeOutput{}, fmt.Errorf("%w: all zones failed due provider/api errors", ErrProvider)
		case hadNoValidWindow || len(failures) > 0:
			return OptimizeOutput{}, fmt.Errorf("%w: no valid window found", ErrNoValidWindow)
		default:
			return OptimizeOutput{}, fmt.Errorf("%w: no valid window found", ErrNoValidWindow)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Emission < results[j].Emission
	})

	best := results[0]
	worst := results[len(results)-1]
	reduction := 0.0
	if worst.Emission > 0 {
		reduction = (worst.Emission - best.Emission) / worst.Emission * 100
	}

	return OptimizeOutput{
		Results:   results,
		Failures:  failures,
		Best:      best,
		Worst:     worst,
		Reduction: reduction,
	}, nil
}

func FormatZoneFailures(failures map[string]string) []string {
	zones := make([]string, 0, len(failures))
	for zone := range failures {
		zones = append(zones, zone)
	}
	sort.Strings(zones)

	lines := make([]string, 0, len(zones))
	for _, zone := range zones {
		lines = append(lines, fmt.Sprintf("zone %s failed: %s", zone, failures[zone]))
	}
	return lines
}
