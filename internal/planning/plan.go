package planning

import (
	"fmt"
	"time"
)

// NewPlan creates a new plan with the given ID, name, and steps
func NewPlan(id, name string, steps []Step) *Plan {
	return &Plan{
		ID:        id,
		Name:      name,
		Steps:     steps,
		CreatedAt: time.Now(),
		Metadata:  make(map[string]string),
	}
}

// Validate validates the plan structure
func (p *Plan) Validate() error {
	if len(p.Steps) == 0 {
		return fmt.Errorf("plan must have at least one step")
	}

	stepIDs := make(map[string]bool)
	var errors []error

	// Check for duplicate IDs
	for _, step := range p.Steps {
		if step.ID == "" {
			errors = append(errors, fmt.Errorf("step must have an ID"))
			continue
		}
		if stepIDs[step.ID] {
			errors = append(errors, fmt.Errorf("duplicate step ID: %s", step.ID))
		}
		stepIDs[step.ID] = true
	}

	// Check that all dependencies exist
	for _, step := range p.Steps {
		for _, depID := range step.DependsOn {
			if !stepIDs[depID] {
				errors = append(errors, fmt.Errorf("step %s depends on non-existent step: %s", step.ID, depID))
			}
		}
	}

	// Check for cyclic dependencies using DAG
	dag := NewDAG(p.Steps)
	if dag.DetectCycle() {
		errors = append(errors, fmt.Errorf("plan contains cyclic dependencies"))
	}

	if len(errors) > 0 {
		return fmt.Errorf("plan validation failed: %v", errors)
	}

	return nil
}

// GetStep retrieves a step by ID
func (p *Plan) GetStep(id string) (*Step, bool) {
	for i := range p.Steps {
		if p.Steps[i].ID == id {
			return &p.Steps[i], true
		}
	}
	return nil, false
}

// StepCount returns the number of steps in the plan
func (p *Plan) StepCount() int {
	return len(p.Steps)
}

// GetStepsByIDs retrieves multiple steps by their IDs
func (p *Plan) GetStepsByIDs(ids []string) []*Step {
	var steps []*Step
	for _, id := range ids {
		if step, ok := p.GetStep(id); ok {
			steps = append(steps, step)
		}
	}
	return steps
}

// AddStep adds a step to the plan
func (p *Plan) AddStep(step Step) error {
	if _, exists := p.GetStep(step.ID); exists {
		return fmt.Errorf("step with ID %s already exists", step.ID)
	}
	p.Steps = append(p.Steps, step)
	return nil
}

// RemoveStep removes a step from the plan
func (p *Plan) RemoveStep(stepID string) error {
	for i, step := range p.Steps {
		if step.ID == stepID {
			p.Steps = append(p.Steps[:i], p.Steps[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("step %s not found", stepID)
}
