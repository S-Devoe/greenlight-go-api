package validator

import (
	"fmt"
	"regexp"
)

var EmailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

type Validator struct {
	// Errors map[string]string
	Errors []string
}

func New() *Validator {
	// return &Validator{Errors: make(map[string]string)}
	return &Validator{Errors: make([]string, 0)}
}

func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

// this stores the errors in a map (object with unique key in Javascript terminology) format
func (v *Validator) AddError(key, message string) {
	// if _, exists := v.Errors[key]; !exists {
	// v.Errors[key] = message
	v.Errors = append(v.Errors, fmt.Sprintf("'%s':%s", key, message))
	// }
}

func (v *Validator) Check(ok bool, key, message string) {
	if !ok {
		v.AddError(key, message)
	}

}

func In(value string, list ...string) bool {
	for i := range list {
		if value == list[i] {
			return true
		}
	}
	return false
}

func Macthes(value string, rx *regexp.Regexp) bool {
	return rx.MatchString(value)
}

func Unique(values []string) bool {
	uniqueValues := make(map[string]bool)
	for _, value := range values {
		uniqueValues[value] = true
	}
	return len(values) == len(uniqueValues)
}
