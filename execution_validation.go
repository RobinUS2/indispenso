package main

// Validates the execution of a process

type ExecutionValidation struct {
	Fatal       bool   // If matched, should we abort the (sequence of) operation(s)?
	MustContain bool   // Should this be in there?
	Text        string // Text to match
}

// Must contain XYZ
func newExecutionValidationMustContain(txt string) *ExecutionValidation {
	return &ExecutionValidation{
		Fatal:       true,
		MustContain: true,
		Text:        txt,
	}
}
