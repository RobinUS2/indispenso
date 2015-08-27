package main

// Validates the execution of a process

type ExecutionValidation struct {
	Fatal        bool   // If matched, should we abort the (sequence of) operation(s)?
	MustContain  bool   // Should this be in there?
	OutputStream int    // 1 = standard output, 2 error output
	Text         string // Text to match
}

// Must contain XYZ
func newExecutionValidationStandardOutputMustContain(txt string) *ExecutionValidation {
	return &ExecutionValidation{
		Fatal:        true,
		MustContain:  true,
		Text:         txt,
		OutputStream: 1,
	}
}
