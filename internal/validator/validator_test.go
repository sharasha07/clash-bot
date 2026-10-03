package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type err struct {
	key   string
	value string
}

func TestNew(t *testing.T) {
	v := New()

	require.NotNil(t, v)
	assert.Empty(t, v.Errors)
	assert.True(t, v.Valid())
}

func TestValid(t *testing.T) {
	tests := []struct {
		name      string
		errors    map[string]string
		wantValid bool
	}{
		{
			name:      "no errors",
			errors:    map[string]string{},
			wantValid: true,
		},
		{
			name:      "single error",
			errors:    map[string]string{"username": "must not be empty"},
			wantValid: false,
		},
		{
			name: "multiple errors",
			errors: map[string]string{
				"username": "must not be empty",
				"password": "must be more than 8 characters",
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &Validator{Errors: tt.errors}

			assert.Equal(t, tt.wantValid, v.Valid())
		})
	}
}

func TestAdd(t *testing.T) {
	tests := []struct {
		name       string
		adds       []err
		wantErrors map[string]string
	}{
		{
			name:       "add error",
			adds:       []err{{"username", "must not be empty"}},
			wantErrors: map[string]string{"username": "must not be empty"},
		},
		{
			name: "add errors for different keys",
			adds: []err{
				{"username", "must not be empty"},
				{"game_tag", "must start with #"},
			},
			wantErrors: map[string]string{
				"username": "must not be empty",
				"game_tag": "must start with #",
			},
		},
		{
			name: "first value for duplicate key",
			adds: []err{
				{"page", "must be greater than zero"},
				{"page", "must be a maximum of 100"},
			},
			wantErrors: map[string]string{"page": "must be greater than zero"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()

			for _, a := range tt.adds {
				v.Add(a.key, a.value)
			}

			assert.Equal(t, tt.wantErrors, v.Errors)
		})
	}
}

func TestCheck(t *testing.T) {
	tests := []struct {
		name       string
		initial    map[string]string
		condition  bool
		key        string
		value      string
		wantErrors map[string]string
	}{
		{
			name:       "true condition - add nothing",
			initial:    map[string]string{},
			condition:  true,
			key:        "page",
			value:      "must be greater than zero",
			wantErrors: map[string]string{},
		},
		{
			name:       "false condition - add error",
			initial:    map[string]string{},
			condition:  false,
			key:        "page",
			value:      "must be greater than zero",
			wantErrors: map[string]string{"page": "must be greater than zero"},
		},
		{
			name:       "false condition - keep existing error",
			initial:    map[string]string{"page": "must be greater than zero"},
			condition:  false,
			key:        "page",
			value:      "must be a maximum of 100",
			wantErrors: map[string]string{"page": "must be greater than zero"},
		},
		{
			name:      "false condition - add another error",
			initial:   map[string]string{"page": "must be greater than zero"},
			condition: false,
			key:       "page_size",
			value:     "must be a maximum of 10",
			wantErrors: map[string]string{
				"page":      "must be greater than zero",
				"page_size": "must be a maximum of 10",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()

			for key, value := range tt.initial {
				v.Add(key, value)
			}

			v.Check(tt.condition, tt.key, tt.value)

			assert.Equal(t, tt.wantErrors, v.Errors)
			assert.Equal(t, len(tt.wantErrors) == 0, v.Valid())
		})
	}
}
