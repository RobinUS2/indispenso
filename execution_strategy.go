package main

// @author Robin Verlangen
// The execution stratey of a command

type ExecutionStrategyType int

type ExecutionStrategy struct {
	Strategy ExecutionStrategyType
}

const (
	SimpleExecutionStrategy ExecutionStrategyType = iota
	OneTestExecutionStrategy
	RollingUpgradeExecutionStrategy
	ExponentialRollingUpgradeExecutionStrategy
)

func newExecutionStrategy(strategy ExecutionStrategyType) *ExecutionStrategy {
	return &ExecutionStrategy{
		Strategy: strategy,
	}
}
