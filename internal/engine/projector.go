package engine

// Projector computes GameState from the Event sequence
type Projector struct{}

// NewProjector creates a standard projector.
func NewProjector() *Projector {
	return &Projector{}
}

// Build folds the standard apply functions.
func (p *Projector) Build(events []Event) (*GameState, error) {
	state := NewGameState()

	for _, evt := range events {
		if err := evt.Apply(state); err != nil {
			return nil, err
		}
	}

	return state, nil
}
