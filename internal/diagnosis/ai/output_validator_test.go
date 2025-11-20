package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOutputValidator_Validate(t *testing.T) {
	validator := NewOutputValidator()

	t.Run("Valid Repair Plan", func(t *testing.T) {
		plan := &RepairPlan{
			Steps: []RepairStep{
				{ID: 1, Description: "Step 1"},
				{ID: 2, Description: "Step 2", DependsOn: []int{1}},
			},
			RollbackPlan: "Rollback",
		}
		err := validator.Validate(plan)
		assert.NoError(t, err)
	})

	t.Run("Invalid Repair Plan - Duplicate ID", func(t *testing.T) {
		plan := &RepairPlan{
			Steps: []RepairStep{
				{ID: 1, Description: "Step 1"},
				{ID: 1, Description: "Step 2"},
			},
			RollbackPlan: "Rollback",
		}
		err := validator.Validate(plan)
		assert.Error(t, err)
	})

	t.Run("Invalid Repair Plan - Circular Dependency", func(t *testing.T) {
		plan := &RepairPlan{
			Steps: []RepairStep{
				{ID: 1, Description: "Step 1", DependsOn: []int{2}},
				{ID: 2, Description: "Step 2", DependsOn: []int{1}},
			},
			RollbackPlan: "Rollback",
		}
		err := validator.Validate(plan)
		assert.Error(t, err)
	})
}
