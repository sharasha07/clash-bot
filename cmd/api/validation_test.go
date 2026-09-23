package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldErrors(t *testing.T) {
	app := newTestApplication(t)

	type input struct {
		Example1 string `validate:"required"`
		Example2 int    `validate:"max=10"`
		Example3 int    `json:"example3" validate:"required,min=5"`
	}

	tests := []struct {
		name    string
		inp     input
		wantMap map[string]string
	}{
		{
			name:    "valid",
			inp:     input{"ex1", 1, 6},
			wantMap: nil,
		},
		{
			name: "required tag violation",
			inp:  input{"", 2, 9},
			wantMap: map[string]string{
				"Example1": "must be provided",
			},
		},
		{
			name: "max tag violation",
			inp:  input{"ex1", 11, 9},
			wantMap: map[string]string{
				"Example2": "must not be more than 10 characters",
			},
		},
		{
			name: "min tag violation",
			inp:  input{"ex1", 9, 2},
			wantMap: map[string]string{
				"example3": "must be at least 5 characters",
			},
		},
		{
			name: "required, max, min tag violations",
			inp:  input{"", 11, 2},
			wantMap: map[string]string{
				"Example1": "must be provided",
				"Example2": "must not be more than 10 characters",
				"example3": "must be at least 5 characters",
			},
		},
		{
			name: "double tag violation for same field",
			inp:  input{"ex1", 9, 0},
			wantMap: map[string]string{
				"example3": "must be provided",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := app.validate.Struct(tt.inp)
			result := app.fieldErrors(err)

			assert.Equal(t, tt.wantMap, result)
		})
	}
}
