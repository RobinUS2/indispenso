package main

// This will coordinate the execution of strategies
// @author Robin Verlangen

type ExecutionCoordinator struct {
	Active map[string]*ConsensusRequest
}

func newExecutionCoordinator() *ExecutionCoordinator {
	return &ExecutionCoordinator{
		Active: make(map[string]*ConsensusRequest),
	}
}
