package ai

import (
	"fmt"
	"github.com/go-playground/validator/v10"
)

type OutputValidator struct {
	validator *validator.Validate
}

func NewOutputValidator() *OutputValidator {
	v := validator.New()
	return &OutputValidator{validator: v}
}

func (v *OutputValidator) Validate(result interface{}) error {
	if err := v.validator.Struct(result); err != nil {
		return fmt.Errorf("basic validation failed: %w", err)
	}

	switch r := result.(type) {
	case *RepairPlan:
		if err := v.validateRepairPlanDependencies(r); err != nil {
			return fmt.Errorf("repair plan dependency validation failed: %w", err)
		}
	}

	return nil
}

func (v *OutputValidator) validateRepairPlanDependencies(plan *RepairPlan) error {
	stepIDs := make(map[int]struct{})
	for _, step := range plan.Steps {
		if _, exists := stepIDs[step.ID]; exists {
			return fmt.Errorf("duplicate step ID found: %d", step.ID)
		}
		stepIDs[step.ID] = struct{}{}
	}

	for _, step := range plan.Steps {
		for _, depID := range step.DependsOn {
			if _, exists := stepIDs[depID]; !exists {
				return fmt.Errorf("step %d depends on a non-existent step %d", step.ID, depID)
			}
		}
	}

	// Check for circular dependencies
	visiting := make(map[int]bool)
	visited := make(map[int]bool)
	var hasCycle func(stepID int) bool

	stepMap := make(map[int]RepairStep)
	for _, step := range plan.Steps {
		stepMap[step.ID] = step
	}

	hasCycle = func(stepID int) bool {
		visiting[stepID] = true

		step := stepMap[stepID]
		for _, depID := range step.DependsOn {
			if visiting[depID] {
				return true // Cycle detected
			}
			if !visited[depID] {
				if hasCycle(depID) {
					return true
				}
			}
		}

		visiting[stepID] = false
		visited[stepID] = true
		return false
	}

	for _, step := range plan.Steps {
		if !visited[step.ID] {
			if hasCycle(step.ID) {
				return fmt.Errorf("circular dependency detected in repair plan involving step %d", step.ID)
			}
		}
	}

	return nil
}
