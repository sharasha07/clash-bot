package validator

import "maps"

type Validator struct {
	errors map[string]string
}

func New() *Validator {
	return &Validator{
		errors: make(map[string]string),
	}
}

func (v *Validator) Valid() bool {
	return len(v.errors) == 0
}

func (v *Validator) Errors() map[string]string {
	return maps.Clone(v.errors)
}

func (v *Validator) AddError(key, value string) {
	if _, ok := v.errors[key]; !ok {
		v.errors[key] = value
	}
}

func (v *Validator) Check(condition bool, key, value string) {
	if !condition {
		v.AddError(key, value)
	}
}
