package app

import "fmt"

const (
	defaultModelRunner = "ubuntu"
	defaultModelLoad   = 0.6
	defaultModelPUE    = 1.2
)

func normalizeModel(model ModelContext) (ModelContext, error) {
	if model == (ModelContext{}) {
		model = ModelContext{
			Runner: defaultModelRunner,
			Load:   defaultModelLoad,
			PUE:    defaultModelPUE,
		}
	}
	if model.Runner == "" {
		model.Runner = defaultModelRunner
	}
	if model.Load < 0 || model.Load > 1 {
		return ModelContext{}, fmt.Errorf("%w: load must be between 0 and 1", ErrInput)
	}
	if model.PUE < 1.0 {
		return ModelContext{}, fmt.Errorf("%w: pue must be >= 1.0", ErrInput)
	}
	return model, nil
}
