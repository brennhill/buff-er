package exercise

import (
	"reflect"
	"testing"
)

func TestConfigExerciseFieldParity(t *testing.T) {
	// Invariant 8: config.Exercise and exercise.ConfigExercise must have identical
	// fields (same names and types). The direct type conversion in GetCatalog depends
	// on this. A mismatch would cause a compile error, but this test makes the coupling
	// explicit and catches it with a clear message.
	exerciseType := reflect.TypeOf(Exercise{})
	configType := reflect.TypeOf(ConfigExercise{})

	if exerciseType.NumField() != configType.NumField() {
		t.Fatalf("Exercise has %d fields, ConfigExercise has %d fields — they must match",
			exerciseType.NumField(), configType.NumField())
	}

	for i := range exerciseType.NumField() {
		ef := exerciseType.Field(i)
		cf := configType.Field(i)

		if ef.Name != cf.Name {
			t.Errorf("field %d: Exercise.%s != ConfigExercise.%s", i, ef.Name, cf.Name)
		}
		if ef.Type != cf.Type {
			t.Errorf("field %d (%s): Exercise type %v != ConfigExercise type %v",
				i, ef.Name, ef.Type, cf.Type)
		}
	}
}
